package auth

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"
)

// KeyInfo é o que a tela mostra de cada chave cadastrada.
type KeyInfo struct {
	IDHex     string
	Name      string
	CreatedAt time.Time
	LastUsed  *time.Time
}

// Credentials devolve as credenciais WebAuthn do usuário.
func (s *Store) Credentials(ctx context.Context, userID string) ([]webauthn.Credential, error) {
	rows, err := s.Pool.Query(ctx, `SELECT credential FROM webauthn_credentials WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []webauthn.Credential
	for rows.Next() {
		var raw []byte
		var c webauthn.Credential
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Keys lista as chaves para exibição.
func (s *Store) Keys(ctx context.Context, userID string) ([]KeyInfo, error) {
	rows, err := s.Pool.Query(ctx, `SELECT encode(id,'hex'), name, created_at, last_used_at
		FROM webauthn_credentials WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KeyInfo
	for rows.Next() {
		var k KeyInfo
		if err := rows.Scan(&k.IDHex, &k.Name, &k.CreatedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// AddCredential grava uma credencial nova.
func (s *Store) AddCredential(ctx context.Context, tx pgx.Tx, userID, name string, c *webauthn.Credential) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO webauthn_credentials (id, user_id, name, credential) VALUES ($1,$2,$3,$4)`,
		c.ID, userID, name, raw)
	return err
}

// TouchCredential atualiza contador de assinaturas e último uso após um login.
func (s *Store) TouchCredential(ctx context.Context, tx pgx.Tx, userID string, c *webauthn.Credential, now time.Time) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE webauthn_credentials SET credential=$3, last_used_at=$4 WHERE id=$1 AND user_id=$2`,
		c.ID, userID, raw, now)
	return err
}

// DeleteCredential remove uma chave do próprio usuário (id em hexadecimal).
func (s *Store) DeleteCredential(ctx context.Context, tx pgx.Tx, userID, idHex string) (bool, error) {
	tag, err := tx.Exec(ctx, `DELETE FROM webauthn_credentials WHERE user_id=$1 AND id=decode($2,'hex')`, userID, idHex)
	return err == nil && tag.RowsAffected() == 1, err
}

// waUser adapta User à interface da biblioteca.
type waUser struct {
	u     *User
	creds []webauthn.Credential
}

func (w waUser) WebAuthnID() []byte                         { return []byte(w.u.ID) }
func (w waUser) WebAuthnName() string                       { return w.u.Email }
func (w waUser) WebAuthnDisplayName() string                { return w.u.Email }
func (w waUser) WebAuthnCredentials() []webauthn.Credential { return w.creds }

// ceremonies guarda, na memória, os dados de cada cerimônia entre "begin" e "finish" (5 min).
// Limite conhecido: instância única; um reinício obriga a repetir a cerimônia.
type ceremonies struct {
	mu sync.Mutex
	m  map[string]ceremony
}

type ceremony struct {
	data *webauthn.SessionData
	till time.Time
}

func (c *ceremonies) put(key string, d *webauthn.SessionData, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]ceremony{}
	}
	for k, v := range c.m {
		if now.After(v.till) {
			delete(c.m, k)
		}
	}
	c.m[key] = ceremony{d, now.Add(5 * time.Minute)}
}

// take devolve e remove (uso único) os dados da cerimônia.
func (c *ceremonies) take(key string, now time.Time) (*webauthn.SessionData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	delete(c.m, key)
	if !ok || now.After(v.till) {
		return nil, errors.New("cerimônia expirada ou inexistente")
	}
	return v.data, nil
}

// flash guarda os códigos de recuperação entre o "finish" do JavaScript e a página que os exibe (uso único).
type flash struct {
	mu sync.Mutex
	m  map[string][]string
}

func (f *flash) put(key string, codes []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.m == nil {
		f.m = map[string][]string{}
	}
	f.m[key] = codes
}

func (f *flash) take(key string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.m[key]
	delete(f.m, key)
	return c
}
