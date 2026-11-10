package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"painelteodoro/internal/handler"
	"painelteodoro/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Inicializa o Worker de Sincronização em background (intervalo de 15 minutos)
	syncWorker := worker.NewWorker(15*time.Minute, nil, nil)
	go syncWorker.Start(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.HealthCheck)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("[server] Servidor HTTP rodando na porta 8081...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] Erro fatal no servidor HTTP: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("[main] Encerrando aplicação e aguardando finalização dos serviços...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[server] Erro ao encerrar servidor HTTP: %v\n", err)
	}

	log.Println("[main] Aplicação encerrada com sucesso.")
}
