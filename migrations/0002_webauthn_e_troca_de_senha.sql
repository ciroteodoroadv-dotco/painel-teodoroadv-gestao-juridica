-- +goose Up
-- Classificação LGPD: a credencial WebAuthn guarda só a CHAVE PÚBLICA (não é segredo), id e contador.
CREATE TABLE webauthn_credentials (
    id           bytea PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         text NOT NULL DEFAULT 'Chave de segurança',
    credential   jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX webauthn_credentials_user_idx ON webauthn_credentials (user_id);

-- Senha provisória (criada ou redefinida por administrador) obriga a troca no próximo acesso.
ALTER TABLE users ADD COLUMN must_change_password boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN must_change_password;
DROP TABLE webauthn_credentials;
