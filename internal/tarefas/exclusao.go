package tarefas

import (
	"context"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/webx"
)

// Excluir faz a exclusão LÓGICA da tarefa depois de reautenticar o usuário (a senha dele, não uma "senha de
// administrador" única). Senha errada não exclui e fica na auditoria. O prazo fatal ligado é encerrado.
func (s *Service) Excluir(ctx context.Context, ac webx.Actor, id string, version int, senha string) error {
	atual, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.Reauth.Confirmar(ctx, ac, senha); err != nil {
		_ = audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "tarefa.exclusao_negada", TargetType: "tarefa",
			TargetID: id, Result: audit.Falha, RequestID: ac.RequestID})
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE tarefas SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "tarefa.excluida", id); err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	if atual.PrazoFatal != nil {
		return s.Prazos.Encerrar(ctx, ac, id, "excluida")
	}
	return nil
}
