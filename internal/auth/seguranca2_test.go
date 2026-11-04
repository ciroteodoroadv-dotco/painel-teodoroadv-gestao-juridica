package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"painelteodoro/internal/platform/cfaccess"
	"painelteodoro/internal/platform/ratelimit"
)

// F5: negativas continuam sendo negadas sempre, mas a auditoria não cresce sem limite.
func TestNegacoesAuditadasComLimite(t *testing.T) {
	e := newEnv(t)
	e.user("adv@exemplo.test", Advogado, senhaBoa)
	b := e.fullLogin("adv@exemplo.test", senhaBoa)

	for i := 0; i < 30; i++ {
		if r, _ := b.do("GET", "/admin/auditoria", nil); r.StatusCode != 403 {
			t.Fatalf("negativa %d = %d, esperado 403 sempre", i, r.StatusCode)
		}
	}
	if n := e.events("acesso.negado"); n != 1 {
		t.Fatalf("30 negativas na mesma rota geraram %d eventos, esperado 1", n)
	}
	// Outra rota tem o seu próprio evento.
	b.do("GET", "/admin/usuarios", nil)
	if n := e.events("acesso.negado"); n != 2 {
		t.Fatalf("outra rota deveria gerar evento próprio: %d", n)
	}
	// Passada a janela, o próximo evento registra quantas foram suprimidas.
	e.offset = 2 * time.Minute
	b.do("GET", "/admin/auditoria", nil)
	var sup string
	_ = e.pool.QueryRow(context.Background(), `SELECT coalesce(meta->>'suprimidos','') FROM audit_events
		WHERE action='acesso.negado' AND meta->>'rota'='/admin/auditoria' ORDER BY seq DESC LIMIT 1`).Scan(&sup)
	if sup != "29" {
		t.Fatalf("o evento seguinte deveria informar 29 suprimidos, informou %q", sup)
	}
}

// F8: o cabeçalho Cf-Connecting-Ip só vale depois que o Access validou o JWT.
func TestClientIPConfiaNoCabecalhoSoComAccess(t *testing.T) {
	novo := func(cabecalho string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.9.9.9:1234"
		r.Header.Set("Cf-Connecting-Ip", cabecalho)
		return r
	}
	if ip := clientIP(novo("1.2.3.4")); ip != "10.9.9.9" {
		t.Errorf("sem Access o cabeçalho é forjável e deve ser ignorado: %q", ip)
	}
	comAccess := func(r *http.Request) *http.Request { return r.WithContext(cfaccess.WithEmail(r.Context(), "a@b.test")) }
	if ip := clientIP(comAccess(novo("1.2.3.4"))); ip != "1.2.3.4" {
		t.Errorf("com Access o cabeçalho do Cloudflare deve valer: %q", ip)
	}
	if ip := clientIP(comAccess(novo("2001:db8::1"))); ip != "2001:db8::1" {
		t.Errorf("IPv6 válido deveria valer: %q", ip)
	}
	for _, lixo := range []string{"", "lixo; DROP TABLE", "999.1.1.1", "1.2.3.4, 5.6.7.8"} {
		if ip := clientIP(comAccess(novo(lixo))); ip != "10.9.9.9" {
			t.Errorf("cabeçalho inválido %q deveria cair para o endereço da conexão: %q", lixo, ip)
		}
	}
}

