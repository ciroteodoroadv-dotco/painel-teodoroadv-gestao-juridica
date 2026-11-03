-- +goose Up
ALTER TABLE agenda_eventos DROP CONSTRAINT agenda_eventos_tipo_check;
ALTER TABLE agenda_eventos ADD CONSTRAINT agenda_eventos_tipo_check
  CHECK (tipo IN ('audiencia','reuniao','ligacao','diligencia','aula','outro'));
ALTER TABLE tarefas DROP CONSTRAINT tarefas_status_check;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_status_check
  CHECK (status IN ('aberta','em_andamento','em_revisao','concluida','cancelada'));
ALTER TABLE tarefas ADD COLUMN subtitulo text NOT NULL DEFAULT '' CHECK (length(subtitulo) <= 80);
ALTER TABLE tarefas ADD COLUMN prioridade text NOT NULL DEFAULT 'normal'
  CHECK (prioridade IN ('baixa','normal','media','alta'));

-- +goose Down
-- Falha de propósito se houver linhas com os valores novos: não se perde dado em silêncio.
ALTER TABLE tarefas DROP COLUMN prioridade;
ALTER TABLE tarefas DROP COLUMN subtitulo;
ALTER TABLE tarefas DROP CONSTRAINT tarefas_status_check;
ALTER TABLE tarefas ADD CONSTRAINT tarefas_status_check CHECK (status IN ('aberta','em_andamento','concluida','cancelada'));
ALTER TABLE agenda_eventos DROP CONSTRAINT agenda_eventos_tipo_check;
ALTER TABLE agenda_eventos ADD CONSTRAINT agenda_eventos_tipo_check CHECK (tipo IN ('audiencia','reuniao','diligencia','outro'));
