package tarefas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de tarefas. Só toca nas próprias tabelas.
type Service struct {
	Pool      *pgxpool.Pool
	Processos ProcessoLookup
	Users     UserLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "tarefa",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
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

// Create cria uma tarefa manual.
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
	err = tx.QueryRow(ctx, `INSERT INTO tarefas (titulo, processo_id, responsavel_id, vencimento, status, origem, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,'manual',$6,$6) RETURNING id`,
		n.titulo, webx.Nullable(n.processo), webx.Nullable(n.responsavel), n.vencimento, n.status, ac.UserID).Scan(&id)
	if err == nil {
		err = s.record(ctx, tx, ac, "tarefa.criada", id)
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
	return s.write(ctx, ac, id, "tarefa.alterada", `UPDATE tarefas SET titulo=$3, processo_id=$4, responsavel_id=$5,
		vencimento=$6, status=$7, updated_at=now(), updated_by=$8, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.titulo, webx.Nullable(n.processo), webx.Nullable(n.responsavel), n.vencimento, n.status, ac.UserID)
}

// SetStatus muda só o status (ex.: concluir), com a mesma proteção de versão.
func (s *Service) SetStatus(ctx context.Context, ac webx.Actor, id string, version int, status string) error {
	if rotulo(status) == status {
		return webx.Invalidf("Status inválido.")
	}
	return s.write(ctx, ac, id, "tarefa.status_alterado", `UPDATE tarefas SET status=$3, updated_at=now(), updated_by=$4,
		version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, status, ac.UserID)
}

// Delete marca como excluída (sem apagar).
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int) error {
	return s.write(ctx, ac, id, "tarefa.excluida", `UPDATE tarefas SET excluido_em=now(), updated_at=now(), updated_by=$3,
		version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
}

// write executa o UPDATE e a auditoria na MESMA transação; 0 linhas = não existe ou versão velha.
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
		e := s.Pool.QueryRow(ctx, `SELECT 1 FROM tarefas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
		if errors.Is(e, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		if e != nil {
			return e
		}
		return webx.ErrConflict
	}
	if err = s.record(ctx, tx, ac, action, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
