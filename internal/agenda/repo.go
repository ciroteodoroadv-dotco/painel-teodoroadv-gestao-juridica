package agenda

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const colsC = `id::text, tipo, descricao, data, to_char(hora,'HH24:MI'), local, coalesce(cliente_pessoa_id::text,''),
	coalesce(atendimento_id::text,''), coalesce(processo_id::text,''), coalesce(criado_por_pessoa_id::text,''), version, cancelado`

func scanC(row pgx.Row) (*Compromisso, error) {
	var c Compromisso
	err := row.Scan(&c.ID, &c.Tipo, &c.Descricao, &c.Data, &c.Hora, &c.Local, &c.ClienteID, &c.AtendimentoID, &c.ProcessoID,
		&c.CriadoPorID, &c.Version, &c.Cancelado)
	return &c, err
}

// Get devolve o compromisso não excluído.
func (s *Service) Get(ctx context.Context, id string) (*Compromisso, error) {
	if !webx.UUID(id) {
		return nil, webx.ErrNotFound
	}
	c, err := scanC(s.Pool.QueryRow(ctx, `SELECT `+colsC+` FROM agenda_compromissos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	return c, err
}

// EmAberto devolve os compromissos com data de hoje em diante, por data e hora (itensAgendaEmAberto do B).
func (s *Service) EmAberto(ctx context.Context, hoje time.Time) ([]*Compromisso, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+colsC+` FROM agenda_compromissos WHERE excluido_em IS NULL AND data >= $1
		ORDER BY data, hora, id`, hoje)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Compromisso
	for rows.Next() {
		c, err := scanC(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Contar conta os compromissos não excluídos.
func (s *Service) Contar(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM agenda_compromissos WHERE excluido_em IS NULL`).Scan(&n)
	return n, err
}
