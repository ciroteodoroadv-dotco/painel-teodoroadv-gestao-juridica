-- +goose Up
-- Plano 2 da fusão: Pessoas e Processos com os campos do painel B.
-- Classificação LGPD por coluna (pessoas):
--   SENSÍVEL (cifrado na aplicação, AES-256-GCM, subchave por finalidade): documento_enc (CPF ou CNPJ),
--     rg_enc, representante_cpf_enc; documento_idx = índice cego (HMAC), não reversível.
--   PESSOAL: nome_completo, razao_social, nome_fantasia, representante_nome, data_nascimento, endereço
--     (logradouro, numero, complemento, bairro, cep, cidade, uf), email, telefone, estado_civil, profissao.
--   COMUM: codigo, tipo_pessoa, nacionalidade, representante_cargo, observacoes (não deve conter dado sensível).
-- O papel da pessoa (cliente, parte contrária…) NÃO é coluna: deriva dos processos (decisão do B de 13/09/2026).
-- As tabelas pessoas/processos precisam estar vazias: o v2 começa limpo, e converter dado real em silêncio seria perigoso.

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pessoas) OR EXISTS (SELECT 1 FROM processos) THEN
        RAISE EXCEPTION 'migração 0008: pessoas/processos já têm linhas; ela só aplica sobre tabelas vazias (converta os dados antes)';
    END IF;
END
$$;
-- +goose StatementEnd

CREATE TABLE codigos_contador (
    prefixo text PRIMARY KEY CHECK (prefixo ~ '^[A-Z]{3}$'),
    ultimo  bigint NOT NULL CHECK (ultimo >= 0)
);

-- ---------------------------------------------------------------- pessoas
DROP INDEX pessoas_nome_idx;
ALTER TABLE pessoas
    DROP COLUMN tipo,
    DROP COLUMN nome,
    DROP COLUMN papel,
    DROP COLUMN telefone,
    DROP COLUMN email;
ALTER TABLE pessoas
    ADD COLUMN codigo               text NOT NULL,
    ADD COLUMN tipo_pessoa          text NOT NULL CHECK (tipo_pessoa IN ('Física', 'Jurídica')),
    ADD COLUMN nome_completo        text NOT NULL DEFAULT '',
    ADD COLUMN rg_enc               bytea,
    ADD COLUMN estado_civil         text NOT NULL DEFAULT '',
    ADD COLUMN profissao            text NOT NULL DEFAULT '',
    ADD COLUMN nacionalidade        text NOT NULL DEFAULT 'Brasileiro(a)',
    ADD COLUMN data_nascimento      date,
    ADD COLUMN razao_social         text NOT NULL DEFAULT '',
    ADD COLUMN nome_fantasia        text NOT NULL DEFAULT '',
    ADD COLUMN representante_nome   text NOT NULL DEFAULT '',
    ADD COLUMN representante_cpf_enc bytea,
    ADD COLUMN representante_cargo  text NOT NULL DEFAULT '',
    ADD COLUMN logradouro           text NOT NULL DEFAULT '',
    ADD COLUMN numero               text NOT NULL DEFAULT '',
    ADD COLUMN complemento          text NOT NULL DEFAULT '',
    ADD COLUMN bairro               text NOT NULL DEFAULT '',
    ADD COLUMN cep                  text NOT NULL DEFAULT '',
    ADD COLUMN cidade               text NOT NULL DEFAULT '',
    ADD COLUMN uf                   char(2),
    ADD COLUMN email                text NOT NULL DEFAULT '',
    ADD COLUMN telefone             text NOT NULL DEFAULT '',
    ADD COLUMN observacoes          text NOT NULL DEFAULT '',
    ADD COLUMN nome_exibicao        text GENERATED ALWAYS AS (
        CASE WHEN tipo_pessoa = 'Jurídica' THEN razao_social ELSE nome_completo END) STORED,
    ADD CONSTRAINT pessoas_nome_do_tipo_ck CHECK (
        (tipo_pessoa = 'Física' AND length(btrim(nome_completo)) > 0) OR
        (tipo_pessoa = 'Jurídica' AND length(btrim(razao_social)) > 0));
CREATE UNIQUE INDEX pessoas_codigo_uq ON pessoas (codigo);
CREATE INDEX pessoas_nome_idx ON pessoas (lower(nome_exibicao));

-- --------------------------------------------------------------- processos
DROP INDEX processos_numero_uq;
ALTER TABLE processos DROP CONSTRAINT processos_situacao_check;
ALTER TABLE processos
    DROP COLUMN orgao,
    DROP COLUMN responsavel_id;
