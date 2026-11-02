package documentos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, nome_original, tipo, descricao, tamanho, coalesce(processo_id::text,''), coalesce(pessoa_id::text,''), created_at, version, arquivo`

func scan(row pgx.Row) (*Doc, error) {
	var d Doc
	err := row.Scan(&d.ID, &d.Nome, &d.Tipo, &d.Descricao, &d.Tamanho, &d.ProcessoID, &d.PessoaID, &d.CriadoEm, &d.Version, &d.arquivo)
	return &d, err
}

func (s *Service) get(ctx context.Context, id string) (*Doc, error) {
	d, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM documentos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	return d, err
}

// Filtro da lista.
type Filtro struct{ Q, ProcessoID, PessoaID string }

// List devolve documentos não excluídos, mais novos primeiro.
func (s *Service) List(ctx context.Context, f Filtro, limit, offset int) ([]Doc, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM documentos
		WHERE excluido_em IS NULL AND ($1 = '' OR nome_original ILIKE $2 OR descricao ILIKE $2)
		  AND ($3 = '' OR processo_id = $3::uuid) AND ($4 = '' OR pessoa_id = $4::uuid)
		ORDER BY created_at DESC, id LIMIT $5 OFFSET $6`, f.Q, webx.Like(f.Q), f.ProcessoID, f.PessoaID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Doc
	var pids, pes []string
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
		if d.ProcessoID != "" {
			pids = append(pids, d.ProcessoID)
		}
		if d.PessoaID != "" {
			pes = append(pes, d.PessoaID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pm, _ := s.Processos.Resumos(ctx, pids)
	nm, _ := s.Pessoas.Names(ctx, pes)
	for i := range out {
		out[i].ProcessoNumero, out[i].PessoaNome = pm[out[i].ProcessoID], nm[out[i].PessoaID]
	}
	return out, nil
}
