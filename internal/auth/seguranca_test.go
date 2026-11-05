package auth

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// contador finge um corpo enorme e registra quantos bytes o servidor realmente leu.
type contador struct{ restante, lidos int }

func (c *contador) Read(p []byte) (int, error) {
	if c.restante <= 0 {
		return 0, io.EOF
	}
	n := min(len(p), c.restante)
	c.restante -= n
	c.lidos += n
	return n, nil
}

// F4 da auditoria: corpo multipart não pode ser lido antes de sessão e permissão.
func TestMultipartNaoLidoSemSessao(t *testing.T) {
	e := newEnv(t)
	casos := []struct {
		nome, path string
		quer       int
	}{
		{"rota protegida sem sessão", "/conta/recuperacao", http.StatusUnauthorized},
		{"rota pública (nunca recebe arquivo)", "/login", http.StatusUnsupportedMediaType},
	}
	for _, c := range casos {
		corpo := &contador{restante: 20 << 20}
		req := httptest.NewRequest(http.MethodPost, c.path, corpo)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
		rec := httptest.NewRecorder()
		e.mux.ServeHTTP(rec, req)
		if rec.Code != c.quer {
			t.Errorf("%s: status %d, esperado %d", c.nome, rec.Code, c.quer)
		}
		if corpo.lidos > 64<<10 {
			t.Errorf("%s: o servidor leu %d bytes do corpo antes de recusar", c.nome, corpo.lidos)
		}
	}
}

// Com sessão válida o multipart continua sendo lido (envio de documentos depende disso).
func TestMultipartComSessaoContinuaFuncionando(t *testing.T) {
	e := newEnv(t)
	e.user("adv@exemplo.test", Advogado, senhaBoa)
	b := e.fullLogin("adv@exemplo.test", senhaBoa)
	tok := b.csrfFrom("/conta")
	corpo := "--xyz\r\nContent-Disposition: form-data; name=\"csrf_token\"\r\n\r\n" + tok +
		"\r\n--xyz\r\nContent-Disposition: form-data; name=\"senha_atual\"\r\n\r\n" + senhaBoa + "\r\n--xyz--\r\n"
	req, _ := http.NewRequest("POST", e.srv.URL+"/conta/recuperacao", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	for k, v := range b.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("multipart com sessão e CSRF no corpo = %d, esperado 200", resp.StatusCode)
	}
}

// F6: falha do banco não é erro da pessoa e não conta tentativa.
func TestFalhaDeBancoNaoContaTentativa(t *testing.T) {
	e := newEnv(t)
	id := e.user("fb@exemplo.test", Advogado, senhaBoa)
	e.fullLogin("fb@exemplo.test", senhaBoa) // cadastra o 2FA e gera os códigos de recuperação
	ctx := context.Background()

	b := e.browser()
	b.password("fb@exemplo.test", senhaBoa)
	tok := b.csrfFrom("/2fa")
	if _, err := e.pool.Exec(ctx, `ALTER TABLE recovery_codes RENAME TO recovery_codes_off`); err != nil {
		t.Fatal(err)
	}
	r, _ := b.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {"aaaaa-bbbbb"}})
	_, _ = e.pool.Exec(ctx, `ALTER TABLE recovery_codes_off RENAME TO recovery_codes`)
	if r.StatusCode != http.StatusInternalServerError {
		t.Fatalf("falha do banco = %d, esperado 500", r.StatusCode)
	}
	var falhas int
	_ = e.pool.QueryRow(ctx, `SELECT failed_attempts FROM users WHERE id=$1`, id).Scan(&falhas)
	if falhas != 0 {
		t.Fatalf("falha do banco contou %d tentativa(s) contra a pessoa", falhas)
	}
	// Código realmente errado continua contando.
	if r, _ := b.do("POST", "/2fa", url.Values{"csrf_token": {tok}, "codigo": {"aaaaa-bbbbb"}}); r.StatusCode != 401 {
		t.Fatalf("código errado = %d, esperado 401", r.StatusCode)
	}
	_ = e.pool.QueryRow(ctx, `SELECT failed_attempts FROM users WHERE id=$1`, id).Scan(&falhas)
	if falhas != 1 {
		t.Fatalf("código errado deveria contar 1 tentativa, contou %d", falhas)
	}
}

// F7: hash adulterado no banco não pode pedir memória/tempo absurdos.
func TestHashAdulteradoNaoExplode(t *testing.T) {
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	chave := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	curta := base64.RawStdEncoding.EncodeToString(make([]byte, 8))
	hashes := map[string]string{
		"memória de 4 GiB":   "$argon2id$v=19$m=4194304,t=3,p=2$" + salt + "$" + chave,
		"tempo 100000":       "$argon2id$v=19$m=65536,t=100000,p=2$" + salt + "$" + chave,
		"paralelismo 200":    "$argon2id$v=19$m=65536,t=3,p=200$" + salt + "$" + chave,
		"memória zero":       "$argon2id$v=19$m=0,t=3,p=2$" + salt + "$" + chave,
		"chave curta demais": "$argon2id$v=19$m=65536,t=3,p=2$" + salt + "$" + curta,
		"sal curto demais":   "$argon2id$v=19$m=65536,t=3,p=2$" + curta[:4] + "$" + chave,
	}
	for nome, h := range hashes {
		ini := time.Now()
		if VerifyPassword("qualquer-senha-longa", h) {
			t.Errorf("%s: aceitou hash inválido", nome)
		}
		if d := time.Since(ini); d > 300*time.Millisecond {
			t.Errorf("%s: demorou %v para recusar (deveria recusar sem calcular)", nome, d)
		}
	}
	// O hash legítimo continua verificando.
	h, _ := HashPassword("cavalo-bateria-grampo-7")
	if !VerifyPassword("cavalo-bateria-grampo-7", h) {
		t.Error("hash legítimo deveria verificar")
	}
}
