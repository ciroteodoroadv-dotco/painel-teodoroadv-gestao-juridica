package diligencias

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id::text, codigo, titulo, processo_numero, coalesce(processo_id::text,''), coalesce(cliente_pessoa_id::text,''), local,
	responsavel, status, data_limite, coalesce(to_char(hora,'HH24:MI'),''), version`

func scan(row pgx.Row) (*Diligencia, error) {
	var d Diligencia
	err := row.Scan(&d.ID, &d.Codigo, &d.Titulo, &d.ProcessoNumero, &d.ProcessoID, &d.ClienteID, &d.Local, &d.Responsavel, &d.Status,
		&d.DataLimite, &d.Hora, &d.Version)
	return &d, err
}

// Get devolve a diligência não excluída.
func (s *Service) Get(ctx context.Context, id string) (*Diligencia, error) {
	if !webx.UUID(id) {
		return nil, webx.ErrNotFound
	}
	d, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM diligencias WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	return d, err
}

// Total conta as diligências não excluídas (contador do menu Agenda).
func (s *Service) Total(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM diligencias WHERE excluido_em IS NULL`).Scan(&n)
	return n, err
}

// EmAberto devolve as diligências com data-limite de hoje em diante, por data-limite e hora (diligenciasEmAberto do B).
func (s *Service) EmAberto(ctx context.Context, hoje time.Time) ([]Linha, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM diligencias WHERE excluido_em IS NULL AND data_limite >= $1
		ORDER BY data_limite, hora NULLS FIRST, codigo`, hoje)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ds []*Diligencia
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.linhas(ctx, ds)
}

func (s *Service) linhas(ctx context.Context, ds []*Diligencia) ([]Linha, error) {
	var ids []string
	for _, d := range ds {
		if d.ClienteID != "" {
			ids = append(ids, d.ClienteID)
		}
	}
	nomes := map[string]string{}
	if len(ids) > 0 {
		rs, err := s.Pessoas.Refs(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			nomes[r.ID] = r.Nome
		}
	}
	out := make([]Linha, len(ds))
	for i, d := range ds {
		l := Linha{ID: d.ID, Codigo: d.Codigo, Titulo: d.Titulo, DataLimite: webx.DataBR(d.DataLimite), Hora: d.Hora, Local: d.Local,
			Responsavel: d.Responsavel, Status: d.Status, Processo: d.ProcessoNumero, Cancelada: d.Status == StatusCancelada}
		l.Busca = webx.SemAcento(strings.Join([]string{d.Codigo, d.Titulo, d.ProcessoNumero, nomes[d.ClienteID], d.Local, d.Responsavel,
			d.Status, d.DataISO(), l.DataLimite, d.Hora}, " "))
		out[i] = l
	}
	return out, nil
}
