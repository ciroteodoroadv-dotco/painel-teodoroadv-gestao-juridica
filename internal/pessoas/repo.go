package pessoas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/crypt"
	"painelteodoro/internal/platform/webx"
)

const cols = `id, tipo, nome, documento_enc, coalesce(email,''), coalesce(telefone,''), papel,
	retencao_ate, version, created_at, updated_at`

func (s *Service) scan(row pgx.Row) (*Pessoa, error) {
	var p Pessoa
	var enc []byte
	if err := row.Scan(&p.ID, &p.Tipo, &p.Nome, &enc, &p.Email, &p.Telefone, &p.Papel,
		&p.RetencaoAte, &p.Version, &p.CriadoEm, &p.AtualizadoEm); err != nil {
		return nil, err
	}
	p.DocMask = br.MaskDocumento(s.unseal(enc))
	return &p, nil
}

// Get devolve uma pessoa não excluída (documento só mascarado).
func (s *Service) Get(ctx context.Context, id string) (*Pessoa, error) {
	p, err := s.scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	return p, err
}

// List busca por nome (trecho) ou por documento exato (CPF/CNPJ digitado inteiro). Devolve até limit linhas.
func (s *Service) List(ctx context.Context, q string, limit, offset int) ([]Pessoa, error) {
	var idx []byte
	if d := br.Digits(q); len(d) == 11 || len(d) == 14 {
		idx = crypt.BlindIndex(s.FieldKey, purpose, d)
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM pessoas
		WHERE excluido_em IS NULL AND ($1 = '' OR nome ILIKE $2 OR ($3::bytea IS NOT NULL AND documento_idx = $3))
		ORDER BY lower(nome), id LIMIT $4 OFFSET $5`, q, webx.Like(q), idx, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Pessoa
	for rows.Next() {
		p, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// Names devolve id → nome (para outros domínios exibirem partes sem acessar esta tabela).
func (s *Service) Names(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, nome FROM pessoas WHERE id = ANY($1::uuid[])`, ids)
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
	n, err := s.Names(ctx, ids)
	if err != nil {
		return nil, err
	}
	ok := map[string]bool{}
	for id := range n {
		ok[id] = true
	}
	if len(ids) > 0 { // Names inclui excluídas; refina
		rows, err := s.Pool.Query(ctx, `SELECT id FROM pessoas WHERE id = ANY($1::uuid[]) AND excluido_em IS NOT NULL`, ids)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			delete(ok, id)
		}
	}
	return ok, nil
}

// Relatorio reúne os dados de UMA pessoa para atender pedido do titular (LGPD art. 18) e audita a emissão.
type Relatorio struct {
	Pessoa    *Pessoa
	Documento string
	Processos []ProcessoRef
}

// Relatorio emite o relatório "dados desta pessoa".
func (s *Service) Relatorio(ctx context.Context, ac webx.Actor, id string) (*Relatorio, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	doc, err := s.Reveal(ctx, ac, id)
	if err != nil {
		return nil, err
	}
	r := &Relatorio{Pessoa: p, Documento: br.FormatDocumento(doc)}
	if s.Processos != nil {
		if r.Processos, err = s.Processos.ProcessosDaPessoa(ctx, id); err != nil {
			return nil, err
		}
	}
	err = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "pessoa.relatorio_titular",
		TargetType: "pessoa", TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
	return r, err
}

// Buscar devolve opções (id → nome) para seletores de outros domínios.
func (s *Service) Buscar(ctx context.Context, q string, limit int) ([]Opcao, error) {
	rows, err := s.List(ctx, q, limit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(rows))
	for i, p := range rows {
		out[i] = Opcao{Valor: p.ID, Rotulo: p.Nome + " (" + p.Tipo + ")"}
	}
	return out, nil
}
