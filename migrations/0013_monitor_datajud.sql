-- +goose Up
-- Plano 6 da fusão: Monitor DataJud, movimentos, processos em segredo de justiça e tarefas automáticas.
-- Classificação LGPD por coluna:
--   monitor_processos.mensagem = COMUM (só contagens e códigos de erro); tribunal_alias, status, datas = COMUM.
--   processo_movimentos.nome, complemento, complementos, orgao_julgador = PESSOAL (o movimento pode citar parte);
--     grau, codigo, chave, fonte, data = COMUM. O complemento resumido (usado na chave) fica em 500 caracteres.
--   monitor_sigilosos = COMUM (liga processo cadastrado a OAB cadastrada).
--   intimacoes.texto = PESSOAL (texto integral da comunicação; teto de segurança de 100.000 caracteres + marca de corte).
-- A integração com o DataJud vem DESLIGADA por padrão (PAINEL_DATAJUD_ATIVO=false).

CREATE TABLE monitor_processos (
    processo_id         uuid PRIMARY KEY REFERENCES processos (id),
    tribunal_alias      text NOT NULL DEFAULT '',
    status              text NOT NULL DEFAULT 'nao_sincronizado' CHECK (status IN ('sucesso', 'parcial', 'falha', 'bloqueado', 'nao_sincronizado')),
    mensagem            text NOT NULL DEFAULT '' CHECK (char_length(mensagem) <= 500),
    ultima_consulta     timestamptz,
    ultimo_sucesso      timestamptz,
    base_registrada_em  timestamptz
);

CREATE TABLE processo_movimentos (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    processo_id      uuid NOT NULL REFERENCES processos (id),
    fonte            text NOT NULL CHECK (fonte IN ('DataJud', 'DJEN')),
    grau             text NOT NULL DEFAULT '',
    orgao_julgador   text NOT NULL DEFAULT '',
    chave            text NOT NULL CHECK (length(chave) > 0),
    data             date,
    data_hora        timestamptz,
    codigo           text NOT NULL DEFAULT '',
    nome             text NOT NULL,
    complemento      text NOT NULL DEFAULT '' CHECK (char_length(complemento) <= 500),
    complementos     jsonb NOT NULL DEFAULT '[]'::jsonb,
    base             boolean NOT NULL DEFAULT false,
    intimacao_id     uuid REFERENCES intimacoes (id),
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (processo_id, fonte, chave)
);
CREATE INDEX processo_movimentos_proc_idx ON processo_movimentos (processo_id, data DESC);

CREATE TABLE monitor_sigilosos (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    processo_id uuid NOT NULL UNIQUE REFERENCES processos (id),
    oab_id      uuid NOT NULL REFERENCES oab_monitoradas (id),
    ativo       boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  uuid NOT NULL REFERENCES users (id),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    updated_by  uuid NOT NULL REFERENCES users (id),
    version     integer NOT NULL DEFAULT 1
);

-- Intimações: origem restrita; texto integral (E2 do Plano 6). Migra o que já existe sem perda: texto = resumo.
-- oab_id já é opcional desde a 0011 (DataJud não tem OAB).
ALTER TABLE intimacoes ADD CONSTRAINT intimacoes_origem_ck CHECK (origem IN ('DJEN', 'DataJud', 'DJEN (sigiloso)'));
ALTER TABLE intimacoes ADD COLUMN texto text NOT NULL DEFAULT '' CHECK (char_length(texto) <= 100100);
UPDATE intimacoes SET texto = resumo WHERE texto = '';

-- Tarefas automáticas: monitoramento (uma aberta por processo), consolidada do dia (uma por processo e dia) e prazo interno.
ALTER TABLE tarefas ADD COLUMN monitoramento boolean NOT NULL DEFAULT false;
ALTER TABLE tarefas ADD COLUMN consolidada_dia date;
ALTER TABLE tarefas ADD COLUMN prazo_interno boolean NOT NULL DEFAULT false;
-- Fonte da tarefa de análise (mostrada na Origem da lista de tarefas): vazio = tarefa comum.
ALTER TABLE tarefas ADD COLUMN fonte_automatica text NOT NULL DEFAULT '' CHECK (fonte_automatica IN ('', 'DJEN', 'DataJud', 'DJEN (sigiloso)'));
CREATE UNIQUE INDEX tarefas_monitoramento_aberta_uq ON tarefas (processo_id)
    WHERE monitoramento AND estado NOT IN ('concluidas-com-sucesso', 'concluidas-sem-sucesso', 'canceladas');
CREATE UNIQUE INDEX tarefas_consolidada_uq ON tarefas (processo_id, consolidada_dia) WHERE consolidada_dia IS NOT NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM monitor_processos) OR EXISTS (SELECT 1 FROM processo_movimentos) OR EXISTS (SELECT 1 FROM monitor_sigilosos)
       OR EXISTS (SELECT 1 FROM tarefas WHERE monitoramento OR consolidada_dia IS NOT NULL OR prazo_interno OR fonte_automatica <> '')
       OR EXISTS (SELECT 1 FROM intimacoes WHERE origem <> 'DJEN' OR char_length(texto) > 2000) THEN
        RAISE EXCEPTION 'migração 0013 não é reversível com dados: nada se apaga (esvazie o monitor e as tarefas automáticas antes)';
    END IF;
END
$$;
-- +goose StatementEnd
DROP INDEX tarefas_consolidada_uq;
DROP INDEX tarefas_monitoramento_aberta_uq;
ALTER TABLE tarefas DROP COLUMN fonte_automatica;
ALTER TABLE tarefas DROP COLUMN prazo_interno;
ALTER TABLE tarefas DROP COLUMN consolidada_dia;
ALTER TABLE tarefas DROP COLUMN monitoramento;
ALTER TABLE intimacoes DROP COLUMN texto;
ALTER TABLE intimacoes DROP CONSTRAINT intimacoes_origem_ck;
DROP TABLE monitor_sigilosos;
DROP TABLE processo_movimentos;
DROP TABLE monitor_processos;
