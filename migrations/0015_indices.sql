-- +goose Up
-- Plano 8 da fusão: índices econômicos do Banco Central (SGS) para o rodapé.
-- Classificação LGPD: indices_valores = COMUM (dado público do BCB; nenhum dado pessoal).
-- Guarda só o ÚLTIMO valor de cada série (o rodapé é informativo; não serve de base para cálculo judicial).
-- "falhou" = a última tentativa de coleta desta série falhou: o valor antigo fica com a data antiga e a marca "desatualizado".
-- A integração vem DESLIGADA por padrão (PAINEL_INDICES_ATIVO=false).

CREATE TABLE indices_valores (
    serie_codigo integer PRIMARY KEY,
    rotulo       text NOT NULL CHECK (char_length(rotulo) BETWEEN 1 AND 80),
    valor        numeric(20, 6) NOT NULL,
    unidade      text NOT NULL CHECK (unidade IN ('% a.a.', '% mensal', 'R$')),
    referencia   date NOT NULL,
    obtido_em    timestamptz NOT NULL,
    fonte        text NOT NULL DEFAULT 'BCB SGS',
    falhou       boolean NOT NULL DEFAULT false
);
REVOKE DELETE ON indices_valores FROM painel_app;

-- +goose Down
DROP TABLE indices_valores;
