package prazos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

// O prazo fatal de uma tarefa é espelhado aqui (uma linha por tarefa, coluna tarefa_id). A REGRA DE OURO continua:
// o prazo só vale depois de confirmado por advogado; quem não é advogado deixa "a conferir". O sistema não calcula a data.

// SincronizarNaTx cria ou atualiza o prazo da tarefa DENTRO da transação do chamador (a da própria tarefa) e devolve a
// situação resultante. Não faz commit. Nunca confirma: o prazo novo nasce "a conferir"; mudar a data (ou reabrir um
// prazo encerrado) volta a "a conferir" sem quem/quando; manter a data mantém a situação. Confirmar é ConfirmarDaTarefa.
func (s *Service) SincronizarNaTx(ctx context.Context, tx pgx.Tx, ac webx.Actor, tarefaID, processoID, descricao string, data *time.Time) (string, error) {
	if data == nil {
		return "", webx.Invalidf("Informe a data do prazo fatal.")
	}
	var id, sit string
	var err error
	var velha time.Time
	err = tx.QueryRow(ctx, `SELECT id, data_fatal, situacao FROM prazos WHERE tarefa_id=$1::uuid AND excluido_em IS NULL FOR UPDATE`, tarefaID).Scan(&id, &velha, &sit)
	novo := AConferir
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `INSERT INTO prazos (descricao, processo_id, data_fatal, situacao, tarefa_id, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$6) RETURNING id`,
			descricao, webx.Nullable(processoID), data, novo, tarefaID, ac.UserID).Scan(&id)
		if err == nil {
			err = s.record(ctx, tx, ac, "prazo.criado", id, map[string]string{"etapa": novo})
		}
	case err != nil:
		return "", err
	default:
		novo = situacaoDepois(sit, velha.Equal(*data))
		_, err = tx.Exec(ctx, `UPDATE prazos SET descricao=$2, processo_id=$3, data_fatal=$4, situacao=$5, cumprido_em=NULL,
			confirmado_por = CASE WHEN $5 = 'confirmado' THEN confirmado_por END,
			confirmado_em = CASE WHEN $5 = 'confirmado' THEN confirmado_em END,
			updated_at=now(), updated_by=$6, version=version+1 WHERE id=$1`,
			id, descricao, webx.Nullable(processoID), data, novo, ac.UserID)
		if err == nil {
			err = s.record(ctx, tx, ac, "prazo.alterado", id, map[string]string{"etapa": novo})
		}
	}
	return novo, err
}

// situacaoDepois decide a situação do prazo da tarefa depois de salvar. Nunca confirma.
func situacaoDepois(atual string, mesmaData bool) string {
	if atual == Cumprido || atual == Cancelado || !mesmaData {
		return AConferir
	}
	return atual
}

// SituacoesDeTarefas devolve tarefa → situação do prazo (só das que têm prazo vivo).
func (s *Service) SituacoesDeTarefas(ctx context.Context, tarefaIDs []string) (map[string]string, error) {
	out := map[string]string{}
	if len(tarefaIDs) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT tarefa_id::text, situacao FROM prazos WHERE tarefa_id = ANY($1::uuid[]) AND excluido_em IS NULL`, tarefaIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, sit string
		if err := rows.Scan(&id, &sit); err != nil {
			return nil, err
		}
		out[id] = sit
	}
	return out, rows.Err()
}

// EncerrarNaTx fecha o prazo da tarefa DENTRO da transação do chamador. motivo: "cumprida" (confirmado vira cumprido; a conferir nunca valeu, então
// é cancelado), "cancelada" (qualquer prazo aberto ou cumprido vira cancelado) ou "excluida" (o registro some das listas).
// Sem prazo vivo, não faz nada.
func (s *Service) EncerrarNaTx(ctx context.Context, tx pgx.Tx, ac webx.Actor, tarefaID, motivo string) error {
	var sql, etapa string
	switch motivo {
	case "cumprida":
		sql = `UPDATE prazos SET situacao = CASE WHEN situacao='confirmado' THEN 'cumprido' ELSE 'cancelado' END,
			cumprido_em = CASE WHEN situacao='confirmado' THEN now() END, updated_at=now(), updated_by=$2, version=version+1
			WHERE tarefa_id=$1::uuid AND excluido_em IS NULL AND situacao IN ('a_conferir','confirmado')`
		etapa = "cumprido"
	case "cancelada":
		sql = `UPDATE prazos SET situacao='cancelado', cumprido_em=NULL, updated_at=now(), updated_by=$2, version=version+1
			WHERE tarefa_id=$1::uuid AND excluido_em IS NULL AND situacao IN ('a_conferir','confirmado','cumprido')`
		etapa = Cancelado
	case "excluida":
		sql = `UPDATE prazos SET excluido_em=now(), updated_at=now(), updated_by=$2, version=version+1
			WHERE tarefa_id=$1::uuid AND excluido_em IS NULL`
		etapa = "excluido"
	default:
		return webx.Invalidf("Motivo de encerramento inválido.")
	}
	tag, err := tx.Exec(ctx, sql, tarefaID, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return s.record(ctx, tx, ac, "prazo.encerrado_pela_tarefa", tarefaID, map[string]string{"etapa": etapa})
	}
	return nil
}

// ConfirmarDaTarefa confirma o prazo "a conferir" da tarefa (o advogado atesta). A rota exige PermPrazoConfirm.
func (s *Service) ConfirmarDaTarefa(ctx context.Context, ac webx.Actor, tarefaID string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `UPDATE prazos SET situacao='confirmado', confirmado_por=$2, confirmado_em=now(), updated_at=now(),
		updated_by=$2, version=version+1 WHERE tarefa_id=$1::uuid AND excluido_em IS NULL AND situacao='a_conferir' RETURNING id`,
		tarefaID, ac.UserID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "prazo.confirmado", id, nil)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return err
}
