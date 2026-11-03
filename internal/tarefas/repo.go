package tarefas

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, titulo, coalesce(processo_id::text,''), coalesce(responsavel_id::text,''), vencimento, status, origem, version`

func scan(row pgx.Row) (*Tarefa, error) {
	var t Tarefa
	err := row.Scan(&t.ID, &t.Titulo, &t.ProcessoID, &t.ResponsavelID, &t.Vencimento, &t.Status, &t.Origem, &t.Version)
	return &t, err
}

// Filtro das listas.
type Filtro struct {
	Q           string
	Status      string // "" = abertas (aberta + em andamento); "todas"; ou um status
	Responsavel string // "" = todos; id de usuário
}

// enrich preenche número do processo, e-mail do responsável e a marca de atraso (só exibição).
func (s *Service) enrich(ctx context.Context, list []Tarefa) {
	var pids, uids []string
	for _, t := range list {
		if t.ProcessoID != "" {
			pids = append(pids, t.ProcessoID)
		}
		if t.ResponsavelID != "" {
			uids = append(uids, t.ResponsavelID)
		}
	}
	pm, _ := s.Processos.Resumos(ctx, pids)
	um, _ := s.Users.Emails(ctx, uids)
	hoje := Hoje(time.Now())
	for i := range list {
		t := &list[i]
		t.ProcessoNumero, t.Responsavel = pm[t.ProcessoID], um[t.ResponsavelID]
		t.Atrasada = t.Aberta() && t.Vencimento != nil && t.Vencimento.Before(hoje)
	}
}

// List devolve tarefas ordenadas por vencimento (sem data por último).
func (s *Service) List(ctx context.Context, f Filtro, limit, offset int) ([]Tarefa, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM tarefas
		WHERE excluido_em IS NULL
		  AND (($1 = '' AND status IN ('aberta','em_andamento')) OR $1 = 'todas' OR status = $1)
		  AND ($2 = '' OR responsavel_id = $2::uuid)
		  AND ($3 = '' OR titulo ILIKE $4)
		ORDER BY vencimento NULLS LAST, created_at DESC, id LIMIT $5 OFFSET $6`,
		f.Status, f.Responsavel, f.Q, webx.Like(f.Q), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tarefa
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.enrich(ctx, out)
	return out, nil
}

// Get devolve uma tarefa não excluída.
func (s *Service) Get(ctx context.Context, id string) (*Tarefa, error) {
	t, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM tarefas WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l := []Tarefa{*t}
	s.enrich(ctx, l)
	return &l[0], nil
}

// Proximas devolve as tarefas em aberto do usuário vencidas ou vencendo nos próximos dias (tela "Hoje").
func (s *Service) Proximas(ctx context.Context, userID string, dias, limit int) ([]Tarefa, error) {
	ate := Hoje(time.Now()).AddDate(0, 0, dias)
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM tarefas
		WHERE excluido_em IS NULL AND status IN ('aberta','em_andamento') AND responsavel_id = $1::uuid
		  AND vencimento IS NOT NULL AND vencimento <= $2
		ORDER BY vencimento, id LIMIT $3`, userID, ate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tarefa
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.enrich(ctx, out)
	return out, nil
}

// AbertosDoProcesso conta as tarefas em aberto ligadas ao processo (usado para pedir confirmação extra ao excluí-lo).
func (s *Service) AbertosDoProcesso(ctx context.Context, processoID string) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM tarefas WHERE processo_id=$1::uuid AND excluido_em IS NULL
		AND status IN ('aberta','em_andamento','em_revisao')`, processoID).Scan(&n)
	return n, err
}
