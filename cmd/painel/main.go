// Comando painel: só monta as dependências e sobe o servidor.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/db"
	"painelteodoro/internal/platform/health"
	"painelteodoro/internal/platform/httpx"
	"painelteodoro/internal/platform/logs"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro ao iniciar:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.OSEnv())
	if err != nil {
		return fmt.Errorf("configuração incompleta:\n%w", err)
	}
	log := logs.New(os.Stdout, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	mux := http.NewServeMux()
	health.Register(mux, pool, log)
	srv := httpx.NewServer(cfg.Addr, httpx.Chain(mux, log))

	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("servidor iniciado", "addr", cfg.Addr)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("servidor encerrado")
	return nil
}

// healthcheck serve ao HEALTHCHECK do Docker (a imagem distroless não tem curl nem shell).
func healthcheck() int {
	addr := os.Getenv("PAINEL_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://127.0.0.1" + addr + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
