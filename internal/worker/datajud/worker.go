package datajud

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type Worker struct {
	db       *sql.DB
	client   *Client
	interval time.Duration
	logger   *slog.Logger
}

func NewWorker(db *sql.DB, apiKey string, interval time.Duration, logger *slog.Logger) *Worker {
	if interval < 1*time.Minute {
		interval = 30 * time.Minute
	}
	return &Worker{
		db:       db,
		client:   NewClient(apiKey),
		interval: interval,
		logger:   logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Worker do DataJud iniciado", "intervalo", w.interval.String())
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.sincronizar(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker do DataJud encerrado")
			return
		case <-ticker.C:
			w.sincronizar(ctx)
		}
	}
}

func (w *Worker) sincronizar(ctx context.Context) {
	w.logger.Info("Iniciando varredura de processos para atualizaÃ§Ã£o no DataJud")

	rows, err := w.db.QueryContext(ctx, `
SELECT id, numero_cnj, tribunal 
FROM processos 
WHERE status != 'arquivado' AND tribunal IS NOT NULL AND tribunal != ''
LIMIT 50
`)
	if err != nil {
		w.logger.Error("Erro ao buscar processos para sincronizaÃ§Ã£o DataJud", "erro", err)
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id int64
		var numeroCNJ, tribunal string
		if err := rows.Scan(&id, &numeroCNJ, &tribunal); err != nil {
			continue
		}

		source, err := w.client.BuscarProcesso(ctx, tribunal, numeroCNJ)
		if err != nil {
			w.logger.Warn("Falha ao consultar processo no DataJud", "numero", numeroCNJ, "tribunal", tribunal, "erro", err)
			continue
		}

		w.salvarAtualizacao(ctx, id, source)
	}
}

func (w *Worker) salvarAtualizacao(ctx context.Context, processoID int64, source *ProcessoSource) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
UPDATE processos 
SET orgao_julgador = $1, classe_processual = $2, ultima_atualizacao_datajud = NOW()
WHERE id = $3
`, source.OrgaoJulgador.Nome, source.Classe.Nome, processoID)
	if err != nil {
		w.logger.Error("Erro ao atualizar processo no banco", "id", processoID, "erro", err)
		return
	}

	for _, mov := range source.Movimentos {
		_, _ = tx.ExecContext(ctx, `
INSERT INTO processo_movimentacoes (processo_id, codigo_cnj, descricao, data_hora)
VALUES ($1, $2, $3, $4)
ON CONFLICT (processo_id, data_hora, codigo_cnj) DO NOTHING
`, processoID, mov.Codigo, mov.Nome, mov.DataHora)
	}

	if err := tx.Commit(); err != nil {
		w.logger.Error("Erro ao efetivar transaÃ§Ã£o de sincronizaÃ§Ã£o", "id", processoID, "erro", err)
	} else {
		w.logger.Info("Processo sincronizado via DataJud", "id", processoID, "movimentos", len(source.Movimentos))
	}
}
