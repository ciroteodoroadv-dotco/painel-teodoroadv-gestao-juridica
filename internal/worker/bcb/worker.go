package bcb

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

func NewWorker(db *sql.DB, interval time.Duration, logger *slog.Logger) *Worker {
if interval < 1*time.Hour {
interval = 24 * time.Hour
}
return &Worker{
db:       db,
client:   NewClient(),
interval: interval,
logger:   logger,
}
}

func (w *Worker) Start(ctx context.Context) {
w.logger.Info("Worker do Banco Central (BCB) iniciado", "intervalo", w.interval.String())
ticker := time.NewTicker(w.interval)
defer ticker.Stop()

w.sincronizar(ctx)

for {
select {
case <-ctx.Done():
w.logger.Info("Worker do BCB encerrado")
return
case <-ticker.C:
w.sincronizar(ctx)
}
}
}

func (w *Worker) sincronizar(ctx context.Context) {
w.logger.Info("Iniciando atualização de índices econômicos no BCB")

hoje := time.Now().Format("02/01/2006")
inicio := time.Now().AddDate(0, -2, 0).Format("02/01/2006") // Últimos 2 meses

series := map[string]int{
"SELIC": SerieSelic,
"INPC":  SerieINPC,
"IPCA":  SerieIPCA,
}

for nome, codigo := range series {
dados, err := w.client.BuscarSerie(ctx, codigo, inicio, hoje)
if err != nil {
w.logger.Error("Erro ao buscar série no BCB", "indice", nome, "codigo", codigo, "erro", err)
continue
}

w.salvarIndices(ctx, nome, dados)
}
}

func (w *Worker) salvarIndices(ctx context.Context, nome string, dados []DadoSerie) {
tx, err := w.db.BeginTx(ctx, nil)
if err != nil {
return
}
defer tx.Rollback()

inseridos := 0
for _, d := range dados {
dataStr := d.Data.Format("2006-01-02")
res, err := tx.ExecContext(ctx, `
INSERT INTO indices_economicos (nome_indice, data_referencia, valor)
VALUES ($1, $2, $3)
ON CONFLICT (nome_indice, data_referencia) 
DO UPDATE SET valor = EXCLUDED.valor
`, nome, dataStr, d.Valor)

if err == nil {
if n, _ := res.RowsAffected(); n > 0 {
inseridos++
}
}
}

if err := tx.Commit(); err != nil {
w.logger.Error("Erro ao efetivar índices econômicos", "indice", nome, "erro", err)
} else {
w.logger.Info("Índices econômicos atualizados", "indice", nome, "registros", inseridos)
}
}
