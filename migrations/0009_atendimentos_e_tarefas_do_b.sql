-- +goose Up
-- Plano 3 da fusão: Atendimentos e Tarefas com os campos do painel B.
-- Classificação LGPD por coluna:
--   atendimentos.descricao, observacoes = PESSOAL (podem citar fatos e nomes); codigo, tipo, status,
--     data_abertura = COMUM. atendimento_pessoas/tarefa_partes/tarefa_responsaveis ligam pessoas (ids) = COMUM.
--   tarefas.titulo, subtitulo, descricao, observacoes = PESSOAL; estado, prioridade, datas = COMUM.
-- Exclusão é marcação (excluido_em). A tabela tarefas é refeita: o v2 começa limpo, e converter
-- dado real em silêncio seria perigoso — a migração recusa aplicar se houver linhas.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM tarefas) THEN
        RAISE EXCEPTION 'migração 0009: tarefas já tem linhas; ela só aplica sobre tabela vazia (converta os dados antes)';
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE atendimentos (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo                text NOT NULL,
    tipo                  text NOT NULL DEFAULT 'Extrajudicial' CHECK (tipo IN ('Extrajudicial', 'Triagem')),
    status                text NOT NULL DEFAULT 'Em triagem'
                          CHECK (status IN ('Em triagem', 'Em andamento', 'Concluído', 'Convertido em processo')),
    descricao             text NOT NULL CHECK (length(btrim(descricao)) > 0),
    observacoes           text NOT NULL DEFAULT '',
    data_abertura         date NOT NULL,
    criado_por_pessoa_id  uuid REFERENCES pessoas (id),
    excluido_em           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    created_by            uuid NOT NULL REFERENCES users (id),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    updated_by            uuid NOT NULL REFERENCES users (id),
    version               integer NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX atendimentos_codigo_uq ON atendimentos (codigo);
CREATE INDEX atendimentos_data_idx ON atendimentos (data_abertura DESC) WHERE excluido_em IS NULL;

CREATE TABLE atendimento_pessoas (
    atendimento_id uuid NOT NULL REFERENCES atendimentos (id) ON DELETE CASCADE,
    pessoa_id      uuid NOT NULL REFERENCES pessoas (id),
    papel          text NOT NULL CHECK (papel IN ('cliente', 'parte_adversa')),
    ordem          integer NOT NULL DEFAULT 0,
    PRIMARY KEY (atendimento_id, pessoa_id, papel)
);
CREATE INDEX atendimento_pessoas_pessoa_idx ON atendimento_pessoas (pessoa_id);

ALTER TABLE processos ADD CONSTRAINT processos_atendimento_origem_fk
    FOREIGN KEY (atendimento_origem_id) REFERENCES atendimentos (id);

-- ----------------------------------------------------------------- tarefas
DROP TABLE tarefas;
CREATE TABLE tarefas (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo                text NOT NULL,
    titulo                text NOT NULL CHECK (length(titulo) BETWEEN 2 AND 300),
    subtitulo             text NOT NULL DEFAULT '' CHECK (length(subtitulo) <= 80),
    estado                text NOT NULL DEFAULT 'pendente' CHECK (estado IN ('a-confirmar', 'pendente', 'em-execucao',
                          'revisao', 'concluidas-com-sucesso', 'concluidas-sem-sucesso', 'canceladas')),
    origem_tipo           text NOT NULL CHECK (origem_tipo IN ('atendimento', 'processo')),
    atendimento_id        uuid REFERENCES atendimentos (id),
    processo_id           uuid REFERENCES processos (id),
    tarefa_pai_id         uuid REFERENCES tarefas (id),
    criado_por_pessoa_id  uuid REFERENCES pessoas (id),
    descricao             text NOT NULL DEFAULT '',
    observacoes           text NOT NULL DEFAULT '',
    data_base             date,
    prazo                 date,
    prazo_fatal           date,
    prioridade            text NOT NULL DEFAULT 'Normal' CHECK (prioridade IN ('Urgente', 'Alta', 'Normal', 'Baixa')),
    excluido_em           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    created_by            uuid NOT NULL REFERENCES users (id),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    updated_by            uuid NOT NULL REFERENCES users (id),
    version               integer NOT NULL DEFAULT 1,
    CHECK ((origem_tipo = 'atendimento' AND atendimento_id IS NOT NULL AND processo_id IS NULL)
        OR (origem_tipo = 'processo' AND processo_id IS NOT NULL AND atendimento_id IS NULL))
);
CREATE UNIQUE INDEX tarefas_codigo_uq ON tarefas (codigo);
CREATE INDEX tarefas_estado_idx ON tarefas (estado) WHERE excluido_em IS NULL;
CREATE INDEX tarefas_processo_idx ON tarefas (processo_id);
CREATE INDEX tarefas_atendimento_idx ON tarefas (atendimento_id);

CREATE TABLE tarefa_partes (
    tarefa_id uuid NOT NULL REFERENCES tarefas (id) ON DELETE CASCADE,
    pessoa_id uuid NOT NULL REFERENCES pessoas (id),
    papel     text NOT NULL DEFAULT '',
    ordem     integer NOT NULL,
    PRIMARY KEY (tarefa_id, ordem)
);
CREATE TABLE tarefa_responsaveis (
    tarefa_id uuid NOT NULL REFERENCES tarefas (id) ON DELETE CASCADE,
    pessoa_id uuid NOT NULL REFERENCES pessoas (id),
    ordem     integer NOT NULL,
    PRIMARY KEY (tarefa_id, ordem)
);
CREATE INDEX tarefa_partes_pessoa_idx ON tarefa_partes (pessoa_id);
CREATE INDEX tarefa_responsaveis_pessoa_idx ON tarefa_responsaveis (pessoa_id);

-- Prazo fatal da tarefa é espelhado em prazos (regra de ouro: vale só depois de confirmado por advogado).
ALTER TABLE prazos ADD COLUMN tarefa_id uuid REFERENCES tarefas (id);
CREATE UNIQUE INDEX prazos_tarefa_uq ON prazos (tarefa_id) WHERE tarefa_id IS NOT NULL AND excluido_em IS NULL;

-- Documentos do caso podem estar ligados a um atendimento e à tarefa de onde foram enviados.
ALTER TABLE documentos ADD COLUMN atendimento_id uuid REFERENCES atendimentos (id);
ALTER TABLE documentos ADD COLUMN tarefa_id uuid REFERENCES tarefas (id);
ALTER TABLE documentos DROP CONSTRAINT documentos_check;
ALTER TABLE documentos ADD CONSTRAINT documentos_vinculo_ck
    CHECK (processo_id IS NOT NULL OR pessoa_id IS NOT NULL OR atendimento_id IS NOT NULL);
CREATE INDEX documentos_atendimento_idx ON documentos (atendimento_id) WHERE excluido_em IS NULL;
CREATE INDEX documentos_tarefa_idx ON documentos (tarefa_id) WHERE excluido_em IS NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM tarefas) OR EXISTS (SELECT 1 FROM atendimentos) THEN
        RAISE EXCEPTION 'migração 0009 não é reversível com dados: esvazie tarefas e atendimentos antes';
    END IF;
    IF EXISTS (SELECT 1 FROM documentos WHERE processo_id IS NULL AND pessoa_id IS NULL) THEN
        RAISE EXCEPTION 'migração 0009 não é reversível: há documentos ligados só a atendimento/tarefa';
    END IF;
