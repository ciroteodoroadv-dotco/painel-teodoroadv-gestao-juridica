package worker

import (
	"context"
	"log"
	"time"
)

type ProcessoSyncService interface {
	SincronizarProcessos(ctx context.Context) error
}

type IntimacaoSyncService interface {
	SincronizarIntimacoes(ctx context.Context) error
}

type Worker struct {
	interval         time.Duration
	processoService  ProcessoSyncService
	intimacaoService IntimacaoSyncService
}

func NewWorker(interval time.Duration, procSvc ProcessoSyncService, intimSvc IntimacaoSyncService) *Worker {
	return &Worker{
		interval:         interval,
		processoService:  procSvc,
		intimacaoService: intimSvc,
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
	log.Println("[worker] Iniciando ciclo de sincronização...")

	if w.processoService != nil {
		if err := w.processoService.SincronizarProcessos(ctx); err != nil {
			log.Printf("[worker] Erro ao sincronizar processos: %v\n", err)
		}
	}

	if w.intimacaoService != nil {
		if err := w.intimacaoService.SincronizarIntimacoes(ctx); err != nil {
			log.Printf("[worker] Erro ao sincronizar intimações: %v\n", err)
		}
	}
}
