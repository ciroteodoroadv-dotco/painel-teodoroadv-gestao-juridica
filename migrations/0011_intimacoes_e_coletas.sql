-- +goose Up
-- Plano 5 da fusão: Intimações (DJEN) e estado das coletas.
-- Classificação LGPD por coluna:
--   oab_monitoradas.advogado_nome = PESSOAL (nome de advogado parceiro); numero_oab, uf = COMUM; autorizacao_base, autorizacao_em = COMUM
--     (base da autorização e data em que o titular do escritório autorizou a coleta pela OAB; quem cadastrou fica em created_by).
--   intimacoes.resumo, orgao, processo_numero = PESSOAL (o texto publicado pode citar partes); hash, tipo, status,
--     tribunal, data_publicacao, link = COMUM. O resumo é cortado em 2000 caracteres.
--   coletas.mensagem = COMUM (só contagens e códigos de erro; nunca texto de intimação).
-- A integração com o DJEN vem DESLIGADA por padrão (PAINEL_DJEN_ATIVO=false).

CREATE TABLE oab_monitoradas (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    advogado_nome    text NOT NULL CHECK (length(btrim(advogado_nome)) BETWEEN 2 AND 200),
    numero_oab       text NOT NULL CHECK (numero_oab ~ '^[0-9]{1,10}$'),
    uf               char(2) NOT NULL CHECK (uf ~ '^[A-Z]{2}$'),
    autorizacao_base text NOT NULL CHECK (length(btrim(autorizacao_base)) BETWEEN 3 AND 500),
    autorizacao_em   date NOT NULL,
    ativa            boolean NOT NULL DEFAULT true,
    ultimo_sucesso   timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    created_by       uuid NOT NULL REFERENCES users (id),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       uuid NOT NULL REFERENCES users (id),
    version          integer NOT NULL DEFAULT 1,
    UNIQUE (numero_oab, uf)
);

CREATE TABLE intimacoes (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hash            text NOT NULL UNIQUE CHECK (length(hash) > 0),
    origem          text NOT NULL DEFAULT 'DJEN',
    oab_id          uuid REFERENCES oab_monitoradas (id),
    tribunal        text NOT NULL DEFAULT '',
    orgao           text NOT NULL DEFAULT '',
    processo_numero text NOT NULL DEFAULT '',
    processo_id     uuid REFERENCES processos (id),
    tipo            text NOT NULL DEFAULT '',
    data_publicacao date,
    resumo          text NOT NULL DEFAULT '' CHECK (char_length(resumo) <= 2000),
    link            text NOT NULL DEFAULT '' CHECK (link = '' OR link ~ '^https://'),
    status          text NOT NULL DEFAULT 'triagem' CHECK (status IN ('triagem', 'confirmada', 'descartada')),
    decidida_por    uuid REFERENCES users (id),
    decidida_em     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (status = 'triagem' OR decidida_por IS NOT NULL)
);
CREATE INDEX intimacoes_data_idx ON intimacoes (data_publicacao DESC, created_at DESC);

CREATE TABLE coletas (
    modulo          text PRIMARY KEY,
    origem          text NOT NULL DEFAULT '',
    ultima_tentativa timestamptz,
    ultimo_sucesso  timestamptz,
    status          text NOT NULL CHECK (status IN ('sucesso', 'parcial', 'falha', 'bloqueado', 'nao_sincronizado')),
    mensagem        text NOT NULL DEFAULT '',
    janela_desde    date
);

-- A tarefa de análise nasce da intimação: vínculo próprio (origem 'intimacao') ou o processo do v2 quando o número existe.
ALTER TABLE tarefas ADD COLUMN intimacao_id uuid REFERENCES intimacoes (id);
CREATE UNIQUE INDEX tarefas_intimacao_uq ON tarefas (intimacao_id) WHERE intimacao_id IS NOT NULL;
ALTER TABLE tarefas DROP CONSTRAINT tarefas_origem_tipo_check;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_origem_tipo_check CHECK (origem_tipo IN ('atendimento', 'processo', 'intimacao'));
ALTER TABLE tarefas DROP CONSTRAINT tarefas_check;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_vinculo_ck CHECK (
       (origem_tipo = 'atendimento' AND atendimento_id IS NOT NULL AND processo_id IS NULL AND intimacao_id IS NULL)
    OR (origem_tipo = 'processo' AND processo_id IS NOT NULL AND atendimento_id IS NULL)
    OR (origem_tipo = 'intimacao' AND intimacao_id IS NOT NULL AND atendimento_id IS NULL AND processo_id IS NULL));

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM tarefas WHERE intimacao_id IS NOT NULL) OR EXISTS (SELECT 1 FROM intimacoes)
       OR EXISTS (SELECT 1 FROM oab_monitoradas) THEN
        RAISE EXCEPTION 'migração 0011 não é reversível com dados: esvazie intimacoes e oab_monitoradas antes (coletas é só estado e pode ser descartada)';
    END IF;
END
$$;
-- +goose StatementEnd
ALTER TABLE tarefas DROP CONSTRAINT tarefas_vinculo_ck;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_check CHECK (
       (origem_tipo = 'atendimento' AND atendimento_id IS NOT NULL AND processo_id IS NULL)
    OR (origem_tipo = 'processo' AND processo_id IS NOT NULL AND atendimento_id IS NULL));
ALTER TABLE tarefas DROP CONSTRAINT tarefas_origem_tipo_check;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_origem_tipo_check CHECK (origem_tipo IN ('atendimento', 'processo'));
DROP INDEX tarefas_intimacao_uq;
ALTER TABLE tarefas DROP COLUMN intimacao_id;
DROP TABLE coletas;
DROP TABLE intimacoes;
DROP TABLE oab_monitoradas;
