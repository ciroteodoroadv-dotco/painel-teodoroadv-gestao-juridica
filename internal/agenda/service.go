package agenda

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras da agenda. Só toca nas próprias tabelas.
type Service struct {
	Pool      *pgxpool.Pool
	Processos ProcessoLookup
	Pessoas   PessoaLookup
	Users     UserLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "evento",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID, Meta: meta})
}

func (s *Service) checkRefs(ctx context.Context, n normalized) error {
	if n.processo != "" {
		ok, err := s.Processos.Existe(ctx, n.processo)
		if err != nil {
			return err
		}
		if !ok {
			return webx.Invalidf("Processo não encontrado.")
		}
	}
	if n.responsavel != "" {
		m, err := s.Users.Emails(ctx, []string{n.responsavel})
		if err != nil {
			return err
		}
		if _, ok := m[n.responsavel]; !ok {
			return webx.Invalidf("Responsável inválido.")
		}
	}
	return nil
}

// Create cadastra o evento.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	if err := s.checkRefs(ctx, n); err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO agenda_eventos (tipo, titulo, inicio, fim, local, processo_id, responsavel_id, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`, n.tipo, n.titulo, n.inicio, n.fim, n.local,
		webx.Nullable(n.processo), webx.Nullable(n.responsavel), ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "evento.criado", id, map[string]string{"papel": n.tipo})
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return id, err
}

// Update altera com concorrência otimista.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	n, err := in.normalize()
	if err != nil {
		return err
	}
	if err := s.checkRefs(ctx, n); err != nil {
		return err
	}
	return s.write(ctx, ac, id, "evento.alterado", `UPDATE agenda_eventos SET tipo=$3, titulo=$4, inicio=$5, fim=$6, local=$7,
		processo_id=$8, responsavel_id=$9, updated_at=now(), updated_by=$10, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, n.tipo, n.titulo, n.inicio, n.fim, n.local,
		webx.Nullable(n.processo), webx.Nullable(n.responsavel), ac.UserID)
}

// Delete marca como excluído (sem apagar).
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "evento.excluido", `UPDATE agenda_eventos SET excluido_em=now(), updated_at=now(),
		updated_by=$3, version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
}

func (s *Service) write(ctx context.Context, ac webx.Actor, id, action, sql string, args ...any) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var one int
		e := s.Pool.QueryRow(ctx, `SELECT 1 FROM agenda_eventos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
		if errors.Is(e, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		if e != nil {
			return e
		}
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, action, id, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AddParticipante convoca uma pessoa (escrita + auditoria na mesma transação).
func (s *Service) AddParticipante(ctx context.Context, ac webx.Actor, eventoID, pessoaID string) error {
	ok, err := s.Pessoas.Exist(ctx, []string{pessoaID})
	if err != nil {
		return err
	}
	if !ok[pessoaID] {
		return webx.Invalidf("Pessoa não encontrada.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var one int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM agenda_eventos WHERE id=$1 AND excluido_em IS NULL FOR UPDATE`, eventoID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO agenda_participantes (evento_id, pessoa_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, eventoID, pessoaID); err != nil {
		return err
	}
	if err = s.record(ctx, tx, ac, "evento.participante_adicionado", eventoID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RemoveParticipante retira a pessoa do evento.
func (s *Service) RemoveParticipante(ctx context.Context, ac webx.Actor, eventoID, pessoaID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `DELETE FROM agenda_participantes WHERE evento_id=$1 AND pessoa_id=$2`, eventoID, pessoaID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrNotFound
	}
	if err = s.record(ctx, tx, ac, "evento.participante_removido", eventoID, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
