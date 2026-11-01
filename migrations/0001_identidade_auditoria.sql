-- +goose Up
-- Classificação LGPD: users.email = pessoal; password_hash/totp_secret_enc/recovery = segredo (nunca em log).
CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             text NOT NULL,
    role              text NOT NULL CHECK (role IN ('administrador', 'advogado', 'assistente')),
    password_hash     text NOT NULL,
    totp_secret_enc   bytea,
    totp_confirmed_at timestamptz,
    totp_last_step    bigint NOT NULL DEFAULT 0, -- impede reuso do mesmo código TOTP
    failed_attempts   integer NOT NULL DEFAULT 0,
    locked_until      timestamptz,
    active            boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    version           integer NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX users_email_lower_uq ON users (lower(email));

CREATE TABLE sessions (
    id_hash      bytea PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    mfa_verified boolean NOT NULL DEFAULT false
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE recovery_codes (
    id        bigserial PRIMARY KEY,
    user_id   uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash bytea NOT NULL,
    used_at   timestamptz
);
CREATE INDEX recovery_codes_user_idx ON recovery_codes (user_id);

CREATE TABLE audit_events (
    seq         bigserial PRIMARY KEY,
    at          timestamptz NOT NULL,
    actor       text NOT NULL,
    action      text NOT NULL,
    target_type text NOT NULL DEFAULT '',
    target_id   text NOT NULL DEFAULT '',
    result      text NOT NULL,
    request_id  text NOT NULL DEFAULT '',
    meta        jsonb NOT NULL DEFAULT '{}',
    prev_hash   bytea NOT NULL,
    hash        bytea NOT NULL
);

-- +goose StatementBegin
CREATE FUNCTION audit_events_imutavel() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_events é append-only: % bloqueado', TG_OP;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER audit_events_sem_update_delete
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_imutavel();
CREATE TRIGGER audit_events_sem_truncate
    BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION audit_events_imutavel();

-- +goose Down
DROP TABLE audit_events;
DROP FUNCTION audit_events_imutavel();
DROP TABLE recovery_codes;
DROP TABLE sessions;
DROP TABLE users;
