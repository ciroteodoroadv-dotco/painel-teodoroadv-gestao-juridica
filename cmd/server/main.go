package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"painelteodoro/internal/handler"
	"painelteodoro/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	var dbPool *pgxpool.Pool
	if dbURL != "" {
		var err error
		ctxConn, cancelConn := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelConn()

		dbPool, err = pgxpool.New(ctxConn, dbURL)
		if err != nil {
			log.Printf("Aviso: Falha ao conectar ao banco de dados: %v", err)
		} else {
			defer dbPool.Close()
		}
	}

	processoRepo := repository.NewProcessoRepository(dbPool)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.HealthCheck)
	mux.HandleFunc("/readyz", handler.ReadyCheck(dbPool))
	mux.HandleFunc("/api/processos", handler.ListarProcessos(processoRepo))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		log.Printf("Servidor PAINEL-TEODORO rodando na porta :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro fatal no servidor HTTP: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Encerrando servidor com graceful shutdown...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Erro durante o graceful shutdown: %v", err)
	}

	fmt.Println("Servidor finalizado com sucesso.")
}