END
$$;
-- +goose StatementEnd
DROP INDEX documentos_tarefa_idx;
DROP INDEX documentos_atendimento_idx;
ALTER TABLE documentos DROP CONSTRAINT documentos_vinculo_ck;
ALTER TABLE documentos DROP COLUMN tarefa_id, DROP COLUMN atendimento_id;
ALTER TABLE documentos ADD CONSTRAINT documentos_check CHECK (processo_id IS NOT NULL OR pessoa_id IS NOT NULL);
DROP INDEX prazos_tarefa_uq;
ALTER TABLE prazos DROP COLUMN tarefa_id;
DROP TABLE tarefa_responsaveis;
DROP TABLE tarefa_partes;
DROP TABLE tarefas;
CREATE TABLE tarefas (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo         text NOT NULL CHECK (length(titulo) BETWEEN 2 AND 300),
    processo_id    uuid REFERENCES processos (id),
    responsavel_id uuid REFERENCES users (id),
    vencimento     date,
    status         text NOT NULL CHECK (status IN ('aberta', 'em_andamento', 'em_revisao', 'concluida', 'cancelada')),
    origem         text NOT NULL DEFAULT 'manual' CHECK (origem IN ('manual', 'automacao_a_conferir')),
    excluido_em    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid NOT NULL REFERENCES users (id),
    version        integer NOT NULL DEFAULT 1,
    subtitulo      text NOT NULL DEFAULT '' CHECK (length(subtitulo) <= 80),
    prioridade     text NOT NULL DEFAULT 'normal' CHECK (prioridade IN ('baixa', 'normal', 'media', 'alta'))
);
CREATE INDEX tarefas_resp_venc_idx ON tarefas (responsavel_id, vencimento) WHERE excluido_em IS NULL;
CREATE INDEX tarefas_processo_idx ON tarefas (processo_id);
ALTER TABLE processos DROP CONSTRAINT processos_atendimento_origem_fk;
DROP TABLE atendimento_pessoas;
DROP TABLE atendimentos;
