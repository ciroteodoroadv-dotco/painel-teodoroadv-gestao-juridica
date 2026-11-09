package djen

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type Worker struct {
	db        *sql.DB
	client    *Client
	interval  time.Duration
	numeroOAB string
	ufOAB     string
	logger    *slog.Logger
}

func NewWorker(db *sql.DB, numeroOAB, ufOAB string, interval time.Duration, logger *slog.Logger) *Worker {
	if interval < 10*time.Minute {
		interval = 1 * time.Hour
	}
	return &Worker{
		db:        db,
		client:    NewClient(),
		interval:  interval,
		numeroOAB: numeroOAB,
		ufOAB:     ufOAB,
		logger:    logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Worker do DJEN iniciado", "oab", w.numeroOAB, "uf", w.ufOAB, "intervalo", w.interval.String())
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.sincronizar(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker do DJEN encerrado")
			return
		case <-ticker.C:
			w.sincronizar(ctx)
		}
	}
}

func (w *Worker) sincronizar(ctx context.Context) {
	if w.numeroOAB == "" || w.ufOAB == "" {
		w.logger.Warn("SincronizaÃ§Ã£o DJEN ignorada: OAB ou UF nÃ£o configurados")
		return
	}

	hoje := time.Now().Format("2006-01-02")
	inicio := time.Now().AddDate(0, 0, -5).Format("2006-01-02") // Varre os Ãºltimos 5 dias por seguranÃ§a

	w.logger.Info("Buscando publicaÃ§Ãµes no DJEN", "inicio", inicio, "fim", hoje)

	params := QueryParams{
		NumeroOAB:  w.numeroOAB,
		UFOAB:      w.ufOAB,
		DataInicio: inicio,
		DataFim:    hoje,
		Pagina:     1,
	}

	resp, err := w.client.BuscarIntimacoes(ctx, params)
	if err != nil {
		w.logger.Error("Erro ao consultar DJEN", "erro", err)
		return
	}

	w.logger.Info("PublicaÃ§Ãµes retornadas pelo DJEN", "total", resp.Count)

	for _, item := range resp.Items {
		w.salvarIntimacao(ctx, item)
	}
}

func (w *Worker) salvarIntimacao(ctx context.Context, item ComunicacaoItem) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Tenta associar a intimaÃ§Ã£o a um processo existente no banco pelo nÃºmero CNJ
	var processoID sql.NullInt64
	_ = tx.QueryRowContext(ctx, `SELECT id FROM processos WHERE numero_cnj = $1`, item.NumeroProcesso).Scan(&processoID)

	_, err = tx.ExecContext(ctx, `
INSERT INTO intimacoes (
djen_id, processo_id, numero_processo, tribunal, orgao_julgador, 
conteudo, tipo, data_disponibilizacao, lida
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE)
ON CONFLICT (djen_id) DO NOTHING
`, item.ID, processoID, item.NumeroProcesso, item.SiglaTribunal, item.OrgaoJulgador, item.TextoComunicacao, item.TipoComunicacao, item.DataDisponibilizacao)

	if err != nil {
		w.logger.Error("Erro ao salvar intimaÃ§Ã£o do DJEN no banco", "djen_id", item.ID, "erro", err)
		return
	}

	if err := tx.Commit(); err != nil {
		w.logger.Error("Erro ao efetivar transaÃ§Ã£o de intimaÃ§Ã£o", "djen_id", item.ID, "erro", err)
	}
}