ALTER TABLE processos
    ADD COLUMN codigo               text NOT NULL,
    ADD COLUMN titulo               text NOT NULL DEFAULT '',
    ADD COLUMN polo                 text NOT NULL DEFAULT '',
    ADD COLUMN sistema              text NOT NULL DEFAULT '',
    ADD COLUMN instancia            text NOT NULL DEFAULT '',
    ADD COLUMN classe_processual    text NOT NULL DEFAULT '',
    ADD COLUMN materia_direito      text NOT NULL DEFAULT '',
    ADD COLUMN comarca              text NOT NULL DEFAULT '',
    ADD COLUMN juizo                text NOT NULL DEFAULT '',
    ADD COLUMN tipo_contingencia    text NOT NULL DEFAULT '' CHECK (tipo_contingencia IN ('Ativa', 'Passiva', '')),
    ADD COLUMN valor_causa          numeric(16,2),
    ADD COLUMN distribuicao         date,
    ADD COLUMN citacao              date,
    ADD COLUMN audiencia            date,
    ADD COLUMN sentenca             date,
    ADD COLUMN recursos             date,
    ADD COLUMN fase_execucao        date,
    ADD COLUMN atendimento_origem_id uuid, -- a chave estrangeira nasce no Plano 3 (Atendimentos)
    ADD COLUMN projeto_vinculado    text NOT NULL DEFAULT '',
    ADD COLUMN observacoes          text NOT NULL DEFAULT '',
    ADD COLUMN prox_o_que           text NOT NULL DEFAULT '',
    ADD COLUMN prox_ate_quando      date,
    ADD COLUMN prox_quem            text NOT NULL DEFAULT '';
ALTER TABLE processos ADD CONSTRAINT processos_situacao_check
    CHECK (situacao IN ('ativo', 'sobrestado', 'extinto', 'arquivado'));
CREATE UNIQUE INDEX processos_codigo_uq ON processos (codigo);
-- Número vazio é permitido ("Em branco se ainda não distribuído"); só os preenchidos são únicos.
CREATE UNIQUE INDEX processos_numero_uq ON processos (numero_cnj) WHERE numero_cnj <> '' AND excluido_em IS NULL;

-- ---------------------------------------------------------- partes e afins
ALTER TABLE processo_partes DROP CONSTRAINT processo_partes_pkey;
ALTER TABLE processo_partes
    DROP COLUMN qualidade,
    ADD COLUMN papel   text    NOT NULL CHECK (length(btrim(papel)) > 0), -- livre: Autor, Réu… ou o texto de "Outro"
    ADD COLUMN cliente boolean NOT NULL DEFAULT false,
    ADD COLUMN ordem   integer NOT NULL DEFAULT 0;
ALTER TABLE processo_partes ADD PRIMARY KEY (processo_id, pessoa_id, papel);

CREATE TABLE processo_responsaveis (
    processo_id uuid NOT NULL REFERENCES processos (id) ON DELETE CASCADE,
    pessoa_id   uuid NOT NULL REFERENCES pessoas (id),
    ordem       integer NOT NULL,
    PRIMARY KEY (processo_id, ordem)
);
CREATE INDEX processo_responsaveis_pessoa_idx ON processo_responsaveis (pessoa_id);

CREATE TABLE processo_andamentos (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    processo_id uuid NOT NULL REFERENCES processos (id) ON DELETE CASCADE,
    data        date NOT NULL,
    o_que       text NOT NULL CHECK (length(btrim(o_que)) > 0),
    documento   text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  uuid NOT NULL REFERENCES users (id)
);
CREATE INDEX processo_andamentos_proc_idx ON processo_andamentos (processo_id, data DESC);

CREATE TABLE processo_financeiro (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    processo_id uuid NOT NULL REFERENCES processos (id) ON DELETE CASCADE,
    item        text NOT NULL CHECK (length(btrim(item)) > 0),
    valor       text NOT NULL DEFAULT '',
    situacao    text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  uuid NOT NULL REFERENCES users (id)
);
CREATE UNIQUE INDEX processo_financeiro_item_uq ON processo_financeiro (processo_id, lower(item));

-- O app (papel painel_app) já recebe SELECT/INSERT/UPDATE/DELETE nas tabelas novas pelos privilégios
-- padrão da migração 0003; o REVOKE da auditoria continua valendo. Reforço explícito:
GRANT SELECT, INSERT, UPDATE, DELETE ON codigos_contador, processo_responsaveis, processo_andamentos, processo_financeiro TO painel_app;

