package processos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

// MonitorRef é o processo como o monitor do DataJud o enxerga.
type MonitorRef struct{ ID, Numero, CriadoPor string }

// ListarParaMonitor devolve TODOS os processos não excluídos, em qualquer estado (E1 do Plano 6); quem decide se o número é
// consultável é o monitor.
func (s *Service) ListarParaMonitor(ctx context.Context) ([]MonitorRef, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text, numero_cnj, created_by::text FROM processos WHERE excluido_em IS NULL ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MonitorRef
	for rows.Next() {
		var m MonitorRef
		if err := rows.Scan(&m.ID, &m.Numero, &m.CriadoPor); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RefParaMonitor devolve um processo não excluído.
func (s *Service) RefParaMonitor(ctx context.Context, id string) (MonitorRef, bool, error) {
	if !webx.UUID(id) {
		return MonitorRef{}, false, nil
	}
	var m MonitorRef
	err := s.Pool.QueryRow(ctx, `SELECT id::text, numero_cnj, created_by::text FROM processos WHERE id=$1::uuid AND excluido_em IS NULL`, id).Scan(&m.ID, &m.Numero, &m.CriadoPor)
	if errors.Is(err, pgx.ErrNoRows) {
		return MonitorRef{}, false, nil
	}
	return m, err == nil, err
}

// MonitorProc é o processo para o qual a tarefa de monitoramento é garantida.
type MonitorProc struct{ ID, Numero, Titulo string }

// Monitoramento é o elo com a tarefa de monitoramento (implementado por Tarefas, via adaptador em internal/app). É chamado
// DENTRO da transação de criar ou mover o estado: falhou a tarefa, a mudança do processo não acontece.
type Monitoramento interface {
	Garantir(ctx context.Context, tx pgx.Tx, ac webx.Actor, p MonitorProc) (criada bool, err error)
	Baixar(ctx context.Context, tx pgx.Tx, ac webx.Actor, processoID, motivo string) (encontrada bool, err error)
}

// emAndamento: estados que exigem a tarefa de monitoramento (a tarefa do B; o MONITOR consulta todos, E1 do Plano 6).
func emAndamento(situacao string) bool { return situacao == "ativo" || situacao == "sobrestado" }

func (s *Service) efeitoMonitoramento(ctx context.Context, tx pgx.Tx, ac webx.Actor, id, situacao string) error {
	if s.Monitor == nil {
		return nil
	}
	if emAndamento(situacao) {
		var numero, titulo string
		if err := tx.QueryRow(ctx, `SELECT numero_cnj, titulo FROM processos WHERE id=$1::uuid`, id).Scan(&numero, &titulo); err != nil {
			return err
		}
		_, err := s.Monitor.Garantir(ctx, tx, ac, MonitorProc{ID: id, Numero: numero, Titulo: titulo})
		return err
	}
	_, err := s.Monitor.Baixar(ctx, tx, ac, id, situacao)
	return err
}
