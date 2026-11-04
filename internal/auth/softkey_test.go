package auth

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// softKey é um autenticador de software (ES256, atestação "none") para testar a cerimônia WebAuthn de ponta a ponta.
type softKey struct {
	t      *testing.T
	priv   *ecdsa.PrivateKey
	credID []byte
	rpID   string
	origin string
	userID []byte
	semUV  bool // autenticador sem PIN/biometria: não liga o bit "user verified"
}

func newSoftKey(t *testing.T, rpID, origin string) *softKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	_, _ = rand.Read(id)
	return &softKey{t: t, priv: k, credID: id, rpID: rpID, origin: origin}
}

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (k *softKey) clientData(typ, challenge string) []byte {
	b, _ := json.Marshal(map[string]any{"type": typ, "challenge": challenge, "origin": k.origin, "crossOrigin": false})
	return b
}

// register responde a opções de criação (JSON da API) com uma credencial nova.
func (k *softKey) register(options []byte) []byte {
	var o struct {
		PublicKey struct {
			Challenge string              `json:"challenge"`
			User      struct{ ID string } `json:"user"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(options, &o); err != nil || o.PublicKey.Challenge == "" {
		k.t.Fatalf("opções de registro inválidas: %s", options)
	}
	uid, _ := base64.RawURLEncoding.DecodeString(o.PublicKey.User.ID)
	k.userID = uid
	ponto, _ := k.priv.PublicKey.Bytes() // 0x04 || X || Y (formato não depreciado)
	cose, _ := cbor.Marshal(map[int]any{
		1: 2, 3: -7, -1: 1,
		-2: ponto[1:33],
		-3: ponto[33:65],
	})
	rp := sha256.Sum256([]byte(k.rpID))
	auth := append([]byte{}, rp[:]...)
	flags := byte(0x45) // UP | UV | AT
	if k.semUV {
		flags = 0x41 // UP | AT
	}
	auth = append(auth, flags)
	auth = binary.BigEndian.AppendUint32(auth, 0)
	auth = append(auth, make([]byte, 16)...) // AAGUID
	auth = binary.BigEndian.AppendUint16(auth, uint16(len(k.credID)))
	auth = append(auth, k.credID...)
	auth = append(auth, cose...)
	att, _ := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": auth})
	out, _ := json.Marshal(map[string]any{
		"id": b64u(k.credID), "rawId": b64u(k.credID), "type": "public-key",
		"response": map[string]any{
			"attestationObject": b64u(att),
			"clientDataJSON":    b64u(k.clientData("webauthn.create", o.PublicKey.Challenge)),
		},
		"clientExtensionResults": map[string]any{},
	})
	return out
}

// assert responde a opções de login. count é o contador de assinaturas que o autenticador informa.
func (k *softKey) assert(options []byte, count uint32) []byte {
	var o struct {
		PublicKey struct{ Challenge string } `json:"publicKey"`
	}
	if err := json.Unmarshal(options, &o); err != nil || o.PublicKey.Challenge == "" {
		k.t.Fatalf("opções de login inválidas: %s", options)
	}
	rp := sha256.Sum256([]byte(k.rpID))
	auth := append([]byte{}, rp[:]...)
	flags := byte(0x05) // UP | UV
	if k.semUV {
		flags = 0x01 // só UP
	}
	auth = append(auth, flags)
	auth = binary.BigEndian.AppendUint32(auth, count)
	cd := k.clientData("webauthn.get", o.PublicKey.Challenge)
	h := sha256.Sum256(cd)
	sig, err := ecdsa.SignASN1(rand.Reader, k.priv, sha256Sum(append(append([]byte{}, auth...), h[:]...)))
	if err != nil {
		k.t.Fatal(err)
	}
	out, _ := json.Marshal(map[string]any{
		"id": b64u(k.credID), "rawId": b64u(k.credID), "type": "public-key",
		"response": map[string]any{
			"authenticatorData": b64u(auth), "clientDataJSON": b64u(cd),
			"signature": b64u(sig), "userHandle": b64u(k.userID),
		},
		"clientExtensionResults": map[string]any{},
	})
	return out
}

func sha256Sum(b []byte) []byte { s := sha256.Sum256(b); return s[:] }

// jsonPost envia JSON com o token CSRF no cabeçalho (como o JavaScript do painel faz).
func (b *browser) jsonPost(path, csrf string, body []byte) (*http.Response, string) {
	b.e.t.Helper()
	req, _ := http.NewRequest("POST", b.e.srv.URL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	for k, v := range b.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		b.e.t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	for _, ck := range resp.Cookies() {
		if ck.MaxAge < 0 {
			delete(b.cookies, ck.Name)
		} else {
			b.cookies[ck.Name] = ck.Value
		}
	}
	return resp, buf.String()
}

func TestChaveDeSegurancaDePontaAPonta(t *testing.T) {
	e := newEnv(t)
	e.user("chave@exemplo.test", Advogado, "cavalo-bateria-grampo-7")
	key := newSoftKey(t, "localhost", "http://localhost")

	// 1) Primeiro acesso: senha → cadastra a chave como PRIMEIRO fator (sem TOTP).
	b := e.browser()
	b.password("chave@exemplo.test", "cavalo-bateria-grampo-7")
	csrf := b.csrfFrom("/2fa/setup")
	_, page := b.do("GET", "/2fa/setup", nil)
	if !strings.Contains(page, `data-wa="registrar"`) {
		t.Fatal("a tela de cadastro deveria oferecer a chave de segurança")
	}
	_, opts := b.jsonPost("/2fa/webauthn/register/begin", csrf, nil)
	r, body := b.jsonPost("/2fa/webauthn/register/finish?nome=Chave+azul", csrf, key.register([]byte(opts)))
	if r.StatusCode != 200 || !strings.Contains(body, "/2fa/recovery") {
		t.Fatalf("finish do cadastro: %d %s", r.StatusCode, body)
	}
	_, rec := b.do("GET", "/2fa/recovery", nil)
	codes := reCodes.FindAllStringSubmatch(rec, -1)
	if len(codes) != 8 {
		t.Fatalf("códigos de recuperação = %d", len(codes))
	}
	if r, _ := b.do("GET", "/2fa/recovery", nil); r.StatusCode != 303 {
		t.Error("os códigos só podem ser vistos uma vez")
	}
	if r, _ := b.do("GET", "/", nil); r.StatusCode != 302 || r.Header.Get("Location") != "/agenda" {
		t.Fatalf("home após cadastrar a chave = %d (a página inicial é a Agenda)", r.StatusCode)
	}
	// Com fator cadastrado, sessão PENDENTE não pode cadastrar outro (só a sessão com 2FA completo).
	b2 := e.browser()
	b2.password("chave@exemplo.test", "cavalo-bateria-grampo-7")
	c2 := b2.csrfFrom("/2fa")
	if r, _ := b2.jsonPost("/2fa/webauthn/register/begin", c2, nil); r.StatusCode != 403 {
		t.Fatalf("cadastro de fator em sessão sem 2FA = %d, esperado 403", r.StatusCode)
	}

	// 2) Login com a chave.
	_, mfa := b2.do("GET", "/2fa", nil)
	if !strings.Contains(mfa, `data-wa="entrar"`) {
		t.Fatal("a tela de 2FA deveria oferecer a chave")
	}
	_, lo := b2.jsonPost("/2fa/webauthn/login/begin", c2, nil)
	r, body = b2.jsonPost("/2fa/webauthn/login/finish", c2, key.assert([]byte(lo), 5))
	if r.StatusCode != 200 || !strings.Contains(body, `"redirect":"/"`) {
		t.Fatalf("login com chave: %d %s", r.StatusCode, body)
	}
	if r, _ := b2.do("GET", "/", nil); r.StatusCode != 302 {
		t.Fatalf("home após login com chave = %d", r.StatusCode)
	}

	// 3) Contador que regride = chave clonada → recusa e audita.
	b3 := e.browser()
	b3.password("chave@exemplo.test", "cavalo-bateria-grampo-7")
	c3 := b3.csrfFrom("/2fa")
	_, lo = b3.jsonPost("/2fa/webauthn/login/begin", c3, nil)
	if r, _ := b3.jsonPost("/2fa/webauthn/login/finish", c3, key.assert([]byte(lo), 3)); r.StatusCode != 401 {
		t.Fatalf("contador regredindo deveria ser recusado: %d", r.StatusCode)
	}
	// Assinatura de OUTRA chave também é recusada.
	other := newSoftKey(t, "localhost", "http://localhost")
	other.credID, other.userID = key.credID, key.userID
	_, lo = b3.jsonPost("/2fa/webauthn/login/begin", c3, nil)
	if r, _ := b3.jsonPost("/2fa/webauthn/login/finish", c3, other.assert([]byte(lo), 9)); r.StatusCode != 401 {
		t.Fatalf("assinatura de chave estranha deveria ser recusada: %d", r.StatusCode)
	}
	if e.events("mfa.falha") < 2 || e.events("mfa.chave_cadastrada") != 1 {
		t.Error("falhas e cadastro deveriam estar auditados")
	}

	// 4) Não dá para remover a única chave se não houver outro fator.
	if r, _ := b2.do("POST", "/conta/chave/remover", map[string][]string{"csrf_token": {b2.csrfFrom("/conta")}, "id": {hexOf(key.credID)}}); r.StatusCode != 409 {
		t.Fatalf("remover único fator = %d, esperado 409", r.StatusCode)
	}
	// Sem o chute do contador: o login com contador maior volta a funcionar (a chave não foi bloqueada à toa).
	b4 := e.browser()
	b4.password("chave@exemplo.test", "cavalo-bateria-grampo-7")
	c4 := b4.csrfFrom("/2fa")
	_, lo = b4.jsonPost("/2fa/webauthn/login/begin", c4, nil)
	if r, _ := b4.jsonPost("/2fa/webauthn/login/finish", c4, key.assert([]byte(lo), 10)); r.StatusCode != 200 {
		t.Fatalf("login legítimo posterior = %d", r.StatusCode)
	}
}

func hexOf(b []byte) string {
	const d = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, d[c>>4], d[c&15])
	}
	return string(out)
}
