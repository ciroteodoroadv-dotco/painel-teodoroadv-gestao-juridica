package pessoas

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/crypt"
	"painelteodoro/internal/platform/webx"
)

const cols = `id, codigo, tipo_pessoa, nome_exibicao, nome_completo, documento_enc, rg_enc, estado_civil, profissao,
	nacionalidade, data_nascimento, razao_social, nome_fantasia, representante_nome, representante_cpf_enc,
	representante_cargo, logradouro, numero, complemento, bairro, cep, cidade, coalesce(uf,''), email, telefone,
	observacoes, version, created_at, updated_at`

// LimiteLista protege a tela: o acordeão do B mostra tudo, sem paginação. Acima disso a tela avisa e a busca vai ao banco.
const LimiteLista = 2000

func scan(row pgx.Row) (*Pessoa, error) {
	var p Pessoa
	err := row.Scan(&p.ID, &p.Codigo, &p.TipoPessoa, &p.Nome, &p.NomeCompleto, &p.encDoc, &p.encRG, &p.EstadoCivil,
		&p.Profissao, &p.Nacionalidade, &p.DataNascimento, &p.RazaoSocial, &p.NomeFantasia, &p.RepresentanteNome,
		&p.encRep, &p.RepresentanteCargo, &p.Logradouro, &p.Numero, &p.Complemento, &p.Bairro, &p.CEP, &p.Cidade,
		&p.UF, &p.Email, &p.Telefone, &p.Observacoes, &p.Version, &p.CriadoEm, &p.AtualizadoEm)
	return &p, err
}

// mostrarDocumento preenche Documento: completo e formatado, ou mascarado.
func (s *Service) mostrarDocumento(p *Pessoa, completo bool) {
	d := s.unseal(purposeDoc, p.encDoc)
	if completo {
		p.Documento = br.FormatDocumento(d)
	} else {
		p.Documento = br.MaskDocumento(d)
	}
}

// Get devolve uma pessoa não excluída, com o documento só mascarado.
func (s *Service) Get(ctx context.Context, id string) (*Pessoa, error) {
	p, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.mostrarDocumento(p, false)
	return p, nil
}

// GetCompleto devolve a pessoa com CPF/CNPJ, RG e CPF do representante em claro e AUDITA o acesso (LGPD).
func (s *Service) GetCompleto(ctx context.Context, ac webx.Actor, id string) (*Pessoa, error) {
	p, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.mostrarDocumento(p, true)
	p.RG = s.unseal(purposeRG, p.encRG)
	p.RepresentanteCPF = s.unseal(purposeRep, p.encRep)
	err = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "pessoa.documento_revelado",
		TargetType: "pessoa", TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
	return p, err
}

// Listar devolve as pessoas da TELA, por nome (até LimiteLista). Com completo=true os documentos saem inteiros
// (quem pode ver documento) e o acesso é auditado uma vez; senão, mascarados. busca (opcional) filtra no banco por
// nome ou CPF/CNPJ completo, para achar quem ficou fora do limite (auditoria 05/11, item 1.4).
func (s *Service) Listar(ctx context.Context, ac webx.Actor, completo bool, busca ...string) ([]Pessoa, error) {
	q := ""
	if len(busca) > 0 {
		q = strings.TrimSpace(busca[0])
	}
	return s.listar(ctx, ac, completo, q, LimiteLista)
}

// ListarTodas é Listar sem o corte da tela (seletores de pessoa).
func (s *Service) ListarTodas(ctx context.Context, ac webx.Actor, completo bool) ([]Pessoa, error) {
	return s.listar(ctx, ac, completo, "", 0)
}

