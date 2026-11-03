package agenda

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, tipo, titulo, inicio, fim, local, coalesce(processo_id::text,''), coalesce(responsavel_id::text,''), version`

func scan(row pgx.Row) (*Evento, error) {
	var e Evento
	err := row.Scan(&e.ID, &e.Tipo, &e.Titulo, &e.Inicio, &e.Fim, &e.Local, &e.ProcessoID, &e.ResponsavelID, &e.Version)
	return &e, err
}

func (s *Service) enrich(ctx context.Context, list []Evento) {
	var pids, uids []string
	for _, e := range list {
		if e.ProcessoID != "" {
			pids = append(pids, e.ProcessoID)
		}
		if e.ResponsavelID != "" {
			uids = append(uids, e.ResponsavelID)
		}
	}
	pm, _ := s.Processos.Resumos(ctx, pids)
	um, _ := s.Users.Emails(ctx, uids)
	for i := range list {
		list[i].ProcessoNumero, list[i].Responsavel = pm[list[i].ProcessoID], um[list[i].ResponsavelID]
	}
}

// Filtro das listas. De/Ate em UTC; zero = sem limite.
type Filtro struct {
	Tipo     string
	De, Ate  time.Time
	Passados bool // inclui eventos anteriores a "De"
}

// List devolve eventos em ordem cronológica.
func (s *Service) List(ctx context.Context, f Filtro, limit, offset int) ([]Evento, error) {
	var de, ate *time.Time
	if !f.De.IsZero() {
		de = &f.De
	}
	if !f.Ate.IsZero() {
		ate = &f.Ate
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM agenda_eventos
		WHERE excluido_em IS NULL AND ($1 = '' OR tipo = $1)
		  AND ($2::timestamptz IS NULL OR inicio >= $2) AND ($3::timestamptz IS NULL OR inicio < $3)
		ORDER BY inicio, id LIMIT $4 OFFSET $5`, f.Tipo, de, ate, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Evento
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.enrich(ctx, out)
	return out, nil
}

// Get devolve o evento com os participantes.
func (s *Service) Get(ctx context.Context, id string) (*Evento, error) {
	e, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM agenda_eventos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT pessoa_id FROM agenda_participantes WHERE evento_id=$1 ORDER BY pessoa_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		ids = append(ids, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	names, _ := s.Pessoas.Names(ctx, ids)
	for _, pid := range ids {
		e.Participantes = append(e.Participantes, Participante{PessoaID: pid, Nome: names[pid]})
	}
	l := []Evento{*e}
	s.enrich(ctx, l)
	l[0].Participantes = e.Participantes
	return &l[0], nil
}

// Proximos devolve os eventos dos próximos dias (a partir de agora), para a tela "Hoje".
func (s *Service) Proximos(ctx context.Context, dias, limit int) ([]Evento, error) {
	now := time.Now()
	return s.List(ctx, Filtro{De: now.Add(-2 * time.Hour), Ate: now.AddDate(0, 0, dias)}, limit, 0)
}

// ExisteNoInstante procura outro evento (não excluído) que comece no mesmo minuto e tenha o mesmo
// responsável (ou ambos sem responsável). Devolve «título» às HH:MM do primeiro encontrado, ou "".
func (s *Service) ExisteNoInstante(ctx context.Context, inicio time.Time, responsavelID, ignorarID string) (string, error) {
	var titulo string
	var ini time.Time
	err := s.Pool.QueryRow(ctx, `SELECT titulo, inicio FROM agenda_eventos
		WHERE excluido_em IS NULL AND date_trunc('minute', inicio) = date_trunc('minute', $1::timestamptz)
		  AND responsavel_id IS NOT DISTINCT FROM $2::uuid AND ($3 = '' OR id <> $3::uuid)
		ORDER BY created_at, id LIMIT 1`, inicio, webx.Nullable(responsavelID), ignorarID).Scan(&titulo, &ini)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return "«" + titulo + "» às " + ini.In(webx.SP()).Format("15:04"), nil
}
