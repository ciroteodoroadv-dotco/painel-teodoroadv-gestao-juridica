package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type txLike = pgx.Tx

// UserRow é a linha da tela de usuários (sem hash nem segredo).
type UserRow struct {
	ID, Email string
	Role      Role
	Active    bool
	Locked    bool
	HasMFA    bool
	Temporary bool
}

// ListUsers devolve todos os usuários para o administrador.
func (s *Store) ListUsers(ctx context.Context, now time.Time) ([]UserRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT u.id, u.email, u.role, u.active,
		(u.locked_until IS NOT NULL AND u.locked_until > $1),
		(u.totp_confirmed_at IS NOT NULL OR EXISTS (SELECT 1 FROM webauthn_credentials w WHERE w.user_id=u.id)),
		u.must_change_password FROM users u ORDER BY lower(u.email)`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserRow
	for rows.Next() {
		var r UserRow
		var role string
		if err := rows.Scan(&r.ID, &r.Email, &role, &r.Active, &r.Locked, &r.HasMFA, &r.Temporary); err != nil {
			return nil, err
		}
		r.Role = Role(role)
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateRole troca o papel e encerra as sessões do usuário (para o novo papel valer já).
func (s *Store) UpdateRole(ctx context.Context, tx pgx.Tx, id string, role Role) error {
	tag, err := tx.Exec(ctx, `UPDATE users SET role=$2, updated_at=now(), version=version+1 WHERE id=$1`, id, string(role))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id)
	return err
}

// SetActive ativa ou desativa o usuário; desativar encerra as sessões.
func (s *Store) SetActive(ctx context.Context, tx pgx.Tx, id string, active bool) error {
	tag, err := tx.Exec(ctx, `UPDATE users SET active=$2, updated_at=now(), version=version+1 WHERE id=$1`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !active {
		_, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id)
	}
	return err
}

// UserOptions lista usuários ativos para seletores de responsável (id → e-mail).
func (s *Store) UserOptions(ctx context.Context) ([]UserOption, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, email FROM users WHERE active ORDER BY lower(email)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserOption
	for rows.Next() {
		var o UserOption
		if err := rows.Scan(&o.ID, &o.Email); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// UserEmails devolve id → e-mail.
func (s *Store) UserEmails(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, email FROM users WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[id] = email
	}
	return out, rows.Err()
}

// UserOption é um usuário ativo para seleção.
type UserOption struct{ ID, Email string }