// ContarAtivas devolve quantas pessoas não excluídas existem (para avisar quando a tela está cortada).
func (s *Service) ContarAtivas(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM pessoas WHERE excluido_em IS NULL`).Scan(&n)
	return n, err
}

func (s *Service) listar(ctx context.Context, ac webx.Actor, completo bool, q string, limite int) ([]Pessoa, error) {
	var idx []byte
	if d := br.Digits(q); len(d) == 11 || len(d) == 14 {
		idx = crypt.BlindIndex(s.FieldKey, purposeDoc, d)
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM pessoas WHERE excluido_em IS NULL
		AND ($1 = '' OR nome_exibicao ILIKE $2 OR ($3::bytea IS NOT NULL AND documento_idx = $3))
		ORDER BY lower(nome_exibicao), id LIMIT CASE WHEN $4 > 0 THEN $4 END`, q, webx.Like(q), idx, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Pessoa
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		s.mostrarDocumento(p, completo)
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if completo && len(out) > 0 {
		err = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "pessoa.documentos_listados",
			TargetType: "pessoa", TargetID: "lista", Result: audit.OK, RequestID: ac.RequestID,
			Meta: map[string]string{"quantidade": strconv.Itoa(len(out))}})
	}
	return out, err
}

// Opcoes monta as opções de um <select> de pessoa, com o rótulo do B: "NOME — Física (123.456.789-09)".
// O documento só entra no rótulo se comDocumento (permissão de ver documento).
func (s *Service) Opcoes(ctx context.Context, ac webx.Actor, comDocumento bool) ([]Opcao, error) {
	ps, err := s.ListarTodas(ctx, ac, comDocumento) // todas: o seletor não pode herdar o corte da tela (item 1.4)
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(ps))
	for i, p := range ps {
		r := p.NomeLista() + " — " + p.TipoPessoa
		if comDocumento && p.Documento != "" {
			r += " (" + p.Documento + ")"
		}
		out[i] = Opcao{Valor: p.ID, Rotulo: r}
	}
	return out, nil
}

// Buscar devolve opções (id → nome) para seletores de outros domínios; busca por nome ou por documento exato.
func (s *Service) Buscar(ctx context.Context, q string, limit int) ([]Opcao, error) {
	var idx []byte
	if d := br.Digits(q); len(d) == 11 || len(d) == 14 {
		idx = crypt.BlindIndex(s.FieldKey, purposeDoc, d)
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, nome_exibicao, tipo_pessoa FROM pessoas
		WHERE excluido_em IS NULL AND ($1 = '' OR nome_exibicao ILIKE $2 OR ($3::bytea IS NOT NULL AND documento_idx = $3))
		ORDER BY lower(nome_exibicao), id LIMIT $4`, q, webx.Like(q), idx, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Opcao
	for rows.Next() {
		var id, nome, tipo string
		if err := rows.Scan(&id, &nome, &tipo); err != nil {
			return nil, err
		}
		out = append(out, Opcao{Valor: id, Rotulo: nome + " (" + tipo + ")"})
	}
	return out, rows.Err()
}

// Names devolve id → nome (para outros domínios exibirem partes sem acessar esta tabela).
func (s *Service) Names(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, nome_exibicao FROM pessoas WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, nome string
		if err := rows.Scan(&id, &nome); err != nil {
			return nil, err
		}
		out[id] = nome
	}
	return out, rows.Err()
}

// Exist informa quais ids existem e não estão excluídos.
func (s *Service) Exist(ctx context.Context, ids []string) (map[string]bool, error) {
	ok := map[string]bool{}
	if len(ids) == 0 {
		return ok, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id FROM pessoas WHERE id = ANY($1::uuid[]) AND excluido_em IS NULL`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ok[id] = true
	}
	return ok, rows.Err()
}

// Relatorio reúne os dados de UMA pessoa para atender pedido do titular (LGPD art. 18) e audita a emissão.
type Relatorio struct {
	Pessoa    *Pessoa
	Processos []PapelEmProcesso
}

// Relatorio emite o relatório "dados desta pessoa".
func (s *Service) Relatorio(ctx context.Context, ac webx.Actor, id string) (*Relatorio, error) {
	p, err := s.GetCompleto(ctx, ac, id)
	if err != nil {
		return nil, err
	}
	r := &Relatorio{Pessoa: p}
	if s.Papeis != nil {
		m, err := s.Papeis.PapeisPorPessoa(ctx, []string{id})
		if err != nil {
			return nil, err
		}
		r.Processos = m[id]
	}
	err = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "pessoa.relatorio_titular",
		TargetType: "pessoa", TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
	return r, err
}
