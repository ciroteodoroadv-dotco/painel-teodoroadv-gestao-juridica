package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/hotp"
	"github.com/pquerna/otp/totp"

	"painelteodoro/internal/platform/crypt"
)

const (
	issuer        = "Painel Teodoro"
	totpPeriod    = 30
	recoveryCount = 8
)

// ErrCodigoUsado indica código TOTP já consumido (reuso): é erro da pessoa, não do sistema.
var ErrCodigoUsado = errors.New("código já utilizado")

var otpOpts = hotp.ValidateOpts{Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}

// NewTOTPKey cria um segredo novo para o e-mail.
func NewTOTPKey(email string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{Issuer: issuer, AccountName: email})
}

// CheckTOTP valida o código aceitando ±1 janela e rejeita passo já usado (anti-reuso).
// Devolve o passo aceito para ser gravado.
func CheckTOTP(secret, code string, lastStep int64, now time.Time) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	cur := now.Unix() / totpPeriod
	for step := cur - 1; step <= cur+1; step++ {
		if step <= lastStep {
			continue
		}
		// #nosec G115 -- step = unix/30, sempre positivo
		if ok, err := hotp.ValidateCustom(code, uint64(step), secret, otpOpts); err == nil && ok {
			return step, true
		}
	}
	return 0, false
}

// Formato do segredo TOTP cifrado (auditoria 05/11, item 2.7). Versão 1: byte 0x01 || AES-GCM com a SUBCHAVE
// "auth.totp" (não a chave mestra) e o id do usuário como dado associado. Sem versão (legado): AES-GCM com a chave
// mestra, sem dado associado — continua sendo lido; é regravado no formato novo quando o 2º fator é redefinido.
const versaoTOTP byte = 1

func chaveTOTP(master []byte) []byte { return crypt.Derive(master, "auth.totp") }

// SaveTOTPSecret grava o segredo cifrado (ainda não confirmado).
func (s *Store) SaveTOTPSecret(ctx context.Context, fieldKey []byte, userID, secret string) error {
	sealed, err := crypt.SealAD(chaveTOTP(fieldKey), []byte(secret), []byte(userID))
	if err != nil {
		return err
	}
	enc := append([]byte{versaoTOTP}, sealed...)
	_, err = s.Pool.Exec(ctx, `UPDATE users SET totp_secret_enc=$2, totp_confirmed_at=NULL, totp_last_step=0 WHERE id=$1`, userID, enc)
	return err
}

// TOTPSecret decifra o segredo do usuário.
func TOTPSecret(fieldKey []byte, u *User) (string, error) {
	if len(u.TOTPSecretEnc) == 0 {
		return "", ErrNotFound
	}
	enc := u.TOTPSecretEnc
	if enc[0] == versaoTOTP {
		if b, err := crypt.OpenAD(chaveTOTP(fieldKey), enc[1:], []byte(u.ID)); err == nil {
			return string(b), nil
		}
	}
	b, err := crypt.Open(fieldKey, enc) // legado (o 1º byte do nonce aleatório pode valer 0x01: por isso a segunda tentativa)
	return string(b), err
}

// ConfirmTOTP marca o 2FA como ativo e grava o passo usado.
func (s *Store) ConfirmTOTP(ctx context.Context, tx pgx.Tx, userID string, step int64) error {
	_, err := tx.Exec(ctx, `UPDATE users SET totp_confirmed_at=now(), totp_last_step=$2 WHERE id=$1`, userID, step)
	return err
}

// UseTOTPStep grava o passo aceito; falha se outra requisição já o usou (corrida).
func (s *Store) UseTOTPStep(ctx context.Context, tx pgx.Tx, userID string, step int64) error {
	tag, err := tx.Exec(ctx, `UPDATE users SET totp_last_step=$2 WHERE id=$1 AND totp_last_step<$2`, userID, step)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrCodigoUsado
	}
	return err
}

func hashCode(c string) []byte {
	h := sha256.Sum256([]byte(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c), "-", ""))))
	return h[:]
}

// NewRecoveryCodes substitui os códigos do usuário por 8 novos e devolve o texto claro (mostrado uma vez).
func (s *Store) NewRecoveryCodes(ctx context.Context, tx pgx.Tx, userID string) ([]string, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM recovery_codes WHERE user_id=$1`, userID); err != nil {
		return nil, err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	codes := make([]string, recoveryCount)
	for i := range codes {
		b := make([]byte, 7)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		raw := strings.ToLower(enc.EncodeToString(b))[:10]
		codes[i] = raw[:5] + "-" + raw[5:]
		if _, err := tx.Exec(ctx, `INSERT INTO recovery_codes (user_id, code_hash) VALUES ($1,$2)`, userID, hashCode(codes[i])); err != nil {
			return nil, err
		}
	}
	return codes, nil
}

// UseRecoveryCode consome um código de uso único. Devolve false se não existir ou já foi usado.
func (s *Store) UseRecoveryCode(ctx context.Context, tx pgx.Tx, userID, code string) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE recovery_codes SET used_at=now()
		WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL`, userID, hashCode(code))
	return err == nil && tag.RowsAffected() == 1, err
}
