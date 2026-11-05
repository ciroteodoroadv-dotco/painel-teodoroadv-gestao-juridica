package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Limites de sessão da especificação.
const (
	IdleTimeout     = 30 * time.Minute
	AbsoluteTimeout = 10 * time.Hour
)

// Session é a sessão válida já carregada junto com o usuário.
type Session struct {
	Token         string // valor do cookie (nunca vai ao banco)
	UserID, Email string
	Role          Role
	MFAVerified   bool
	MustChange    bool // senha provisória: só pode trocar a senha
}

// NewToken gera um identificador de 256 bits.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(t string) []byte { h := sha256.Sum256([]byte(t)); return h[:] }

// CreateSession abre uma sessão; só o hash do token vai ao banco.
func (s *Store) CreateSession(ctx context.Context, tx pgx.Tx, userID string, mfa bool, now time.Time) (string, error) {
	tok, err := NewToken()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `INSERT INTO sessions (id_hash, user_id, created_at, last_seen_at, expires_at, mfa_verified)
		VALUES ($1,$2,$3,$3,$4,$5)`, hashToken(tok), userID, now, now.Add(AbsoluteTimeout), mfa)
	return tok, err
}

// LookupSession valida o cookie: existe, não expirou (absoluto e ociosidade) e o usuário está ativo.
func (s *Store) LookupSession(ctx context.Context, token string, now time.Time) (*Session, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	var (
		ss                Session
		role              string
		lastSeen, expires time.Time
		active            bool
	)
	err := s.Pool.QueryRow(ctx, `SELECT s.user_id, u.email, u.role, s.mfa_verified, s.last_seen_at,
		s.expires_at, u.active, u.must_change_password FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id_hash=$1`,
		hashToken(token)).Scan(&ss.UserID, &ss.Email, &role, &ss.MFAVerified, &lastSeen, &expires, &active, &ss.MustChange)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !active || !now.Before(expires) || now.Sub(lastSeen) > IdleTimeout {
		_ = s.DeleteSession(ctx, token)
		return nil, ErrNotFound
	}
	if now.Sub(lastSeen) > 30*time.Second {
		_, _ = s.Pool.Exec(ctx, `UPDATE sessions SET last_seen_at=$2 WHERE id_hash=$1`, hashToken(token), now)
	}
	ss.Token, ss.Role = token, Role(role)
	return &ss, nil
}

// DeleteSession encerra uma sessão (logout).
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE id_hash=$1`, hashToken(token))
	return err
}

// PromoteSession troca a sessão pendente por uma nova, já com 2FA — evita fixação de sessão.
func (s *Store) PromoteSession(ctx context.Context, tx pgx.Tx, oldToken, userID string, now time.Time) (string, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE id_hash=$1`, hashToken(oldToken)); err != nil {
		return "", err
	}
	return s.CreateSession(ctx, tx, userID, true, now)
}

// ExpurgarSessoes apaga as sessões vencidas (prazo absoluto ou ociosidade). Antes elas só sumiam no logout ou quando o
// mesmo cookie voltava, e a tabela crescia sem fim (auditoria 05/11, item 4.4). Devolve quantas apagou.
func (s *Store) ExpurgarSessoes(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= $1 OR last_seen_at < $2`, now, now.Add(-IdleTimeout))
	return tag.RowsAffected(), err
}

// IniciarExpurgoDeSessoes roda ExpurgarSessoes a cada hora até ctx terminar.
func (a *Auth) IniciarExpurgoDeSessoes(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			if n, err := a.Store.ExpurgarSessoes(ctx, a.now()); err != nil && ctx.Err() == nil {
				a.Log.Error("expurgo de sessões falhou")
			} else if n > 0 {
				a.Log.Info("sessões vencidas apagadas", "quantidade", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}