-- +goose Down
-- Não é reversível sem perda: recusa se houver dado de negócio (nada se perde em silêncio).
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pessoas) OR EXISTS (SELECT 1 FROM processos) OR EXISTS (SELECT 1 FROM processo_andamentos)
       OR EXISTS (SELECT 1 FROM processo_financeiro) OR EXISTS (SELECT 1 FROM processo_responsaveis) THEN
        RAISE EXCEPTION 'migração 0008 não é reversível com dados: esvazie pessoas, processos e tabelas filhas antes';
    END IF;
END
$$;
-- +goose StatementEnd

DROP TABLE processo_financeiro;
DROP TABLE processo_andamentos;
DROP TABLE processo_responsaveis;

ALTER TABLE processo_partes DROP CONSTRAINT processo_partes_pkey;
ALTER TABLE processo_partes
    DROP COLUMN papel, DROP COLUMN cliente, DROP COLUMN ordem,
    ADD COLUMN qualidade text NOT NULL CHECK (qualidade IN ('autor', 'reu', 'terceiro', 'cliente'));
ALTER TABLE processo_partes ADD PRIMARY KEY (processo_id, pessoa_id, qualidade);

DROP INDEX processos_numero_uq;
DROP INDEX processos_codigo_uq;
ALTER TABLE processos DROP CONSTRAINT processos_situacao_check;
ALTER TABLE processos
    DROP COLUMN codigo, DROP COLUMN titulo, DROP COLUMN polo, DROP COLUMN sistema, DROP COLUMN instancia,
    DROP COLUMN classe_processual, DROP COLUMN materia_direito, DROP COLUMN comarca, DROP COLUMN juizo,
    DROP COLUMN tipo_contingencia, DROP COLUMN valor_causa, DROP COLUMN distribuicao, DROP COLUMN citacao,
    DROP COLUMN audiencia, DROP COLUMN sentenca, DROP COLUMN recursos, DROP COLUMN fase_execucao,
    DROP COLUMN atendimento_origem_id, DROP COLUMN projeto_vinculado, DROP COLUMN observacoes,
    DROP COLUMN prox_o_que, DROP COLUMN prox_ate_quando, DROP COLUMN prox_quem;
ALTER TABLE processos
    ADD COLUMN orgao text NOT NULL DEFAULT '',
    ADD COLUMN responsavel_id uuid REFERENCES users (id);
ALTER TABLE processos ADD CONSTRAINT processos_situacao_check
    CHECK (situacao IN ('ativo', 'suspenso', 'arquivado', 'encerrado'));
CREATE UNIQUE INDEX processos_numero_uq ON processos (numero_cnj) WHERE excluido_em IS NULL;

DROP INDEX pessoas_nome_idx;
DROP INDEX pessoas_codigo_uq;
ALTER TABLE pessoas DROP CONSTRAINT pessoas_nome_do_tipo_ck;
ALTER TABLE pessoas
    DROP COLUMN nome_exibicao, DROP COLUMN codigo, DROP COLUMN tipo_pessoa, DROP COLUMN nome_completo,
    DROP COLUMN rg_enc, DROP COLUMN estado_civil, DROP COLUMN profissao, DROP COLUMN nacionalidade,
    DROP COLUMN data_nascimento, DROP COLUMN razao_social, DROP COLUMN nome_fantasia,
    DROP COLUMN representante_nome, DROP COLUMN representante_cpf_enc, DROP COLUMN representante_cargo,
    DROP COLUMN logradouro, DROP COLUMN numero, DROP COLUMN complemento, DROP COLUMN bairro, DROP COLUMN cep,
    DROP COLUMN cidade, DROP COLUMN uf, DROP COLUMN email, DROP COLUMN telefone, DROP COLUMN observacoes;
ALTER TABLE pessoas
    ADD COLUMN tipo text NOT NULL CHECK (tipo IN ('PF', 'PJ')),
    ADD COLUMN nome text NOT NULL CHECK (length(nome) BETWEEN 2 AND 200),
    ADD COLUMN email text,
    ADD COLUMN telefone text,
    ADD COLUMN papel text NOT NULL CHECK (papel IN ('cliente', 'parte_contraria', 'testemunha', 'outro'));
CREATE INDEX pessoas_nome_idx ON pessoas (lower(nome));

DROP TABLE codigos_contador;
