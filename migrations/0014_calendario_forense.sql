-- +goose Up
-- Calendário forense (Plano 7). Cada dia que entra no cálculo de prazo tem FONTE citada (norma/portaria), quem
-- verificou e quando. SEM FONTE NÃO GRAVA (CHECK abaixo). Nada se apaga: dia errado é INATIVADO com motivo.
CREATE TABLE calendarios (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sigla      text NOT NULL UNIQUE CHECK (length(btrim(sigla)) BETWEEN 2 AND 30),
    nome       text NOT NULL CHECK (length(btrim(nome)) BETWEEN 2 AND 120),
    ambito     text NOT NULL CHECK (ambito IN ('nacional', 'federal', 'estadual', 'trabalhista', 'eleitoral', 'outro')),
    verificar  boolean NOT NULL DEFAULT true, -- true = ainda marcado [VERIFICAR] (advogado não validou)
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid REFERENCES users (id)     -- NULL = semente do sistema
);

CREATE TABLE calendario_dias (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    calendario_id       uuid NOT NULL REFERENCES calendarios (id),
    data                date NOT NULL,
    tipo                text NOT NULL CHECK (tipo IN ('feriado', 'suspensao_prazo', 'recesso', 'sem_expediente', 'ponto_facultativo')),
    conta_como_nao_util boolean NOT NULL,
    descricao           text NOT NULL CHECK (length(btrim(descricao)) BETWEEN 2 AND 200),
    fonte_citacao       text NOT NULL CHECK (length(btrim(fonte_citacao)) >= 3),
    fonte_url           text NOT NULL DEFAULT '' CHECK (length(fonte_url) <= 500),
    observacao          text NOT NULL DEFAULT '' CHECK (length(observacao) <= 500),
    verificado_por      uuid REFERENCES users (id),
    verificado_em       timestamptz,
    ativo               boolean NOT NULL DEFAULT true,
    motivo_inativacao   text NOT NULL DEFAULT '',
    inativado_por       uuid REFERENCES users (id),
    inativado_em        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    created_by          uuid REFERENCES users (id), -- NULL = semente do sistema
    -- Ponto facultativo só conta como não útil com fonte do tribunal E alguém que verificou.
    CHECK (tipo <> 'ponto_facultativo' OR NOT conta_como_nao_util OR verificado_por IS NOT NULL),
    CHECK (ativo OR length(btrim(motivo_inativacao)) >= 3)
);
-- Um dia ATIVO por calendário, data e tipo; o inativado permanece listado e libera o cadastro correto.
CREATE UNIQUE INDEX calendario_dias_uq ON calendario_dias (calendario_id, data, tipo) WHERE ativo;
CREATE INDEX calendario_dias_data_idx ON calendario_dias (calendario_id, data);

-- +goose StatementBegin
CREATE FUNCTION calendario_sem_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'nada se apaga: inative o dia do calendário com o motivo';
END
$$;
-- +goose StatementEnd
CREATE TRIGGER calendario_dias_sem_delete BEFORE DELETE ON calendario_dias FOR EACH ROW EXECUTE FUNCTION calendario_sem_delete();
CREATE TRIGGER calendarios_sem_delete BEFORE DELETE ON calendarios FOR EACH ROW EXECUTE FUNCTION calendario_sem_delete();
REVOKE DELETE ON calendario_dias, calendarios FROM painel_app;

-- Prazo criado a partir da Calculadora: fonte própria, para nunca se confundir com prazo digitado ou do tribunal.
ALTER TABLE prazos DROP CONSTRAINT prazos_fonte_check;
ALTER TABLE prazos ADD CONSTRAINT prazos_fonte_check CHECK (fonte IN ('manual', 'djen', 'datajud', 'outra', 'calculadora'));

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM calendario_dias WHERE created_by IS NOT NULL OR verificado_por IS NOT NULL)
       OR EXISTS (SELECT 1 FROM calendarios WHERE created_by IS NOT NULL) THEN
        RAISE EXCEPTION 'migração 0014 não é reversível com dados: nada se apaga (calendários cadastrados por pessoas)';
    END IF;
    IF EXISTS (SELECT 1 FROM prazos WHERE fonte = 'calculadora') THEN
        RAISE EXCEPTION 'migração 0014 não é reversível: há prazo criado pela Calculadora (nada se apaga)';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE prazos DROP CONSTRAINT prazos_fonte_check;
ALTER TABLE prazos ADD CONSTRAINT prazos_fonte_check CHECK (fonte IN ('manual', 'djen', 'datajud', 'outra'));
DROP TABLE calendario_dias;
DROP TABLE calendarios;
DROP FUNCTION calendario_sem_delete();
