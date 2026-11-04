package tarefas

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// SubtituloMonitoramento marca a tarefa automática de monitoramento processual (elo cadastro → monitor, herdado do B).
const SubtituloMonitoramento = "Monitoramento"

// MonitorIn é o processo para o qual se garante a tarefa de monitoramento.
type MonitorIn struct{ ProcessoID, Numero, Titulo string }

func tituloMonitor(in MonitorIn) string {
	t := "Monitoramento processual"
	if n := strings.TrimSpace(in.Numero); n != "" {
		t += " — " + n
	} else if x := strings.TrimSpace(in.Titulo); x != "" {
		t += " — " + x
	}
	if r := []rune(t); len(r) > 300 {
		t = string(r[:300])
	}
	return t
}

// descricaoMonitor repete os textos do B (ficha/monitoramento.go).
func descricaoMonitor(numero string) string {
	d := "Tarefa criada automaticamente no cadastro do processo (elo cadastro → monitor).\n\n" +
		"Enquanto esta tarefa estiver aberta, o processo consta como EM ANDAMENTO e o\n" +
		"monitor processual (DataJud) acompanha as movimentações dele na rotina diária.\n" +
		"Ao encerrar o processo (extinto/arquivado), esta tarefa é baixada automaticamente."
	if strings.TrimSpace(numero) == "" {
		d += "\n\nATENÇÃO: ficha sem número CNJ — o DataJud só consegue monitorar\nprocesso com número preenchido. Completar o número na ficha do processo."
	}
	return d
}

// GarantirMonitoramento cria, na transação do chamador, a tarefa de monitoramento do processo se não houver uma ABERTA.
// Idempotente (o índice único impede duas abertas). Se só existir baixada, cria uma nova: a baixada é histórico.
func (s *Service) GarantirMonitoramento(ctx context.Context, tx pgx.Tx, ac webx.Actor, in MonitorIn) (bool, error) {
	var um int
	err := tx.QueryRow(ctx, `SELECT 1 FROM tarefas WHERE monitoramento AND processo_id=$1::uuid
		AND estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas') LIMIT 1`, in.ProcessoID).Scan(&um)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	codigo, err := codigos.Proximo(ctx, tx, "TAR")
	if err != nil {
		return false, err
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO tarefas (codigo, titulo, subtitulo, estado, origem_tipo, processo_id, descricao, prioridade, monitoramento, created_by, updated_by)
		VALUES ($1,$2,$3,'pendente','processo',$4::uuid,$5,'Normal',true,$6::uuid,$6::uuid) RETURNING id::text`,
		codigo, tituloMonitor(in), SubtituloMonitoramento, in.ProcessoID, descricaoMonitor(in.Numero), ac.UserID).Scan(&id)
	if err != nil {
		return false, err
	}
	return true, audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: "monitoramento.criado", TargetType: "tarefa", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID})
}

// BaixarMonitoramento é a baixa em cadeia: o processo saiu do andamento (extinto/arquivado) e a tarefa de monitoramento
// aberta vai a "Concluída com sucesso" com a nota de encerramento. encontrada=false quando não há tarefa aberta (não é erro).
func (s *Service) BaixarMonitoramento(ctx context.Context, tx pgx.Tx, ac webx.Actor, processoID, motivo string) (bool, error) {
	var id string
	err := tx.QueryRow(ctx, `UPDATE tarefas SET estado='concluidas-com-sucesso', updated_at=now(), updated_by=$2::uuid, version=version+1,
		descricao = descricao || E'\n\nMonitoramento encerrado automaticamente em ' || to_char((now() AT TIME ZONE 'America/Sao_Paulo')::date, 'YYYY-MM-DD')
			|| ': processo movido para ' || $3::text || '.'
		WHERE monitoramento AND processo_id=$1::uuid AND estado NOT IN ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas')
		RETURNING id::text`, processoID, ac.UserID, strings.TrimSpace(motivo)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: "monitoramento.baixado", TargetType: "tarefa", TargetID: id,
		Result: audit.OK, RequestID: ac.RequestID, Meta: map[string]string{"para": motivo}})
}
