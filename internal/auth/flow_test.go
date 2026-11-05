package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/migrate"
	"painelteodoro/internal/platform/ratelimit"
)

var (
	reCSRF   = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	reSecret = regexp.MustCompile(`<code class="chave">([^<]+)</code>`)
	reCodes  = regexp.MustCompile(`<li><code>([a-z2-7]{5}-[a-z2-7]{5})</code></li>`)
)

// env monta um app completo sobre um PostgreSQL de teste (PAINEL_TEST_DB).
type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	a      *Auth
	srv    *httptest.Server
	mux    *http.ServeMux
	offset time.Duration
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
	if err := migrate.Reset(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, pool: pool}
	e.a = &Auth{
		Store: &Store{Pool: pool}, Pool: pool, Log: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		FieldKey: []byte("0123456789abcdef0123456789abcdef"), SessionKey: []byte("fedcba9876543210fedcba9876543210"),
		Limiter: ratelimit.New(1000, time.Minute), Now: func() time.Time { return time.Now().Add(e.offset) },
	}
	wa, err := webauthn.New(&webauthn.Config{RPID: "localhost", RPDisplayName: "Teste", RPOrigins: []string{"http://localhost"}})
	if err != nil {
		t.Fatal(err)
	}
	e.a.WA = wa
	mux := http.NewServeMux()
	rt := NewRouter(e.a, mux)
	e.a.Register(rt)
	e.a.RegisterWebAuthn(rt)
	e.a.RegisterAccount(rt)
	e.a.RegisterAdmin(rt)
	e.a.RegisterPages(rt)
	e.mux = mux
	e.srv = httptest.NewServer(mux)
	t.Cleanup(e.srv.Close)
	return e
}

func (e *env) user(email string, role Role, pw string) string {
	e.t.Helper()
	h, _ := HashPassword(pw)
	tx, _ := e.pool.Begin(context.Background())
	defer func() { _ = tx.Rollback(context.Background()) }()
	id, err := e.a.Store.CreateUser(context.Background(), tx, email, role, h, false)
	if err != nil {
		e.t.Fatal(err)
	}
	_ = tx.Commit(context.Background())
	return id
}

// browser guarda cookies à mão (os cookies são Secure e o servidor de teste é http).
type browser struct {
	e       *env
	cookies map[string]string
}

func (e *env) browser() *browser { return &browser{e: e, cookies: map[string]string{}} }

func (b *browser) do(method, path string, form url.Values) (*http.Response, string) {
	b.e.t.Helper()
	req, _ := http.NewRequest(method, b.e.srv.URL+path, strings.NewReader(form.Encode()))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range b.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	for _, ck := range resp.Cookies() {
		if ck.MaxAge < 0 {
			delete(b.cookies, ck.Name)
		} else {
			b.cookies[ck.Name] = ck.Value
		}
	}
	return resp, string(body)
}

func (b *browser) csrfFrom(path string) string {
	b.e.t.Helper()
	_, body := b.do("GET", path, nil)
	m := reCSRF.FindStringSubmatch(body)
	if m == nil {
		b.e.t.Fatalf("sem token CSRF em %s: %s", path, body)
	}
	return m[1]
}

// loginToSetup faz a senha e devolve o navegador na tela de cadastro do 2FA.
func (b *browser) password(email, pw string) *http.Response {
	tok := b.csrfFrom("/login")
	resp, _ := b.do("POST", "/login", url.Values{"csrf_token": {tok}, "email": {email}, "senha": {pw}})
	return resp
}

func (e *env) events(action string) int {
	var n int
	_ = e.pool.QueryRow(context.Background(), "SELECT count(*) FROM audit_events WHERE action=$1", action).Scan(&n)
	return n
}

