package cfaccess

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	team = "escritorio.cloudflareaccess.com"
	aud  = "aud-de-teste"
)

type fixture struct {
	key *rsa.PrivateKey
	v   *Verifier
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func newFixture(t *testing.T) *fixture {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kid": "k1", "kty": "RSA", "n": b64(k.N.Bytes()), "e": b64(big.NewInt(int64(k.E)).Bytes()),
	}}})
	v := New(team, aud, func(context.Context) ([]byte, error) { return jwks, nil })
	return &fixture{key: k, v: v}
}

func (f *fixture) token(t *testing.T, mod func(c map[string]any)) string {
	t.Helper()
	c := map[string]any{"iss": "https://" + team, "aud": []string{aud}, "email": "Ana@Exemplo.test",
		"exp": time.Now().Add(time.Hour).Unix()}
	if mod != nil {
		mod(c)
	}
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1"})
	p, _ := json.Marshal(c)
	signing := b64(h) + "." + b64(p)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + b64(sig)
}

func TestVerifyValido(t *testing.T) {
	f := newFixture(t)
	email, err := f.v.Verify(context.Background(), f.token(t, nil))
	if err != nil || email != "ana@exemplo.test" {
		t.Fatalf("email=%q err=%v", email, err)
	}
}

func TestVerifyRecusas(t *testing.T) {
	f := newFixture(t)
	casos := map[string]func(map[string]any){
		"emissor errado":   func(c map[string]any) { c["iss"] = "https://outro.cloudflareaccess.com" },
		"audiência errada": func(c map[string]any) { c["aud"] = []string{"outra"} },
		"expirado":         func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() },
		"sem exp":          func(c map[string]any) { delete(c, "exp") },
		"sem e-mail":       func(c map[string]any) { delete(c, "email") },
		"nbf no futuro":    func(c map[string]any) { c["nbf"] = time.Now().Add(time.Hour).Unix() },
	}
	for nome, mod := range casos {
		if _, err := f.v.Verify(context.Background(), f.token(t, mod)); err == nil {
			t.Errorf("%s: deveria ser recusado", nome)
		}
	}
	good := f.token(t, nil)
	if _, err := f.v.Verify(context.Background(), good[:len(good)-4]+"AAAA"); err == nil {
		t.Error("assinatura adulterada aceita")
	}
	for _, lixo := range []string{"", "a.b", "a.b.c", "x.y.z.w"} {
		if _, err := f.v.Verify(context.Background(), lixo); err == nil {
			t.Errorf("%q deveria ser recusado", lixo)
		}
	}
}

func TestMiddleware(t *testing.T) {
	f := newFixture(t)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	var gotEmail string
	h := Middleware(f.v, log, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { gotEmail = Email(r.Context()) }))

	do := func(path string, hdr map[string]string) int {
		req := httptest.NewRequest("GET", path, nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := do("/", nil); c != 403 {
		t.Errorf("sem JWT = %d, esperado 403", c)
	}
	if c := do("/", map[string]string{"Cf-Access-Jwt-Assertion": f.token(t, nil)}); c != 200 || gotEmail != "ana@exemplo.test" {
		t.Errorf("com JWT = %d email=%q", c, gotEmail)
	}
	if c := do("/healthz", nil); c != 200 {
		t.Errorf("healthz interno = %d, esperado 200", c)
	}
	if c := do("/healthz", map[string]string{"Cf-Ray": "abc"}); c != 403 {
		t.Errorf("healthz vindo pelo Cloudflare = %d, esperado 403", c)
	}
	if Middleware(nil, log, nil) != nil {
		t.Error("modo dev (verificador nulo) deve devolver o próximo handler intacto")
	}
}

// Auditoria 05/11, item 3.1: falha ao renovar o JWKS depois de 1 h NÃO derruba o acesso (usa a chave conhecida até
// 24 h); só uma tentativa por minuto; depois de 24 h sem renovar, recusa.
func TestRenovacaoDoJWKSFalhandoUsaChaveConhecida(t *testing.T) {
	f := newFixture(t)
	jwks, _ := f.v.fetch(context.Background())
	agora := time.Now()
	falhar, buscas := false, 0
	f.v.now = func() time.Time { return agora }
	f.v.fetch = func(context.Context) ([]byte, error) {
		buscas++
		if falhar {
			return nil, io.ErrUnexpectedEOF
		}
		return jwks, nil
	}
	tok := f.token(t, nil)
	if _, err := f.v.Verify(context.Background(), tok); err != nil || buscas != 1 {
		t.Fatalf("primeira verificação: %v (buscas=%d)", err, buscas)
	}
	falhar = true
	agora = agora.Add(2 * time.Hour)
	tok = f.token(t, func(c map[string]any) { c["exp"] = agora.Add(time.Hour).Unix() })
	for i := 0; i < 5; i++ {
		if _, err := f.v.Verify(context.Background(), tok); err != nil {
			t.Fatalf("com o JWKS fora do ar, a chave conhecida deveria valer: %v", err)
		}
	}
	if buscas != 2 {
		t.Fatalf("buscas = %d, esperado 2 (no máximo uma tentativa por minuto)", buscas)
	}
	agora = agora.Add(25 * time.Hour)
	tok = f.token(t, func(c map[string]any) { c["exp"] = agora.Add(time.Hour).Unix() })
	if _, err := f.v.Verify(context.Background(), tok); err == nil {
		t.Fatal("depois de 24 h sem renovar, a chave antiga não pode valer")
	}
	falhar = false
	agora = agora.Add(2 * time.Minute)
	tok = f.token(t, func(c map[string]any) { c["exp"] = agora.Add(time.Hour).Unix() })
	if _, err := f.v.Verify(context.Background(), tok); err != nil {
		t.Fatalf("com o JWKS de volta: %v", err)
	}
}
