-- +goose Up
-- O app roda com o papel "painel_app", SEM poder de dono: não altera esquema, não desliga gatilho
-- e, na auditoria, só consegue INSERT e SELECT. A senha do papel é definida pelo app no início
-- (db.SetAppPassword), a partir de um segredo — nunca fica no código nem no Git.

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'painel_app') THEN
        CREATE ROLE painel_app LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
    END IF;
END
$$;
-- +goose StatementEnd

GRANT USAGE ON SCHEMA public TO painel_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO painel_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO painel_app;

-- Tabelas futuras (M3 em diante) já nascem acessíveis ao app; tabelas append-only precisam de REVOKE explícito.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO painel_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO painel_app;

REVOKE UPDATE, DELETE, TRUNCATE ON audit_events FROM painel_app;
REVOKE ALL ON goose_db_version FROM painel_app;

-- +goose Down
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM painel_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM painel_app;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM painel_app;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM painel_app;
REVOKE ALL ON SCHEMA public FROM painel_app;
DROP ROLE IF EXISTS painel_app;