func TestFluxoCompletoLoginEMFA(t *testing.T) {
	e := newEnv(t)
	e.user("ana@exemplo.test", Advogado, "cavalo-bateria-grampo-7")
	b := e.browser()

	if r, _ := b.do("GET", "/", nil); r.StatusCode != 303 || r.Header.Get("Location") != "/login" {
		t.Fatalf("sem sessão deveria ir ao login: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
	if r, _ := b.do("POST", "/login", url.Values{"email": {"ana@exemplo.test"}, "senha": {"x"}}); r.StatusCode != 403 {
		t.Fatalf("POST sem CSRF = %d, esperado 403", r.StatusCode)
	}
	if r := b.password("ana@exemplo.test", "senha-errada-qualquer"); r.StatusCode != 401 {
		t.Fatalf("senha errada = %d", r.StatusCode)
	}
	r := b.password("ana@exemplo.test", "cavalo-bateria-grampo-7")
	if r.StatusCode != 303 || r.Header.Get("Location") != "/2fa/setup" {
		t.Fatalf("senha ok deveria ir ao cadastro do 2FA: %d %s", r.StatusCode, r.Header.Get("Location"))
	}
	// Com senha ok mas sem 2FA, nada protegido abre.
	if r, _ := b.do("GET", "/", nil); r.StatusCode != 303 || r.Header.Get("Location") != "/2fa" {
		t.Fatalf("sem 2FA a home deveria redirecionar: %d %s", r.StatusCode, r.Header.Get("Location"))
	}

	_, page := b.do("GET", "/2fa/setup", nil)
	secret := strings.ReplaceAll(reSecret.FindStringSubmatch(page)[1], " ", "")
	if r, _ := b.do("GET", "/2fa/setup/qr.png", nil); r.Header.Get("Content-Type") != "image/png" {
		t.Error("QR deveria ser PNG")
	}
	tok := b.csrfFrom("/2fa/setup")
	if r, _ := b.do("POST", "/2fa/setup", url.Values{"csrf_token": {tok}, "codigo": {"000000"}}); r.StatusCode != 401 {
		t.Fatalf("código errado no cadastro = %d", r.StatusCode)
	}
	code, _ := totp.GenerateCode(secret, e.a.Now())
	r, body := b.do("POST", "/2fa/setup", url.Values{"csrf_token": {tok}, "codigo": {code}})
	codes := reCodes.FindAllStringSubmatch(body, -1)
	if r.StatusCode != 200 || len(codes) != 8 {
		t.Fatalf("cadastro: status=%d códigos=%d", r.StatusCode, len(codes))
	}
	if r, _ := b.do("GET", "/", nil); r.StatusCode != 302 || r.Header.Get("Location") != "/agenda" {
		t.Fatalf("home após 2FA = %d (a página inicial é a Agenda)", r.StatusCode)
	}

	// Logout exige CSRF; e depois a sessão acaba.
	if r, _ := b.do("POST", "/logout", url.Values{}); r.StatusCode != 403 {
		t.Fatalf("logout sem CSRF = %d", r.StatusCode)
	}
	b.do("POST", "/logout", url.Values{"csrf_token": {b.csrfFrom("/conta")}})
	if r, _ := b.do("GET", "/", nil); r.StatusCode != 303 {
		t.Fatalf("após logout a home deveria redirecionar: %d", r.StatusCode)
	}

	// Novo login: o MESMO código TOTP (mesmo passo) é recusado; o do passo seguinte entra.
	b2 := e.browser()
	if r := b2.password("ana@exemplo.test", "cavalo-bateria-grampo-7"); r.Header.Get("Location") != "/2fa" {
		t.Fatalf("segundo login deveria ir a /2fa: %s", r.Header.Get("Location"))
	}
	tok = b2.csrfFrom("/2fa")
	if r, _ := b2.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {code}}); r.StatusCode != 401 {
		t.Fatalf("reuso do código TOTP = %d, esperado 401", r.StatusCode)
	}
	e.offset = 30 * time.Second
	next, _ := totp.GenerateCode(secret, e.a.Now())
	if r, _ := b2.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {next}}); r.StatusCode != 303 || r.Header.Get("Location") != "/" {
		t.Fatalf("código do passo seguinte = %d %s", r.StatusCode, r.Header.Get("Location"))
	}

	// Código de recuperação vale uma vez só.
	rc := codes[0][1]
	b3 := e.browser()
	b3.password("ana@exemplo.test", "cavalo-bateria-grampo-7")
	tok = b3.csrfFrom("/2fa")
	if r, _ := b3.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {rc}}); r.StatusCode != 303 {
		t.Fatalf("código de recuperação = %d", r.StatusCode)
	}
	b4 := e.browser()
	b4.password("ana@exemplo.test", "cavalo-bateria-grampo-7")
	tok = b4.csrfFrom("/2fa")
	if r, _ := b4.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {rc}}); r.StatusCode != 401 {
		t.Fatalf("reuso do código de recuperação = %d, esperado 401", r.StatusCode)
	}

	if n, err := audit.Verify(context.Background(), e.pool); err != nil || n < 8 {
		t.Fatalf("cadeia de auditoria: n=%d err=%v", n, err)
	}
	for _, a := range []string{"login.falha", "login.senha_ok", "mfa.ativado", "login.completo", "logout"} {
		if e.events(a) == 0 {
			t.Errorf("evento %q não foi auditado", a)
		}
	}
}

