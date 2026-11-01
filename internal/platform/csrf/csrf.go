// Package csrf gera e confere tokens ligados ao identificador de sessão (ou ao cookie anônimo).
package csrf

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// Token = HMAC-SHA256(chave, "csrf|"+id). Não é guardado: é recalculado a cada conferência.
func Token(key []byte, id string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("csrf|" + id))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// Valid compara em tempo constante. id ou token vazios nunca são válidos.
func Valid(key []byte, id, token string) bool {
	if id == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(Token(key, id)), []byte(token))
}
