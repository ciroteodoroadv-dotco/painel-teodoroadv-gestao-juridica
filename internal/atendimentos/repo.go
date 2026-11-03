package atendimentos

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, codigo, tipo, status, descricao, observacoes, data_abertura, coalesce(criado_por_pessoa_id::text,''), version`

func scan(row pgx.Row) (*Atendimento, error) {
	var a Atendimento
	err := row.Scan(&a.ID, &a.Codigo, &a.Tipo, &a.Status, &a.Descricao, &a.Observacoes, &a.DataAbertura, &a.CriadoPorID, &a.Version)
	return &a, err
}

// preencher carrega clientes, partes adversas e o nome de "criado por" de uma lista de atendimentos.
func (s *Service) preencher(ctx context.Context, list []*Atendimento) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	idx := map[string]*Atendimento{}
	for i, a := range list {
		ids[i], idx[a.ID] = a.ID, a
	}
	rows, err := s.Pool.Query(ctx, `SELECT atendimento_id, pessoa_id, papel FROM atendimento_pessoas
		WHERE atendimento_id = ANY($1::uuid[]) ORDER BY atendimento_id, papel, ordem`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	type lig struct{ ate, pessoa, papel string }
	var ligs []lig
	pessoaIDs := []string{}
	for rows.Next() {
		var l lig
		if err := rows.Scan(&l.ate, &l.pessoa, &l.papel); err != nil {
			return err
		}
		ligs = append(ligs, l)
		pessoaIDs = append(pessoaIDs, l.pessoa)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, a := range list {
		if a.CriadoPorID != "" {
			pessoaIDs = append(pessoaIDs, a.CriadoPorID)
		}
	}
	refs, err := s.Pessoas.Refs(ctx, pessoaIDs)
	if err != nil {
		return err
	}
	porID := map[string]Ref{}
	for _, r := range refs {
		porID[r.ID] = r
	}
	for _, l := range ligs {
		a := idx[l.ate]
		if l.papel == "cliente" {
			a.Clientes = append(a.Clientes, porID[l.pessoa])
		} else {
			a.PartesAdversas = append(a.PartesAdversas, porID[l.pessoa])
		}
	}
	for _, a := range list {
		a.CriadoPor = porID[a.CriadoPorID].Nome
	}
	return nil
}

// Get devolve um atendimento não excluído.
func (s *Service) Get(ctx context.Context, id string) (*Atendimento, error) {
	a, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM atendimentos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.preencher(ctx, []*Atendimento{a}); err != nil {
		return nil, err
	}
	return a, nil
}

// Listar devolve os atendimentos em ordem de código (a ata sequencial do B). q filtra sem acento nem maiúsculas por código, clientes,
// partes adversas, objeto e observações (a mesma busca da tela do B).
func (s *Service) Listar(ctx context.Context, q string) ([]*Atendimento, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM atendimentos WHERE excluido_em IS NULL
		ORDER BY codigo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []*Atendimento
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.preencher(ctx, all); err != nil {
		return nil, err
	}
	if q = webx.SemAcento(strings.TrimSpace(q)); q == "" {
		return all, nil
	}
	var out []*Atendimento
	for _, a := range all {
		if strings.Contains(webx.SemAcento(a.Busca()), q) {
			out = append(out, a)
		}
	}
	return out, nil
}

func primeiro(rs []Ref) string {
	if len(rs) == 0 {
		return ""
	}
	return strings.ToUpper(rs[0].Nome)
}

// Opcoes monta o seletor de atendimento das Tarefas: "ATE. 0000001 · NOME DO CLIENTE" (origemOptionsHTML do B).
func (s *Service) Opcoes(ctx context.Context) ([]Opcao, error) {
	l, err := s.Listar(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(l))
	for i, a := range l {
		c := primeiro(a.Clientes)
		if c == "" {
			c = "sem cliente"
		}
		out[i] = Opcao{Valor: a.ID, Rotulo: a.Codigo + " · " + c}
	}
	return out, nil
}

// OpcoesOrigem monta o seletor "Atendimento de origem" do Processo: "ATE. 0000001 · tipo · CLIENTE · dd/mm/aaaa".
func (s *Service) OpcoesOrigem(ctx context.Context) ([]Opcao, error) {
	l, err := s.Listar(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(l))
	for i, a := range l {
		out[i] = Opcao{Valor: a.ID, Rotulo: a.Codigo + " · " + a.Tipo + " · " + primeiro(a.Clientes) + " · " + a.DataBR()}
	}
	return out, nil
}

// Existe informa se o atendimento existe e não está excluído.
func (s *Service) Existe(ctx context.Context, id string) (bool, error) {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM atendimentos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Resumo devolve código e nome do (primeiro) cliente, para outros domínios exibirem o atendimento.
func (s *Service) Resumo(ctx context.Context, id string) (codigo, cliente string, err error) {
	a, err := s.Get(ctx, id)
	if err != nil {
		return "", "", err
	}
	return a.Codigo, primeiro(a.Clientes), nil
}

// Resumos devolve id → {código, cliente} de vários atendimentos (uma leitura só).
func (s *Service) Resumos(ctx context.Context, ids []string) (map[string][2]string, error) {
	out := map[string][2]string{}
	if len(ids) == 0 {
		return out, nil
	}
	l, err := s.Listar(ctx, "")
	if err != nil {
		return nil, err
	}
	quer := map[string]bool{}
	for _, id := range ids {
		quer[id] = true
	}
	for _, a := range l {
		if quer[a.ID] {
			out[a.ID] = [2]string{a.Codigo, primeiro(a.Clientes)}
		}
	}
	return out, nil
}