// fullLogin leva o usuário até a home, cadastrando o 2FA.
func (e *env) fullLogin(email, pw string) *browser {
	e.t.Helper()
	b := e.browser()
	b.password(email, pw)
	_, page := b.do("GET", "/2fa/setup", nil)
	secret := strings.ReplaceAll(reSecret.FindStringSubmatch(page)[1], " ", "")
	code, _ := totp.GenerateCode(secret, e.a.Now())
	b.do("POST", "/2fa/setup", url.Values{"csrf_token": {b.csrfFrom("/2fa/setup")}, "codigo": {code}})
	return b
}

func TestRBACNaAuditoria(t *testing.T) {
	e := newEnv(t)
	e.user("adv@exemplo.test", Advogado, "cavalo-bateria-grampo-7")
	e.user("adm@exemplo.test", Administrador, "cavalo-bateria-grampo-7")

	adv := e.fullLogin("adv@exemplo.test", "cavalo-bateria-grampo-7")
	if r, _ := adv.do("GET", "/admin/auditoria", nil); r.StatusCode != 403 {
		t.Fatalf("advogado na auditoria = %d, esperado 403", r.StatusCode)
	}
	adm := e.fullLogin("adm@exemplo.test", "cavalo-bateria-grampo-7")
	r, body := adm.do("GET", "/admin/auditoria", nil)
	if r.StatusCode != 200 || !strings.Contains(body, "Cadeia íntegra") {
		t.Fatalf("administrador na auditoria = %d", r.StatusCode)
	}
	if e.events("acesso.negado") == 0 {
		t.Error("o 403 deveria ter sido auditado")
	}
}

func TestBloqueioProgressivo(t *testing.T) {
	e := newEnv(t)
	e.user("lock@exemplo.test", Assistente, "cavalo-bateria-grampo-7")
	b := e.browser()
	for i := 0; i < 5; i++ {
		b.password("lock@exemplo.test", "errada-errada-errada")
	}
	if r := b.password("lock@exemplo.test", "cavalo-bateria-grampo-7"); r.StatusCode != 401 {
		t.Fatalf("conta bloqueada aceitou a senha certa: %d", r.StatusCode)
	}
	if e.events("conta.bloqueada") != 1 {
		t.Error("bloqueio deveria ser auditado uma vez")
	}
	e.offset = 2 * time.Minute // passou 1 min de bloqueio
	if r := b.password("lock@exemplo.test", "cavalo-bateria-grampo-7"); r.StatusCode != 303 {
		t.Fatalf("após o prazo deveria entrar: %d", r.StatusCode)
	}
}

func TestRotaSemPermissaoNaoSobe(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registrar rota sem permissão deveria derrubar o programa")
		}
	}()
	NewRouter(&Auth{}, http.NewServeMux()).Handle("GET /x", "", nil)
}

func TestLockSchedule(t *testing.T) {
	casos := []struct {
		n    int
		d    time.Duration
		perm bool
	}{{4, 0, false}, {5, time.Minute, false}, {6, 0, false}, {10, 5 * time.Minute, false},
		{15, 30 * time.Minute, false}, {20, 0, true}, {25, 0, true}}
	for _, c := range casos {
		if d, p := LockSchedule(c.n); d != c.d || p != c.perm {
			t.Errorf("LockSchedule(%d) = %v,%v", c.n, d, p)
		}
	}
}
