// Package cfaccess valida o JWT que o Cloudflare Access injeta em toda requisição
// (cabeçalho Cf-Access-Jwt-Assertion). Sem JWT válido, o app recusa a requisição.
package cfaccess

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"
)

// Fetcher devolve o JSON de chaves públicas (JWKS) do time.
type Fetcher func(ctx context.Context) ([]byte, error)

// Verifier confere assinatura, emissor, audiência e validade.
type Verifier struct {
	issuer, aud string
	fetch       Fetcher
	now         func() time.Time

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

// New cria o verificador para o domínio do time (ex.: "escritorio.cloudflareaccess.com") e a audiência (AUD).
func New(teamDomain, aud string, fetch Fetcher) *Verifier {
	return &Verifier{issuer: "https://" + teamDomain, aud: aud, fetch: fetch, now: time.Now}
}

type claims struct {
	Iss   string          `json:"iss"`
	Aud   json.RawMessage `json:"aud"`
	Email string          `json:"email"`
	Exp   int64           `json:"exp"`
	Nbf   int64           `json:"nbf"`
}

// Verify devolve o e-mail autenticado pelo Access ou o motivo da recusa.
func (v *Verifier) Verify(ctx context.Context, jwt string) (string, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return "", errors.New("JWT malformado")
	}
	var hdr struct{ Alg, Kid string }
	if err := decodeJSON(parts[0], &hdr); err != nil || hdr.Alg != "RS256" || hdr.Kid == "" {
		return "", errors.New("cabeçalho do JWT inválido")
	}
	key, err := v.key(ctx, hdr.Kid)
	if err != nil {
		return "", err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("assinatura ilegível")
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig); err != nil {
		return "", errors.New("assinatura inválida")
	}
	var c claims
	if err := decodeJSON(parts[1], &c); err != nil {
		return "", errors.New("conteúdo do JWT ilegível")
	}
	now := v.now().Unix()
	switch {
	case c.Iss != v.issuer:
		return "", errors.New("emissor inesperado")
	case !audMatches(c.Aud, v.aud):
		return "", errors.New("audiência inesperada")
	case c.Exp == 0 || now >= c.Exp:
		return "", errors.New("JWT expirado")
	case c.Nbf != 0 && now < c.Nbf:
		return "", errors.New("JWT ainda não vale")
	case c.Email == "":
		return "", errors.New("JWT sem e-mail")
	}
	return strings.ToLower(c.Email), nil
}

func audMatches(raw json.RawMessage, want string) bool {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		for _, a := range many {
			if a == want {
				return true
			}
		}
	}
	return false
}

func decodeJSON(seg string, out any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// key devolve a chave pelo kid; relê o JWKS se o kid for desconhecido (no máximo 1x por minuto).
func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.keys[kid]; ok && v.now().Sub(v.fetched) < time.Hour {
		return k, nil
	}
	if v.keys != nil && v.now().Sub(v.fetched) < time.Minute {
		return nil, errors.New("chave desconhecida")
	}
	raw, err := v.fetch(ctx)
	if err != nil {
		return nil, errors.New("não consegui obter as chaves do Access")
	}
	var set struct {
		Keys []struct{ Kid, Kty, N, E string } `json:"keys"`
	}
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, errors.New("chaves do Access ilegíveis")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		n, e := b64Int(k.N), b64Int(k.E)
		if k.Kty == "RSA" && n != nil && e != nil && e.IsInt64() {
			keys[k.Kid] = &rsa.PublicKey{N: n, E: int(e.Int64())}
		}
	}
	v.keys, v.fetched = keys, v.now()
	if k, ok := keys[kid]; ok {
		return k, nil
	}
	return nil, errors.New("chave desconhecida")
}

func b64Int(s string) *big.Int {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) == 0 {
		return nil
	}
	return new(big.Int).SetBytes(b)
}
