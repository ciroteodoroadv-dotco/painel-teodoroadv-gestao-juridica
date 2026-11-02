// Package auth cuida de identidade: senha, 2FA, sessão, CSRF e papéis (RBAC).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Parâmetros Argon2id da especificação (m=64 MiB, t=3, p=2). Ficam gravados no hash.
const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	saltLen      = 16
	minPwLen     = 12

	// Limites aceitos ao LER um hash gravado (com folga sobre o que usamos, para poder aumentar os parâmetros no futuro).
	maxArgonMemory  = 256 * 1024 // KiB = 256 MiB
	maxArgonTime    = 10
	maxArgonThreads = 16
)

// HashPassword devolve o hash no formato PHC: $argon2id$v=19$m=..,t=..,p=..$salt$hash.
func HashPassword(pw string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime,
		argonThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword confere a senha usando os parâmetros gravados no próprio hash.
func VerifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	// Tetos: um hash adulterado no banco não pode pedir memória ou tempo absurdos (auditoria F7).
	if m < 8 || m > maxArgonMemory || t < 1 || t > maxArgonTime || p < 1 || p > maxArgonThreads {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(salt) < 8 || len(salt) > 64 || len(want) < 16 || len(want) > 64 {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want))) // #nosec G115 -- len(want) já validado entre 16 e 64
	return subtle.ConstantTimeCompare(got, want) == 1
}

// CheckPolicy aplica a política de senha: 12+ caracteres, fora da lista de senhas comuns,
// e sem conter a parte local do e-mail.
func CheckPolicy(pw, email string) error {
	if utf8.RuneCountInString(pw) < minPwLen {
		return fmt.Errorf("a senha precisa ter pelo menos %d caracteres", minPwLen)
	}
	low := strings.ToLower(pw)
	if isCommonPassword(low) {
		return errors.New("essa senha é muito comum; escolha outra")
	}
	if local, _, ok := strings.Cut(strings.ToLower(email), "@"); ok && len(local) >= 4 && strings.Contains(low, local) {
		return errors.New("a senha não pode conter o seu e-mail")
	}
	if strings.Count(low, string([]rune(low)[0])) == utf8.RuneCountInString(low) {
		return errors.New("a senha não pode ser um único caractere repetido")
	}
	return nil
}
