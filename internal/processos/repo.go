package processos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/webx"
)

const cols = `id, numero_cnj, orgao, situacao, coalesce(responsavel_id::text,''), version, created_at, updated_at`

func scan(row pgx.Row) (*Processo, error) {
	var p Processo
	err := row.Scan(&p.ID, &p.Numero, &p.Orgao, &p.Situacao, &p.ResponsavelID, &p.Version, &p.CriadoEm, &p.AtualizadoEm)
	return &p, err
}

// Get devolve o processo com as partes (nomes obtidos pela interface de pessoas).
func (s *Service) Get(ctx context.Context, id string) (*Processo, error) {
	p, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM processos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT pessoa_id, qualidade FROM processo_partes WHERE processo_id=$1 ORDER BY qualidade, pessoa_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var pt Parte
		if err := rows.Scan(&pt.PessoaID, &pt.Qualidade); err != nil {
			return nil, err
		}
		p.Partes = append(p.Partes, pt)
		ids = append(ids, pt.PessoaID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if names, err := s.Pessoas.Names(ctx, ids); err == nil {
		for i := range p.Partes {
			p.Partes[i].Nome = names[p.Partes[i].PessoaID]
		}
	}
	if p.ResponsavelID != "" {
		if m, err := s.Users.Emails(ctx, []string{p.ResponsavelID}); err == nil {
			p.ResponsavelEmail = m[p.ResponsavelID]
		}
	}
	return p, nil
}

// Row é a linha da lista.
type Row struct {
	ID, Numero, Orgao, Situacao, Responsavel string
}

// SituacaoRotulo devolve o rótulo da situação.
func (r Row) SituacaoRotulo() string { return rotulo(Situacoes, r.Situacao) }

// List filtra por número (trecho dos dígitos), órgão (trecho) e situação.
func (s *Service) List(ctx context.Context, q, situacao string, limit, offset int) ([]Row, error) {
	digits := br.Digits(q)
	rows, err := s.Pool.Query(ctx, `SELECT id, numero_cnj, orgao, situacao, coalesce(responsavel_id::text,'') FROM processos
		WHERE excluido_em IS NULL AND ($3 = '' OR situacao = $3)
		  AND ($1 = '' OR orgao ILIKE $2 OR ($4 <> '' AND regexp_replace(numero_cnj,'\D','','g') LIKE '%'||$4||'%'))
		ORDER BY created_at DESC, id LIMIT $5 OFFSET $6`, q, webx.Like(q), situacao, digits, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	var resp []string
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Numero, &r.Orgao, &r.Situacao, &r.Responsavel); err != nil {
			return nil, err
		}
		out = append(out, r)
		if r.Responsavel != "" {
			resp = append(resp, r.Responsavel)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if m, err := s.Users.Emails(ctx, resp); err == nil {
		for i := range out {
			out[i].Responsavel = m[out[i].Responsavel]
		}
	}
	return out, nil
}

// ProcessosDaPessoa atende ao domínio de pessoas (interface pessoas.ProcessosLookup).
func (s *Service) ProcessosDaPessoa(ctx context.Context, pessoaID string) ([]pessoas.ProcessoRef, error) {
	rows, err := s.Pool.Query(ctx, `SELECT p.id, p.numero_cnj, pt.qualidade FROM processo_partes pt
		JOIN processos p ON p.id = pt.processo_id WHERE pt.pessoa_id=$1 AND p.excluido_em IS NULL ORDER BY p.numero_cnj`, pessoaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pessoas.ProcessoRef
	for rows.Next() {
		var r pessoas.ProcessoRef
		if err := rows.Scan(&r.ID, &r.Numero, &r.Qualidade); err != nil {
			return nil, err
		}
		r.Qualidade = rotulo(Qualidades, r.Qualidade)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Resumos devolve id → número, para outros domínios (ex.: tarefas) exibirem o processo.
func (s *Service) Resumos(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, numero_cnj FROM processos WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// Existe informa se o processo existe e não está excluído.
func (s *Service) Existe(ctx context.Context, id string) (bool, error) {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM processos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Buscar devolve opções (id → número) para seletores de outros domínios.
func (s *Service) Buscar(ctx context.Context, q string, limit int) ([]Opcao, error) {
	rows, err := s.List(ctx, q, "", limit, 0)
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(rows))
	for i, r := range rows {
		out[i] = Opcao{Valor: r.ID, Rotulo: r.Numero}
	}
	return out, nil
}
