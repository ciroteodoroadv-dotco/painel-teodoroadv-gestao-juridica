-- +goose Up
-- Plano 4 da fusão: Agenda, Audiências e Diligências com os campos do painel B.
-- Classificação LGPD por coluna:
--   agenda_compromissos.descricao, local = PESSOAL (podem citar nomes e endereços); tipo, data, hora = COMUM.
--   audiencias.titulo, processo_numero, juizo, local, descricao, cancelamento_motivo = PESSOAL; tipo_ato, data, hora = COMUM.
--   diligencias.titulo, processo_numero, local, responsavel = PESSOAL; status, datas = COMUM.
-- O B não tem participantes nem página de detalhe de evento: agenda_eventos e agenda_participantes saem,
-- mas só se estiverem vazias (nada de dado é perdido em silêncio).
-- Exclusão é marcação (excluido_em). Cancelamento de audiência preserva a ficha (motivo, quem e quando).

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM agenda_eventos) OR EXISTS (SELECT 1 FROM agenda_participantes) THEN
        RAISE EXCEPTION 'migração 0010: agenda_eventos/agenda_participantes têm linhas; ela só aplica sobre tabelas vazias (converta os dados antes)';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE agenda_participantes;
DROP TABLE agenda_eventos;

CREATE TABLE agenda_compromissos (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo                  text NOT NULL CHECK (tipo IN ('Reunião', 'Ligação', 'Aula')),
    descricao             text NOT NULL CHECK (length(btrim(descricao)) > 0),
    data                  date NOT NULL,
    hora                  time NOT NULL,
    local                 text NOT NULL DEFAULT '',
    cliente_pessoa_id     uuid REFERENCES pessoas (id),
    atendimento_id        uuid REFERENCES atendimentos (id),
    processo_id           uuid REFERENCES processos (id),
    criado_por_pessoa_id  uuid REFERENCES pessoas (id),
    excluido_em           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    created_by            uuid NOT NULL REFERENCES users (id),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    updated_by            uuid NOT NULL REFERENCES users (id),
    version               integer NOT NULL DEFAULT 1
);
CREATE INDEX agenda_compromissos_data_idx ON agenda_compromissos (data, hora) WHERE excluido_em IS NULL;

CREATE TABLE audiencias (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo               text NOT NULL,
    tipo_ato             text NOT NULL CHECK (tipo_ato IN ('audiencia-ad-monitoria', 'conciliacao-e-mediacao',
                         'inicial-justica-do-trabalho', 'instrucao-e-julgamento', 'sustentacao-oral', 'una-justica-do-trabalho')),
    titulo               text NOT NULL CHECK (length(btrim(titulo)) > 0),
    processo_numero      text NOT NULL DEFAULT '',
    processo_id          uuid REFERENCES processos (id),
    cliente_pessoa_id    uuid REFERENCES pessoas (id),
    data                 date NOT NULL,
    hora                 time NOT NULL,
    juizo                text NOT NULL DEFAULT '',
    local                text NOT NULL DEFAULT '',
    descricao            text NOT NULL DEFAULT '',
    cancelada            boolean NOT NULL DEFAULT false,
    cancelamento_motivo  text NOT NULL DEFAULT '',
    cancelada_em         timestamptz,
    cancelada_por        uuid REFERENCES users (id),
    excluido_em          timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now(),
    created_by           uuid NOT NULL REFERENCES users (id),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid NOT NULL REFERENCES users (id),
    version              integer NOT NULL DEFAULT 1,
    CHECK (NOT cancelada OR length(btrim(cancelamento_motivo)) > 0)
);
CREATE UNIQUE INDEX audiencias_codigo_uq ON audiencias (codigo);
CREATE INDEX audiencias_data_idx ON audiencias (data, hora) WHERE excluido_em IS NULL;

CREATE TABLE diligencias (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo              text NOT NULL,
    titulo              text NOT NULL CHECK (length(btrim(titulo)) > 0),
    processo_numero     text NOT NULL DEFAULT '',
    processo_id         uuid REFERENCES processos (id),
    cliente_pessoa_id   uuid REFERENCES pessoas (id),
    local               text NOT NULL DEFAULT '',
    data_limite         date NOT NULL,
    hora                time,
    responsavel         text NOT NULL DEFAULT '',
    status              text NOT NULL DEFAULT 'Pendente' CHECK (status IN ('Pendente', 'Concluída', 'Cancelada')),
    excluido_em         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    created_by          uuid NOT NULL REFERENCES users (id),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    updated_by          uuid NOT NULL REFERENCES users (id),
    version             integer NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX diligencias_codigo_uq ON diligencias (codigo);
CREATE INDEX diligencias_data_idx ON diligencias (data_limite) WHERE excluido_em IS NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM agenda_compromissos) OR EXISTS (SELECT 1 FROM audiencias) OR EXISTS (SELECT 1 FROM diligencias) THEN
        RAISE EXCEPTION 'migração 0010 não é reversível com dados: esvazie agenda_compromissos, audiencias e diligencias antes';
    END IF;
END
$$;
-- +goose StatementEnd
DROP TABLE diligencias;
DROP TABLE audiencias;
DROP TABLE agenda_compromissos;
CREATE TABLE agenda_eventos (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo           text NOT NULL CHECK (tipo IN ('audiencia', 'reuniao', 'ligacao', 'diligencia', 'aula', 'outro')),
    titulo         text NOT NULL CHECK (length(titulo) BETWEEN 2 AND 300),
    inicio         timestamptz NOT NULL,
    fim            timestamptz,
    local          text NOT NULL DEFAULT '' CHECK (length(local) <= 500),
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
