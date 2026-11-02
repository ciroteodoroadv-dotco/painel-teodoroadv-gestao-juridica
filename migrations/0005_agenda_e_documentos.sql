-- +goose Up
-- Classificação LGPD: agenda_eventos.titulo/local podem conter dado pessoal (nome, endereço) = PESSOAL;
-- documentos.nome_original = PESSOAL; o ARQUIVO é cifrado em disco (AES-256-GCM) e nunca vai para o banco.

CREATE TABLE agenda_eventos (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo           text NOT NULL CHECK (tipo IN ('audiencia', 'reuniao', 'diligencia', 'outro')),
    titulo         text NOT NULL CHECK (length(titulo) BETWEEN 2 AND 300),
    inicio         timestamptz NOT NULL,
    fim            timestamptz,
    local          text NOT NULL DEFAULT '' CHECK (length(local) <= 500), -- endereço ou link
    processo_id    uuid REFERENCES processos (id),
    responsavel_id uuid REFERENCES users (id),
    excluido_em    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    created_by     uuid NOT NULL REFERENCES users (id),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    updated_by     uuid NOT NULL REFERENCES users (id),
    version        integer NOT NULL DEFAULT 1,
    CHECK (fim IS NULL OR fim >= inicio)
);
CREATE INDEX agenda_inicio_idx ON agenda_eventos (inicio) WHERE excluido_em IS NULL;
CREATE INDEX agenda_processo_idx ON agenda_eventos (processo_id);

CREATE TABLE agenda_participantes (
    evento_id uuid NOT NULL REFERENCES agenda_eventos (id) ON DELETE CASCADE,
    pessoa_id uuid NOT NULL REFERENCES pessoas (id),
    PRIMARY KEY (evento_id, pessoa_id)
);
CREATE INDEX agenda_participantes_pessoa_idx ON agenda_participantes (pessoa_id);

CREATE TABLE documentos (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nome_original text NOT NULL CHECK (length(nome_original) BETWEEN 1 AND 255),
    tipo          text NOT NULL CHECK (tipo IN ('pdf', 'png', 'jpeg', 'docx')), -- detectado pelo CONTEÚDO
    tamanho       bigint NOT NULL CHECK (tamanho > 0),
    sha256        bytea NOT NULL,
    arquivo       text NOT NULL UNIQUE, -- nome aleatório no volume privado
    descricao     text NOT NULL DEFAULT '' CHECK (length(descricao) <= 500),
    processo_id   uuid REFERENCES processos (id),
    pessoa_id     uuid REFERENCES pessoas (id),
    excluido_em   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    created_by    uuid NOT NULL REFERENCES users (id),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    updated_by    uuid NOT NULL REFERENCES users (id),
    version       integer NOT NULL DEFAULT 1,
    CHECK (processo_id IS NOT NULL OR pessoa_id IS NOT NULL)
);
CREATE INDEX documentos_processo_idx ON documentos (processo_id) WHERE excluido_em IS NULL;
CREATE INDEX documentos_pessoa_idx ON documentos (pessoa_id) WHERE excluido_em IS NULL;

-- +goose Down
DROP TABLE documentos;
DROP TABLE agenda_participantes;
DROP TABLE agenda_eventos;
