// Comando painel: só monta as dependências e sobe o servidor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/cfaccess"
	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/db"
	"painelteodoro/internal/platform/health"
	"painelteodoro/internal/platform/httpx"
	"painelteodoro/internal/platform/logs"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/ratelimit"
	"painelteodoro/web"
)

func main() {
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		exit(serve())
	case args[0] == "healthcheck":
		os.Exit(healthcheck())
	default:
		exit(runCLI(args))
	}
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func serve() error {
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
	if err := migrate.Up(ctx, pool); err != nil {
		return fmt.Errorf("migração do banco falhou: %w", err)
	}

	mux := http.NewServeMux()
	health.Register(mux, pool, log) // internas: o middleware do Access só as libera sem Cf-Ray

	a := &auth.Auth{
		Store: &auth.Store{Pool: pool}, Pool: pool, FieldKey: cfg.FieldKey, SessionKey: cfg.SessionKey,
		Log: log, Limiter: ratelimit.New(10, time.Minute), RequireAccessEmail: !cfg.Dev,
	}
	rt := auth.NewRouter(a, mux)
	rt.Handle("GET /static/", auth.Public, func(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
		http.StripPrefix("/static/", http.FileServerFS(web.Static())).ServeHTTP(w, r)
	})
	a.Register(rt)
	a.RegisterPages(rt)

	srv := httpx.NewServer(cfg.Addr, httpx.Chain(cfaccess.Middleware(verifier(cfg, log), log, mux), log))
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("servidor iniciado", "addr", cfg.Addr, "rotas", len(rt.Routes()))

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

// verifier devolve nil no modo dev (sem Cloudflare), avisando em voz alta.
func verifier(cfg config.Config, log *slog.Logger) *cfaccess.Verifier {
	if cfg.Dev {
		log.Warn("MODO DEV: JWT do Cloudflare Access NÃO é exigido. Nunca use em servidor.")
		return nil
	}
	return cfaccess.New(cfg.AccessTeamDomain, cfg.AccessAUD, cfaccess.HTTPFetcher(cfg.AccessTeamDomain))
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
