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
	"github.com/jackc/pgx/v5/pgxpool"

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

	pool, err := abrirBancoDoServidor(ctx, cfg, log)
	if err != nil {
		return err
	}
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
	a.IniciarExpurgoDeSessoes(ctx) // sessões vencidas não acumulam (auditoria 05/11, item 4.4)
	a.Register(rt)
	a.RegisterWebAuthn(rt)
	a.RegisterAccount(rt)
	a.RegisterAdmin(rt)
	a.RegisterPages(rt)
	var svcs app.Servicos
	djenCfg := app.DJEN{Ativo: cfg.DJENAtivo, BaseURL: cfg.DJENURL, Intervalo: cfg.DJENIntervalo, AlertaHoras: cfg.DJENAlertaHoras}
	dj := cfg.DataJud
	datajudCfg := app.DataJud{Ativo: dj.Ativo, BaseURL: dj.URL, Chave: func() string { return string(dj.Chave) }, Intervalo: dj.Intervalo, Pausa: dj.IntervaloEntreProcessos, AlertaHoras: dj.AlertaHoras}
	ix := cfg.Indices
	indicesCfg := app.Indices{Ativo: ix.Ativo, BaseURL: ix.URL, Intervalo: ix.Intervalo, ValidadeDias: ix.ValidadeDias}
	if err := app.RegisterDomains(a, rt, pool, cfg.FieldKey, cfg.DocsDir, app.WithDJEN(djenCfg), app.WithDataJud(datajudCfg), app.WithIndices(indicesCfg), app.WithCEP(app.CEP{Ativo: cfg.CEP.Ativo, BaseURL: cfg.CEP.URL}), app.WithCalculadoraValidada(cfg.CalculadoraRegrasValidadas), app.WithServicos(&svcs)); err != nil {
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

	// Monitor do DataJud: DESLIGADO por padrão (PAINEL_DATAJUD_ATIVO). Desligado, só registra "bloqueado" (sem rede e sem agendador).
	if err := svcs.Monitor.IniciarEstado(ctx); err != nil {
		return fmt.Errorf("estado do monitor do DataJud: %w", err)
	}
	if svcs.Monitor.Iniciar(ctx, log, 5*time.Minute) {
		log.Warn("monitor do DataJud LIGADO", "intervalo", cfg.DataJud.Intervalo.String())
	} else {
		log.Info("monitor do DataJud desligado neste ambiente")
	}

	// Índices do BCB (rodapé): DESLIGADOS por padrão (PAINEL_INDICES_ATIVO). Desligados, só registram "bloqueado" (sem rede e sem agendador).
	if err := svcs.Indices.IniciarEstado(ctx); err != nil {
		return fmt.Errorf("estado dos índices: %w", err)
	}
	if svcs.Indices.Iniciar(ctx, log, 3*time.Minute) {
		log.Warn("coleta dos índices do BCB LIGADA", "intervalo", cfg.Indices.Intervalo.String())
	} else {
		log.Info("coleta dos índices do BCB desligada neste ambiente")
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

// abrirBancoDoServidor devolve o pool do papel restrito. Sem a senha do dono (o normal em produção), não migra: o
// serviço "migrate" já fez isso. Com ela (desenvolvimento ou instalação antiga), migra como antes e avisa.
func abrirBancoDoServidor(ctx context.Context, cfg config.Config, log *slog.Logger) (*pgxpool.Pool, error) {
	if !cfg.TemDono() {
		return openApp(ctx, cfg)
	}
	if !cfg.Dev {
		log.Warn("o servidor recebeu a senha do DONO do banco: em produção ela deve ficar só no serviço migrate (auditoria 05/11, item 2.1)")
	}
	owner, pool, err := openDBs(ctx, cfg)
	if err != nil {
		return nil, err
	}
	owner.Close() // o app roda só com o papel restrito
	return pool, nil
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
	// #nosec G704 -- falso positivo: host fixo (127.0.0.1) e porta vinda de PAINEL_ADDR, configuração do operador; não é entrada de usuário.
	resp, err := c.Get("http://127.0.0.1" + addr + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