func TestCabecalhoForjadoNaoBurlaLimiteDeLogin(t *testing.T) {
	e := newEnv(t)
	e.a.Limiter = ratelimit.New(3, time.Minute)
	e.user("alvo@exemplo.test", Advogado, senhaBoa)
	b := e.browser()
	tok := b.csrfFrom("/login")
	var limitado bool
	for i := 0; i < 8; i++ {
		req, _ := http.NewRequest("POST", e.srv.URL+"/login",
			strings.NewReader(url.Values{"csrf_token": {tok}, "email": {"alvo@exemplo.test"}, "senha": {"errada-errada-1"}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cf-Connecting-Ip", fmt.Sprintf("9.9.9.%d", i)) // tentativa de trocar de "origem" a cada vez
		for k, v := range b.cookies {
			req.AddCookie(&http.Cookie{Name: k, Value: v})
		}
		resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			limitado = true
		}
	}
	if !limitado {
		t.Fatal("trocar o cabeçalho Cf-Connecting-Ip não pode burlar o limite de tentativas por IP")
	}
}

// F10: administrador exige PIN/biometria (bit "user verified"); os demais papéis aceitam chave sem isso.
func TestChaveExigeVerificacaoDeUsuarioDoAdministrador(t *testing.T) {
	e := newEnv(t)
	e.user("adm@exemplo.test", Administrador, senhaBoa)
	e.user("adv@exemplo.test", Advogado, senhaBoa)

	// ---- administrador ----
	b := e.browser()
	b.password("adm@exemplo.test", senhaBoa)
	csrf := b.csrfFrom("/2fa/setup")
	_, opts := b.jsonPost("/2fa/webauthn/register/begin", csrf, nil)
	if !strings.Contains(opts, `"userVerification":"required"`) {
		t.Fatalf("o cadastro do administrador deveria exigir verificação de usuário: %s", opts)
	}
	chave := newSoftKey(t, "localhost", "http://localhost")
	chave.semUV = true
	if r, _ := b.jsonPost("/2fa/webauthn/register/finish?nome=sem+pin", csrf, chave.register([]byte(opts))); r.StatusCode != 400 {
		t.Fatalf("chave sem verificação de usuário no cadastro do administrador = %d, esperado 400", r.StatusCode)
	}
	_, opts = b.jsonPost("/2fa/webauthn/register/begin", csrf, nil)
	chave.semUV = false
	if r, body := b.jsonPost("/2fa/webauthn/register/finish?nome=com+pin", csrf, chave.register([]byte(opts))); r.StatusCode != 200 {
		t.Fatalf("chave com verificação de usuário deveria ser aceita: %d %s", r.StatusCode, body)
	}

	b2 := e.browser()
	b2.password("adm@exemplo.test", senhaBoa)
	c2 := b2.csrfFrom("/2fa")
	_, lo := b2.jsonPost("/2fa/webauthn/login/begin", c2, nil)
	if !strings.Contains(lo, `"userVerification":"required"`) {
		t.Fatalf("o login do administrador deveria exigir verificação de usuário: %s", lo)
	}
	chave.semUV = true
	if r, _ := b2.jsonPost("/2fa/webauthn/login/finish", c2, chave.assert([]byte(lo), 5)); r.StatusCode != 401 {
		t.Fatalf("login do administrador sem verificação de usuário = %d, esperado 401", r.StatusCode)
	}
	_, lo = b2.jsonPost("/2fa/webauthn/login/begin", c2, nil)
	chave.semUV = false
	if r, body := b2.jsonPost("/2fa/webauthn/login/finish", c2, chave.assert([]byte(lo), 6)); r.StatusCode != 200 {
		t.Fatalf("login do administrador com verificação de usuário = %d %s", r.StatusCode, body)
	}

	// ---- advogado: continua aceitando chave sem PIN ----
	b3 := e.browser()
	b3.password("adv@exemplo.test", senhaBoa)
	c3 := b3.csrfFrom("/2fa/setup")
	_, opts = b3.jsonPost("/2fa/webauthn/register/begin", c3, nil)
	if strings.Contains(opts, `"userVerification":"required"`) {
		t.Fatalf("o advogado não deveria ser obrigado a verificação de usuário: %s", opts)
	}
	k2 := newSoftKey(t, "localhost", "http://localhost")
	k2.semUV = true
	if r, body := b3.jsonPost("/2fa/webauthn/register/finish?nome=simples", c3, k2.register([]byte(opts))); r.StatusCode != 200 {
		t.Fatalf("advogado com chave sem PIN = %d %s", r.StatusCode, body)
	}
	b4 := e.browser()
	b4.password("adv@exemplo.test", senhaBoa)
	c4 := b4.csrfFrom("/2fa")
	_, lo = b4.jsonPost("/2fa/webauthn/login/begin", c4, nil)
	if r, body := b4.jsonPost("/2fa/webauthn/login/finish", c4, k2.assert([]byte(lo), 3)); r.StatusCode != 200 {
		t.Fatalf("login do advogado com chave sem PIN = %d %s", r.StatusCode, body)
	}
}
