-- +goose Up
-- Classificação LGPD por coluna:
--   pessoas.nome, email, telefone = PESSOAL; documento_enc = SENSÍVEL (cifrado, AES-256-GCM na aplicação);
--   documento_idx = índice cego (HMAC), não reversível; demais colunas = comuns.
-- Toda tabela: created_at/by, updated_at/by e version (concorrência otimista). Exclusão é marcação
-- (excluido_em) — o expurgo definitivo é decisão futura, com confirmação, auditoria e backup.

CREATE TABLE pessoas (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo          text NOT NULL CHECK (tipo IN ('PF', 'PJ')),
    nome          text NOT NULL CHECK (length(nome) BETWEEN 2 AND 200),
    documento_enc bytea,
    documento_idx bytea,
    email         text,
    telefone      text,
    papel         text NOT NULL CHECK (papel IN ('cliente', 'parte_contraria', 'testemunha', 'outro')),
    retencao_ate  date,
    excluido_em   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    uuid NOT NULL REFERENCES users (id),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    uuid NOT NULL REFERENCES users (id),
    version       integer NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX pessoas_documento_uq ON pessoas (documento_idx)
    WHERE documento_idx IS NOT NULL AND excluido_em IS NULL;
CREATE INDEX pessoas_nome_idx ON pessoas (lower(nome));

CREATE TABLE processos (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    numero_cnj     text NOT NULL,
    orgao          text NOT NULL DEFAULT '',
    situacao       text NOT NULL CHECK (situacao IN ('ativo', 'suspenso', 'arquivado', 'encerrado')),
    responsavel_id uuid REFERENCES users (id),
    excluido_em    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid NOT NULL REFERENCES users (id),
    version        integer NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX processos_numero_uq ON processos (numero_cnj) WHERE excluido_em IS NULL;

CREATE TABLE processo_partes (
    processo_id uuid NOT NULL REFERENCES processos (id) ON DELETE CASCADE,
    pessoa_id   uuid NOT NULL REFERENCES pessoas (id),
    qualidade   text NOT NULL CHECK (qualidade IN ('autor', 'reu', 'terceiro', 'cliente')),
    PRIMARY KEY (processo_id, pessoa_id, qualidade)
);
CREATE INDEX processo_partes_pessoa_idx ON processo_partes (pessoa_id);

CREATE TABLE tarefas (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo         text NOT NULL CHECK (length(titulo) BETWEEN 2 AND 300),
    processo_id    uuid REFERENCES processos (id),
    responsavel_id uuid REFERENCES users (id),
    vencimento     date, -- informado pelo advogado: o sistema NUNCA calcula prazo
    status         text NOT NULL CHECK (status IN ('aberta', 'em_andamento', 'concluida', 'cancelada')),
    origem         text NOT NULL DEFAULT 'manual' CHECK (origem IN ('manual', 'automacao_a_conferir')),
    excluido_em    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid NOT NULL REFERENCES users (id),
    version        integer NOT NULL DEFAULT 1
);
CREATE INDEX tarefas_resp_venc_idx ON tarefas (responsavel_id, vencimento) WHERE excluido_em IS NULL;
CREATE INDEX tarefas_processo_idx ON tarefas (processo_id);

-- +goose Down
DROP TABLE tarefas;
DROP TABLE processo_partes;
DROP TABLE processos;
DROP TABLE pessoas;
