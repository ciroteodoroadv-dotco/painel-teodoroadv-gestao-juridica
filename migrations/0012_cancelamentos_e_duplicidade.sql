-- +goose Up
-- Plano 4C da fusão (determinação do advogado, 04/11/2026): nada se apaga; processo duplicado com justificativa.
-- Classificação LGPD por coluna:
--   *.cancelamento_motivo = PESSOAL (texto livre do advogado, pode citar partes); cancelado, cancelado_em, cancelado_por = COMUM.
--   processos.duplicado_justificativa = PESSOAL; duplicado, duplicado_de = COMUM.
-- Nada é apagado: as intimações 'descartada' viram 'cancelada' (com motivo explicando a origem) e os processos de mesmo
-- número continuam existindo, agora com o marcador de duplicidade.

-- Compromissos (Reunião, Ligação, Aula) -----------------------------------------------------------------------------
ALTER TABLE agenda_compromissos
    ADD COLUMN cancelado           boolean NOT NULL DEFAULT false,
    ADD COLUMN cancelamento_motivo text NOT NULL DEFAULT '',
    ADD COLUMN cancelado_em        timestamptz,
    ADD COLUMN cancelado_por       uuid REFERENCES users (id),
    ADD CONSTRAINT agenda_cancelado_ck CHECK (NOT cancelado OR length(btrim(cancelamento_motivo)) > 0);

-- Diligências ---------------------------------------------------------------------------------------------------------
ALTER TABLE diligencias
    ADD COLUMN cancelado           boolean NOT NULL DEFAULT false,
    ADD COLUMN cancelamento_motivo text NOT NULL DEFAULT '',
    ADD COLUMN cancelado_em        timestamptz,
    ADD COLUMN cancelado_por       uuid REFERENCES users (id),
    ADD CONSTRAINT diligencias_cancelado_ck CHECK (NOT cancelado OR length(btrim(cancelamento_motivo)) > 0);
UPDATE diligencias SET cancelado = true, cancelamento_motivo = 'Cancelada antes da migração 0012 (sem motivo registrado).'
    WHERE status = 'Cancelada';

-- Intimações: 'descartada' -> 'cancelada' ----------------------------------------------------------------------------
ALTER TABLE intimacoes
    ADD COLUMN cancelamento_motivo text NOT NULL DEFAULT '',
    ADD COLUMN cancelado_em        timestamptz,
    ADD COLUMN cancelado_por       uuid REFERENCES users (id);
ALTER TABLE intimacoes DROP CONSTRAINT intimacoes_status_check;
UPDATE intimacoes SET status = 'cancelada', cancelado_em = decidida_em, cancelado_por = decidida_por,
    cancelamento_motivo = 'Descartada antes da migração 0012 (sem motivo registrado).' WHERE status = 'descartada';
ALTER TABLE intimacoes ADD CONSTRAINT intimacoes_status_check CHECK (status IN ('triagem', 'confirmada', 'cancelada'));
ALTER TABLE intimacoes ADD CONSTRAINT intimacoes_cancelada_ck CHECK (status <> 'cancelada' OR length(btrim(cancelamento_motivo)) > 0);

-- Processos: número deixa de ser único; a duplicata exige marcador e justificativa --------------------------------------
DROP INDEX processos_numero_uq;
CREATE INDEX processos_numero_idx ON processos (numero_cnj) WHERE numero_cnj <> '' AND excluido_em IS NULL;
ALTER TABLE processos
    ADD COLUMN duplicado               boolean NOT NULL DEFAULT false,
    ADD COLUMN duplicado_justificativa text NOT NULL DEFAULT '',
    ADD COLUMN duplicado_de            uuid REFERENCES processos (id),
    ADD CONSTRAINT processos_duplicado_ck CHECK (NOT duplicado OR length(btrim(duplicado_justificativa)) >= 10);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM agenda_compromissos WHERE cancelado) OR EXISTS (SELECT 1 FROM diligencias WHERE cancelado)
       OR EXISTS (SELECT 1 FROM intimacoes WHERE status = 'cancelada') OR EXISTS (SELECT 1 FROM processos WHERE duplicado) THEN
        RAISE EXCEPTION 'migração 0012 não é reversível com cancelamentos ou processos duplicados registrados: nada se apaga';
    END IF;
END
$$;
-- +goose StatementEnd
DROP INDEX processos_numero_idx;
CREATE UNIQUE INDEX processos_numero_uq ON processos (numero_cnj) WHERE numero_cnj <> '' AND excluido_em IS NULL;
ALTER TABLE processos DROP CONSTRAINT processos_duplicado_ck;
ALTER TABLE processos DROP COLUMN duplicado_de, DROP COLUMN duplicado_justificativa, DROP COLUMN duplicado;
ALTER TABLE intimacoes DROP CONSTRAINT intimacoes_cancelada_ck;
ALTER TABLE intimacoes DROP CONSTRAINT intimacoes_status_check;
ALTER TABLE intimacoes ADD CONSTRAINT intimacoes_status_check CHECK (status IN ('triagem', 'confirmada', 'descartada'));
ALTER TABLE intimacoes DROP COLUMN cancelado_por, DROP COLUMN cancelado_em, DROP COLUMN cancelamento_motivo;
ALTER TABLE diligencias DROP CONSTRAINT diligencias_cancelado_ck;
ALTER TABLE diligencias DROP COLUMN cancelado_por, DROP COLUMN cancelado_em, DROP COLUMN cancelamento_motivo, DROP COLUMN cancelado;
ALTER TABLE agenda_compromissos DROP CONSTRAINT agenda_cancelado_ck;
ALTER TABLE agenda_compromissos DROP COLUMN cancelado_por, DROP COLUMN cancelado_em, DROP COLUMN cancelamento_motivo, DROP COLUMN cancelado;
