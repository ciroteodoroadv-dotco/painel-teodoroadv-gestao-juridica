package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound indica usuário ou sessão inexistente.
var ErrNotFound = errors.New("não encontrado")

// User espelha a tabela users.
type User struct {
	ID, Email, PasswordHash string
	Role                    Role
	TOTPSecretEnc           []byte
	TOTPConfirmed           bool
	TOTPLastStep            int64
	FailedAttempts          int
	LockedUntil             *time.Time
	Active                  bool
}

// Store é o repositório de identidade.
type Store struct{ Pool *pgxpool.Pool }

const userCols = `id, email, role, password_hash, totp_secret_enc, totp_confirmed_at IS NOT NULL,
	totp_last_step, failed_attempts, locked_until, active`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	var role string
	err := row.Scan(&u.ID, &u.Email, &role, &u.PasswordHash, &u.TOTPSecretEnc, &u.TOTPConfirmed,
		&u.TOTPLastStep, &u.FailedAttempts, &u.LockedUntil, &u.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	u.Role = Role(role)
	return &u, err
}

// UserByEmail busca sem diferenciar maiúsculas.
func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email)=lower($1)`, strings.TrimSpace(email)))
}

// UserByID busca por id.
func (s *Store) UserByID(ctx context.Context, id string) (*User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

// CreateUser insere um usuário e devolve o id. mustChange marca a senha como provisória.
func (s *Store) CreateUser(ctx context.Context, tx pgx.Tx, email string, role Role, hash string, mustChange bool) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO users (email, role, password_hash, must_change_password) VALUES ($1,$2,$3,$4) RETURNING id`,
		strings.TrimSpace(email), string(role), hash, mustChange).Scan(&id)
	return id, err
}

// SetPassword troca a senha, zera bloqueios e revoga todas as sessões do usuário.
func (s *Store) SetPassword(ctx context.Context, tx pgx.Tx, userID, hash string, mustChange bool) error {
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash=$2, failed_attempts=0, locked_until=NULL,
		must_change_password=$3, updated_at=now(), version=version+1 WHERE id=$1`, userID, hash, mustChange); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID)
	return err
}

// Unlock remove o bloqueio por tentativas.
func (s *Store) Unlock(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `UPDATE users SET failed_attempts=0, locked_until=NULL, updated_at=now(),
		version=version+1 WHERE id=$1`, userID)
	return err
}

// LockSchedule traduz o total de falhas em bloqueio progressivo: 1 min, 5 min, 30 min e,
// a partir de 20 falhas, bloqueio até o administrador liberar (permanent).
func LockSchedule(failed int) (d time.Duration, permanent bool) {
	if failed == 0 || failed%5 != 0 {
		return 0, false
	}
	switch {
	case failed >= 20:
		return 0, true
	case failed >= 15:
		return 30 * time.Minute, false
	case failed >= 10:
		return 5 * time.Minute, false
	}
	return time.Minute, false
}

// RegisterFailure soma uma falha e aplica o bloqueio devido. Devolve o total de falhas.
func (s *Store) RegisterFailure(ctx context.Context, userID string, now time.Time) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `UPDATE users SET failed_attempts=failed_attempts+1 WHERE id=$1
		RETURNING failed_attempts`, userID).Scan(&n)
	if err != nil {
		return 0, err
	}
	d, perm := LockSchedule(n)
	switch {
	case perm:
		_, err = s.Pool.Exec(ctx, `UPDATE users SET locked_until='infinity' WHERE id=$1`, userID)
	case d > 0:
		_, err = s.Pool.Exec(ctx, `UPDATE users SET locked_until=$2 WHERE id=$1`, userID, now.Add(d))
	}
	return n, err
}

// ResetFailures zera o contador após login completo.
func (s *Store) ResetFailures(ctx context.Context, userID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET failed_attempts=0, locked_until=NULL WHERE id=$1 AND failed_attempts>0`, userID)
	return err
}

// IsLocked informa se o usuário está bloqueado agora.
func (u *User) IsLocked(now time.Time) bool { return u.LockedUntil != nil && u.LockedUntil.After(now) }
