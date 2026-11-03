// Comando painel: só monta as dependências e sobe o servidor.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // fuso America/Sao_Paulo embutido (a imagem distroless não tem tzdata)

	"github.com/go-webauthn/webauthn/webauthn"

	"painelteodoro/internal/app"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/cfaccess"
	"painelteodoro/internal/platform/config"
	"painelteodoro/internal/platform/health"
	"painelteodoro/internal/platform/httpx"
	"painelteodoro/internal/platform/logs"
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

	owner, pool, err := openDBs(ctx, cfg)
	if err != nil {
		return err
	}
	owner.Close() // o app roda só com o papel restrito
	defer pool.Close()

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
	if a.WA, err = newWebAuthn(cfg.PublicURL); err != nil {
		return fmt.Errorf("chave de segurança (WebAuthn): %w", err)
	}
	a.Register(rt)
	a.RegisterWebAuthn(rt)
	a.RegisterAccount(rt)
	a.RegisterAdmin(rt)
	a.RegisterPages(rt)
	var svcs app.Servicos
	djenCfg := app.DJEN{Ativo: cfg.DJENAtivo, BaseURL: cfg.DJENURL, Intervalo: cfg.DJENIntervalo, AlertaHoras: cfg.DJENAlertaHoras}
	if err := app.RegisterDomains(a, rt, pool, cfg.FieldKey, cfg.DocsDir, app.WithDJEN(djenCfg), app.WithServicos(&svcs)); err != nil {
		return fmt.Errorf("pasta de documentos ou coleta de intimações: %w", err)
	}
	// Coleta do DJEN: DESLIGADA por padrão. Desligada, só registra o estado "bloqueado" (sem rede e sem agendador).
	if err := svcs.Intimacoes.IniciarEstado(ctx); err != nil {
		return fmt.Errorf("estado da coleta de intimações: %w", err)
	}
	if svcs.Intimacoes.Iniciar(ctx, log, 2*time.Minute) {
		log.Warn("coleta de intimações do DJEN LIGADA", "intervalo", cfg.DJENIntervalo.String())
	} else {
		log.Info("coleta de intimações do DJEN desligada neste ambiente")
	}

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

// newWebAuthn configura a chave de segurança: o domínio (RP ID) vem da URL pública.
func newWebAuthn(publicURL string) (*webauthn.WebAuthn, error) {
	u, err := url.Parse(publicURL)
	if err != nil {
		return nil, err
	}
	return webauthn.New(&webauthn.Config{
		RPDisplayName: "Painel Teodoro",
		RPID:          u.Hostname(),
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
	})
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
