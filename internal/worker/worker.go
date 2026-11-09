package worker

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	interval time.Duration
}

func NewWorker(interval time.Duration) *Worker {
	return &Worker{
		interval: interval,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("[worker] Serviço de sincronização em background iniciado.")

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] Encerrando serviço de sincronização...")
			return
		case <-ticker.C:
			w.RunSync(ctx)
		}
	}
}

func (w *Worker) RunSync(ctx context.Context) {
	log.Println("[worker] Executando rotina de verificação de processos e intimações...")
	// Lógica de consulta de publicações e atualização de prazos
}
