package prazos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

const cols = `id, descricao, coalesce(processo_id::text,''), data_fatal, fundamento, fonte, origem, situacao,
	coalesce(responsavel_id::text,''), coalesce(confirmado_por::text,''), confirmado_em, cumprido_em, version`

func scan(row pgx.Row) (*Prazo, error) {
	var p Prazo
	err := row.Scan(&p.ID, &p.Descricao, &p.ProcessoID, &p.DataFatal, &p.Fundamento, &p.Fonte, &p.Origem, &p.Situacao,
		&p.ResponsavelID, &p.ConfirmadoPor, &p.ConfirmadoEm, &p.CumpridoEm, &p.Version)
	return &p, err
}

// enrich preenche número do processo, e-mails e os dias restantes (hoje = data de Brasília).
func (s *Service) enrich(ctx context.Context, list []Prazo) {
	var pids, uids []string
	for _, p := range list {
		if p.ProcessoID != "" {
			pids = append(pids, p.ProcessoID)
		}
		for _, u := range []string{p.ResponsavelID, p.ConfirmadoPor} {
			if u != "" {
				uids = append(uids, u)
			}
		}
	}
	pm, _ := s.Processos.Resumos(ctx, pids)
	um, _ := s.Users.Emails(ctx, uids)
	hoje := webx.Hoje(time.Now())
	for i := range list {
		p := &list[i]
		p.ProcessoNumero, p.Responsavel, p.ConfirmadoPor = pm[p.ProcessoID], um[p.ResponsavelID], um[p.ConfirmadoPor]
		p.DiasRestantes = int(p.DataFatal.Sub(hoje).Hours() / 24)
	}
}

// Filtro das listas.
type Filtro struct {
	Situacao    string // "" = abertos (a conferir + confirmados); "cumpridos"; "todos"
	Q           string
	ProcessoID  string
	Responsavel string
}

// List devolve prazos ordenados pela data fatal (os mais urgentes primeiro).
func (s *Service) List(ctx context.Context, f Filtro, limit, offset int) ([]Prazo, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM prazos
		WHERE excluido_em IS NULL
		  AND (($1 = '' AND situacao IN ('a_conferir','confirmado')) OR $1 = 'todos'
		       OR ($1 = 'cumpridos' AND situacao IN ('cumprido','cancelado')))
		  AND ($2 = '' OR descricao ILIKE $3)
		  AND ($4 = '' OR processo_id = $4::uuid) AND ($5 = '' OR responsavel_id = $5::uuid)
		ORDER BY data_fatal, created_at, id LIMIT $6 OFFSET $7`,
		f.Situacao, f.Q, webx.Like(f.Q), f.ProcessoID, f.Responsavel, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Prazo
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	s.enrich(ctx, out)
	return out, nil
}

// Get devolve um prazo não excluído.
func (s *Service) Get(ctx context.Context, id string) (*Prazo, error) {
	p, err := scan(s.Pool.QueryRow(ctx, `SELECT `+cols+` FROM prazos WHERE id=$1 AND excluido_em IS NULL`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, webx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	l := []Prazo{*p}
	s.enrich(ctx, l)
	return &l[0], nil
}

// Criticos devolve os prazos abertos vencidos ou que vencem em até `dias` dias (tela "Hoje"), e quantos aguardam conferência.
func (s *Service) Criticos(ctx context.Context, dias, limit int) (list []Prazo, aConferir int, err error) {
	ate := webx.Hoje(time.Now()).AddDate(0, 0, dias)
	rows, err := s.Pool.Query(ctx, `SELECT `+cols+` FROM prazos
		WHERE excluido_em IS NULL AND situacao IN ('a_conferir','confirmado') AND data_fatal <= $1
		ORDER BY data_fatal, id LIMIT $2`, ate, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM prazos WHERE excluido_em IS NULL AND situacao='a_conferir'`).Scan(&aConferir); err != nil {
		return nil, 0, err
	}
	s.enrich(ctx, list)
	return list, aConferir, nil
}
