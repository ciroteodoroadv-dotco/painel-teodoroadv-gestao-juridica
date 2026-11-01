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
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// CheckPolicy aplica a política de senha: 12+ caracteres, fora da lista de senhas comuns,
// e sem conter a parte local do e-mail.
func CheckPolicy(pw, email string) error {
	if utf8.RuneCountInString(pw) < minPwLen {
		return fmt.Errorf("a senha precisa ter pelo menos %d caracteres", minPwLen)
	}
	low := strings.ToLower(pw)
	if commonPasswords[low] {
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
