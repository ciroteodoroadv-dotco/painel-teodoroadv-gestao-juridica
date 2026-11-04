package calendario

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos/calculo"
)

// Service é o cadastro auditado. Só mexe em calendarios e calendario_dias.
type Service struct {
	Pool *pgxpool.Pool

	mu    sync.Mutex
	cache map[string]cacheEnt
}

type cacheEnt struct {
	cal calculo.Calendario
	ate time.Time
}

const validadeCache = 30 * time.Second

func (s *Service) invalida() { s.mu.Lock(); s.cache = nil; s.mu.Unlock() }

func record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "calendario",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID, Meta: meta})
}

// Calendarios lista os calendários cadastrados.
func (s *Service) Calendarios(ctx context.Context) ([]Calendario, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, sigla, nome, ambito, verificar FROM calendarios ORDER BY (ambito <> 'nacional'), sigla`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Calendario
	for rows.Next() {
		var c Calendario
		if err := rows.Scan(&c.ID, &c.Sigla, &c.Nome, &c.Ambito, &c.Verificar); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CriarCalendario cadastra um calendário (a lista de dias começa vazia).
func (s *Service) CriarCalendario(ctx context.Context, ac webx.Actor, in CalInput) (string, error) {
	in, err := in.validar()
	if err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO calendarios (sigla, nome, ambito, created_by) VALUES ($1,$2,$3,$4) RETURNING id::text`,
		in.Sigla, in.Nome, in.Ambito, ac.UserID).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", webx.Invalidf("Já existe um calendário com esta sigla.")
	}
	if err != nil {
		return "", err
	}
	if err := record(ctx, tx, ac, "calendario.criado", id, map[string]string{"calendario": in.Sigla}); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// Dias lista os dias do calendário no ano (ano 0 = todos), inclusive os inativados.
func (s *Service) Dias(ctx context.Context, calID string, ano int) ([]Dia, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, calendario_id::text, data, tipo, conta_como_nao_util, descricao, fonte_citacao, fonte_url,
		observacao, coalesce((SELECT email FROM users WHERE id = verificado_por),''), ativo, motivo_inativacao
		FROM calendario_dias WHERE calendario_id = $1 AND ($2 = 0 OR extract(year FROM data) = $2) ORDER BY data, tipo`, calID, ano)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Dia
	for rows.Next() {
		var d Dia
		if err := rows.Scan(&d.ID, &d.CalendarioID, &d.Data, &d.Tipo, &d.ContaNaoUtil, &d.Descricao, &d.FonteCitacao, &d.FonteURL,
			&d.Observacao, &d.VerificadoPor, &d.Ativo, &d.MotivoInativacao); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CriarDia cadastra o dia. Quem grava é quem verificou a fonte. Ponto facultativo só conta como não útil
// quando um ADVOGADO marca (ehAdvogado), com a fonte do tribunal.
func (s *Service) CriarDia(ctx context.Context, ac webx.Actor, in DiaInput, ehAdvogado bool) (string, error) {
	in, data, err := in.validar()
	if err != nil {
		return "", err
	}
	if in.Tipo == "ponto_facultativo" && in.ContaNaoUtil && !ehAdvogado {
		return "", webx.Invalidf("Só advogado marca ponto facultativo como dia não útil (com a fonte do tribunal).")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO calendario_dias (calendario_id, data, tipo, conta_como_nao_util, descricao, fonte_citacao, fonte_url,
		observacao, verificado_por, verificado_em, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),$9) RETURNING id::text`,
		in.CalendarioID, data, in.Tipo, in.ContaNaoUtil, in.Descricao, in.FonteCitacao, in.FonteURL, in.Observacao, ac.UserID).Scan(&id)
	switch {
	case webx.IsUniqueViolation(err):
		return "", webx.Invalidf("Este dia já está cadastrado (ativo) neste calendário com o mesmo tipo.")
	case webx.IsForeignKeyViolation(err):
		return "", webx.Invalidf("Calendário não encontrado.")
	case err != nil:
		return "", err
	}
	if err := record(ctx, tx, ac, "calendario.dia_criado", id, map[string]string{"data": in.Data, "tipo": in.Tipo}); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	s.invalida()
	return id, nil
}

// InativarDia tira o dia do cálculo, preservando a linha e o motivo. Nada se apaga.
func (s *Service) InativarDia(ctx context.Context, ac webx.Actor, id, motivo string) error {
	motivo = strings.TrimSpace(motivo)
	if !tamanho(motivo, 3, 500) {
		return webx.Invalidf("Informe o motivo da inativação.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE calendario_dias SET ativo=false, motivo_inativacao=$2, inativado_por=$3, inativado_em=now()
		WHERE id=$1 AND ativo`, id, motivo, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrNotFound
	}
	if err := record(ctx, tx, ac, "calendario.dia_inativado", id, map[string]string{"motivo": motivo}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.invalida()
	return nil
}

// Dia devolve um dia pelo id (para o formulário de inativação).
func (s *Service) Dia(ctx context.Context, id string) (*Dia, error) {
	if !webx.UUID(id) {
		return nil, webx.ErrNotFound
	}
	var d Dia
	err := s.Pool.QueryRow(ctx, `SELECT id::text, calendario_id::text, data, tipo, descricao, fonte_citacao, ativo FROM calendario_dias WHERE id=$1`, id).
		Scan(&d.ID, &d.CalendarioID, &d.Data, &d.Tipo, &d.Descricao, &d.FonteCitacao, &d.Ativo)
	if err == pgx.ErrNoRows {
		return nil, webx.ErrNotFound
	}
	return &d, err
}
