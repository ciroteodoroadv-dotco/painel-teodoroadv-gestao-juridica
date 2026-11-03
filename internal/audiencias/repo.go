package audiencias

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id::text, codigo, tipo_ato, titulo, processo_numero, coalesce(processo_id::text,''), coalesce(cliente_pessoa_id::text,''),
	juizo, local, descricao, cancelamento_motivo, data, to_char(hora,'HH24:MI'), cancelada, version`

func scan(row pgx.Row) (*Audiencia, error) {
	var a Audiencia
	err := row.Scan(&a.ID, &a.Codigo, &a.TipoAto, &a.Titulo, &a.ProcessoNumero, &a.ProcessoID, &a.ClienteID, &a.Juizo, &a.Local,
		&a.Descricao, &a.CancelamentoMotivo, &a.Data, &a.Hora, &a.Cancelada, &a.Version)
	return &a, err
}

// Get devolve a audiência não excluída.
func (s *Service) Get(ctx context.Context, id string) (*Audiencia, error) {
	if !webx.UUID(id) {
		return nil, webx.ErrNotFound
	}
	a, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM audiencias WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	return a, err
}

// Listar devolve todas as audiências não excluídas (inclusive canceladas), por data e hora (o B ordena por data+hora).
func (s *Service) Listar(ctx context.Context) ([]*Audiencia, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM audiencias WHERE excluido_em IS NULL ORDER BY data, hora, codigo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Audiencia
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Total conta as audiências não excluídas (contador do menu).
func (s *Service) Total(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM audiencias WHERE excluido_em IS NULL`).Scan(&n)
	return n, err
}

// Linhas monta as linhas das tabelas, com o nome do cliente para a busca.
func (s *Service) Linhas(ctx context.Context, as []*Audiencia) ([]Linha, error) {
	var ids []string
	for _, a := range as {
		if a.ClienteID != "" {
			ids = append(ids, a.ClienteID)
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
	out := make([]Linha, len(as))
	for i, a := range as {
		l := Linha{ID: a.ID, TipoAto: a.TipoAto, TipoRotulo: a.RotuloTipo(), Codigo: a.Codigo, Titulo: a.Titulo, Data: webx.DataBR(a.Data),
			Hora: a.Hora, Local: a.Local, Processo: a.ProcessoNumero, Cancelada: a.Cancelada}
		l.Busca = webx.SemAcento(strings.Join([]string{a.Codigo, a.Titulo, a.ProcessoNumero, nomes[a.ClienteID], a.Juizo, a.Local,
			a.RotuloTipo(), a.DataISO(), l.Data, a.Hora, a.Descricao}, " "))
		out[i] = l
	}
	return out, nil
}

// EmAberto devolve as audiências com data de hoje em diante (audienciasEmAberto do B: não filtra as canceladas).
func (s *Service) EmAberto(ctx context.Context, hoje time.Time) ([]Linha, error) {
	as, err := s.Listar(ctx)
	if err != nil {
		return nil, err
	}
	var abertas []*Audiencia
	for _, a := range as {
		if !a.Data.Before(hoje) {
			abertas = append(abertas, a)
		}
	}
	return s.Linhas(ctx, abertas)
}

// NoHorario devolve as audiências NÃO canceladas no mesmo dia e na mesma hora exata (a regra de conflito do B).
func (s *Service) NoHorario(ctx context.Context, data time.Time, hora string) ([]Conflito, error) {
	rows, err := s.Pool.Query(ctx, `SELECT titulo, to_char(data,'YYYY-MM-DD'), to_char(hora,'HH24:MI') FROM audiencias
		WHERE excluido_em IS NULL AND NOT cancelada AND data=$1 AND hora=$2::time ORDER BY codigo`, data, hora)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Conflito
	for rows.Next() {
		var c Conflito
		if err := rows.Scan(&c.Titulo, &c.Data, &c.Hora); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Conflito é uma audiência que ocupa o mesmo dia e a mesma hora.
type Conflito struct{ Titulo, Data, Hora string }
