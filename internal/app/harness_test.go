package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/csrf"
	"painelteodoro/internal/platform/httpx"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/ratelimit"
)

var (
	sessionKey = []byte("fedcba9876543210fedcba9876543210")
	fieldKey   = []byte("0123456789abcdef0123456789abcdef")
)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	a    *auth.Auth
	srv  *httptest.Server
	docs string // pasta dos documentos desta execução
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("PAINEL_TEST_DB")
	if dsn == "" {
		t.Skip("PAINEL_TEST_DB não definida: teste de integração pulado")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	t.Cleanup(func() { // deixa o banco limpo: o Reset das migrações (0008/0009/0010) só reverte com tabelas vazias
		_, _ = pool.Exec(context.Background(), `TRUNCATE pessoas, processos, atendimentos, agenda_compromissos, audiencias, diligencias CASCADE`)
	})
	_, _ = pool.Exec(ctx, `TRUNCATE pessoas, processos, atendimentos, agenda_compromissos, audiencias, diligencias CASCADE`) // a migração 0008 só reverte com tabelas vazias; erro ignorado na 1ª execução
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	a := &auth.Auth{Store: &auth.Store{Pool: pool}, Pool: pool, Log: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		FieldKey: fieldKey, SessionKey: sessionKey, Limiter: ratelimit.New(1000, time.Minute)}
	mux := http.NewServeMux()
	rt := auth.NewRouter(a, mux)
	a.Register(rt)
	a.RegisterAccount(rt)
	a.RegisterPages(rt)
	docs := t.TempDir()
	if err := RegisterDomains(a, rt, pool, fieldKey, docs); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(httpx.Chain(mux, a.Log)) // cadeia real: limites de corpo, cabeçalhos
	t.Cleanup(srv.Close)
	return &env{t: t, pool: pool, a: a, srv: srv, docs: docs}
}

// user cria um usuário e devolve um navegador JÁ autenticado (sessão com 2FA concluído).
func (e *env) user(email string, role auth.Role) (*browser, string) {
	e.t.Helper()
	ctx := context.Background()
	h, _ := auth.HashPassword("cavalo-bateria-grampo-7")
	tx, _ := e.pool.Begin(ctx)
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := e.a.Store.CreateUser(ctx, tx, email, role, h, false)
	if err != nil {
		e.t.Fatal(err)
	}
	tok, err := e.a.Store.CreateSession(ctx, tx, id, true, time.Now())
	if err != nil {
		e.t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		e.t.Fatal(err)
	}
	return &browser{e: e, sid: tok}, id
}

type browser struct {
	e   *env
	sid string
}

func (b *browser) token() string { return csrf.Token(sessionKey, b.sid) }

func (b *browser) do(method, path string, form url.Values, hdr map[string]string) (*http.Response, string) {
	b.e.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	if method == "POST" {
		form.Set("csrf_token", b.token())
	}
	var body io.Reader
	if method == "POST" {
		body = strings.NewReader(form.Encode())
	}
	req, _ := http.NewRequest(method, b.e.srv.URL+path, body)
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.AddCookie(&http.Cookie{Name: "__Host-sid", Value: b.sid})
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, string(out)
}

func (b *browser) get(path string) (*http.Response, string) { return b.do("GET", path, nil, nil) }
func (b *browser) post(path string, f url.Values) (*http.Response, string) {
	return b.do("POST", path, f, nil)
}

func (e *env) count(sql string, args ...any) int {
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func idFrom(t *testing.T, r *http.Response, prefix string) string {
	t.Helper()
	loc := r.Header.Get("Location")
	if !strings.HasPrefix(loc, prefix) {
		t.Fatalf("redirecionamento inesperado: status=%d location=%q", r.StatusCode, loc)
	}
	id := strings.TrimPrefix(loc, prefix)
	if i := strings.IndexAny(id, "?/"); i >= 0 {
		id = id[:i]
	}
	return id
}
