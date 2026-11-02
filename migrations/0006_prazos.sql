-- +goose Up
-- PRAZO é entidade própria (não é "tarefa com data"): tem fundamento, fonte, situação e confirmação por advogado.
-- Regra de ouro: um prazo só VALE depois de confirmado por advogado (quem e quando ficam registrados). Sugestões
-- automáticas (futuro: DJEN/DataJud) entram como 'a_conferir' com origem='sugestao'; nada é confirmado sozinho.
CREATE TABLE prazos (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    descricao      text NOT NULL CHECK (length(descricao) BETWEEN 2 AND 500),
    processo_id    uuid REFERENCES processos (id),
    data_fatal     date NOT NULL,
    fundamento     text NOT NULL DEFAULT '' CHECK (length(fundamento) <= 300),
    fonte          text NOT NULL DEFAULT 'manual' CHECK (fonte IN ('manual', 'djen', 'datajud', 'outra')),
    origem         text NOT NULL DEFAULT 'manual' CHECK (origem IN ('manual', 'sugestao')),
    situacao       text NOT NULL CHECK (situacao IN ('a_conferir', 'confirmado', 'cumprido', 'cancelado')),
    responsavel_id uuid REFERENCES users (id),
    confirmado_por uuid REFERENCES users (id),
    confirmado_em  timestamptz,
    cumprido_em    timestamptz,
    excluido_em    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid NOT NULL REFERENCES users (id),
    version        integer NOT NULL DEFAULT 1,
    CHECK (situacao NOT IN ('confirmado', 'cumprido') OR (confirmado_por IS NOT NULL AND confirmado_em IS NOT NULL)),
    CHECK (situacao <> 'cumprido' OR cumprido_em IS NOT NULL)
);
CREATE INDEX prazos_abertos_idx ON prazos (data_fatal) WHERE excluido_em IS NULL AND situacao IN ('a_conferir', 'confirmado');
CREATE INDEX prazos_processo_idx ON prazos (processo_id);

-- +goose Down
DROP TABLE prazos;
