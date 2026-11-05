# Fusão — Plano 2: Pessoas e Processos idênticos ao painel B

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever qualquer template, leia por inteiro, linha por linha, no B (somente leitura):** `referencia/painel-b/js/14-pessoas.js`, `referencia/painel-b/js/09-processos.js`, `referencia/painel-b/js/06-helpers-vinculos.js`, `referencia/painel-b/js/03-api-mapa.js`, `referencia/painel-b/js/05-dominio.js`, `referencia/painel-b/js/07-app-core.js`, `referencia/painel-b/js/15-drawers.js` e `referencia/painel-b/css/painel.css`. **O B vale na forma; o v2 vale na regra.** Rótulos, ordem dos campos, dicas (`hint`), `placeholder`, textos de subtítulo/toast, opções de select, valores padrão e ordem das colunas são **copiados literalmente**.

**Goal:** As telas **Pessoas** e **Processos** do v2 passam a ter os mesmos campos, ordem, rótulos, drawers, acordeões e ações do B, sobre PostgreSQL, com a segurança do v2 (RBAC, auditoria, CSRF, cifra de documentos, exclusão lógica).

**Architecture:** Migração `0008` remodela `pessoas`/`processos` (as tabelas estão vazias — o v2 começa limpo) e cria `processo_andamentos` e `processo_financeiro`. Os domínios `pessoas` e `processos` ganham os campos do B; **o papel da pessoa deixa de ser coluna** e passa a ser **derivado dos processos** (decisão do B de 13/09/2026). Formulários são fragmentos de drawer servidos pelo servidor e enviados com `data-drawer-form` (contrato do Plano 1B). Domínios continuam se falando só por interfaces.

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, JS do Plano 1B (`drawer.js`, `forms.js`, `mascaras.js`).

**Spec:** `docs/plano-fusao/ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `01b-casca-identica-ao-b.md`.

## Global Constraints

- Tudo do Plano 1B (CSP estrita sem `on*=`/`style=`/`innerHTML`; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...`).
- **Sem estado novo de UI além do B**: acordeão de Pessoas e de Processos abre **vários painéis**; busca filtra ao digitar (`data-filtro`).
- Código de ficha `PES. 0000003` / `PRO. 0000012`: prefixo + ponto + espaço + 7 dígitos, atribuído **uma vez** na criação, nunca muda, contador por prefixo (tabela `codigos_contador(prefixo text primary key, ultimo bigint)` com `UPDATE … RETURNING` na mesma transação; reaproveitada nos Planos 3 e 4 com `ATE`, `TAR`, `AUD`, `DIL`).
- LGPD: CPF, CNPJ, RG e CPF do representante são **cifrados** (AES-256-GCM, subchave por finalidade) e têm índice cego quando houver unicidade; data de nascimento, endereço, e-mail, telefone são PESSOAIS (classificação documentada na migração). A listagem mostra CPF/CNPJ completos **só a quem tem a permissão de ver documento** (a mesma que o v2 já usa em `pessoas.Reveal`); sem ela, mascarado.
- Exclusão: **lógica** (`excluido_em`), auditada. Texto do B "Excluir definitivamente… não pode ser desfeita" **não é verdadeiro no v2**; o texto do drawer vira: "Excluir esta ficha? Ela deixa de aparecer nas listas e a exclusão fica registrada na auditoria." (única divergência de texto, por regra de segurança).
- Pasta do B somente leitura; sem credenciais reais; sem dado real.
- Commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; sem push.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Campo "Nome do arquivo da ficha" (existe no B só porque a ficha é um `.md`) | **Omitido**: não tem função no v2. Anotado em `docs/arquivo/plano-fusao/NOTAS-PLANO-2.md` para o advogado aprovar a única omissão de campo. |
| Responsáveis (do processo e, depois, da tarefa) | **Pessoas** (selects de pessoa), como no B; não usuários do sistema. |
| Quem pode salvar | `PermPessoaWrite`/`PermProcessoWrite` existentes; leitura `…Read`; botões de escrita só aparecem a quem pode. |
| "Pendências (N)" do processo | O B deriva de marcadores `[VERIFICAR]/[FALTA]` na ficha. No v2 é **calculado**: lista fixa de campos que ainda não foram preenchidos — Número do processo (CNJ), Sistema, Instância, Classe processual, Matéria / ramo do direito, Comarca, Juízo, Distribuição, Valor da causa, "Pelo menos uma parte marcada como cliente". Badge `badge-warn` com `title` listando os faltantes (igual ao B). |
| Estados do processo | `ativo`, `sobrestado`, `extinto`, `arquivado` (rótulos de grupo: Ativos, Sobrestados, Extintos, Arquivados definitivamente). Migração de `situacao` antiga: `suspenso→sobrestado`, `encerrado→extinto`. |
| Botão "Excluir" da Pessoa | Como no B, **pessoa referenciada em processo** abre o(s) processo(s) de origem (drawer "Escolha o processo" se houver mais de um); **pessoa sem processo** abre a confirmação de exclusão lógica. |
| Mover estado → tarefa de monitoramento | O drawer mantém o texto do B ("…a tarefa de monitoramento do processo é baixada em cadeia…"), mas a regra **só passa a valer no Plano 6**; até lá o servidor apenas registra a mudança (o texto fica, o Plano 6 liga o efeito). |
| CEP | Botão "Buscar CEP" e hint "Fonte consultada: …" **idênticos**; a consulta ao ViaCEP só é ligada no Plano 8 (porta da integração desligada por padrão). Até lá o botão responde com toast de erro "Consulta de CEP indisponível neste painel." (nunca dado inventado). |

---

## Task 1: Migração 0008 e contador de códigos

**Files:** Create `migrations/0008_pessoas_processos_do_b.sql`, `internal/platform/codigos/codigos.go` (+ `_test.go`); Test `internal/app/migracoes_0008_test.go`.

**Interfaces — Produces:**
```go
package codigos
// Proximo devolve "PES. 0000001"… dentro da transação tx; o contador é por prefixo.
func Proximo(ctx context.Context, tx pgx.Tx, prefixo string) (string, error)
```
**Esquema (as colunas abaixo, nomes literais):**
- `pessoas`: **remover** `tipo`, `nome`, `papel`, `telefone`, `email` antigos e recriar como: `codigo text NOT NULL UNIQUE`, `tipo_pessoa text CHECK IN ('Física','Jurídica')`, `nome_completo`, `rg_enc bytea`, `estado_civil`, `profissao`, `nacionalidade text DEFAULT 'Brasileiro(a)'`, `data_nascimento date`, `razao_social`, `nome_fantasia`, `representante_nome`, `representante_cpf_enc bytea`, `representante_cargo`, `logradouro`, `numero`, `complemento`, `bairro`, `cep`, `cidade`, `uf char(2)`, `email`, `telefone`, `observacoes`, além das já existentes `documento_enc`/`documento_idx` (CPF **ou** CNPJ conforme o tipo), `retencao_ate`. `CHECK`: PF exige `nome_completo`; PJ exige `razao_social`. `nome_exibicao` gerado (`coalesce`) para busca/ordenacão; índice em `lower(nome_exibicao)`.
- `processos`: `codigo UNIQUE`, `numero_cnj` passa a **aceitar vazio** (B: "Em branco se ainda não distribuído") — índice único parcial só para não vazios; `situacao CHECK IN ('ativo','sobrestado','extinto','arquivado')`; novos: `titulo`, `polo`, `sistema`, `instancia`, `classe_processual`, `materia_direito`, `comarca`, `juizo`, `tipo_contingencia CHECK IN ('Ativa','Passiva','')`, `valor_causa numeric(16,2)`, `distribuicao`, `citacao`, `audiencia`, `sentenca`, `recursos`, `fase_execucao` (todas `date`), `atendimento_origem_id uuid NULL` (FK criada no Plano 3), `projeto_vinculado text`, `observacoes`, `prox_o_que`, `prox_ate_quando date`, `prox_quem`. Remover `orgao` e `responsavel_id`.
- `processo_partes`: `papel text NOT NULL` (livre: Autor, Réu, Exequente, Executado, Embargante, Terceiro interessado, Assistente ou texto de "Outro"), `cliente boolean NOT NULL DEFAULT false`; chave `(processo_id, pessoa_id, papel)`.
- `processo_responsaveis(processo_id, pessoa_id, ordem int)` (até 4 na UI, sem limite no banco).
- `processo_andamentos(id uuid, processo_id, data date NOT NULL, o_que text NOT NULL, documento text NOT NULL DEFAULT '', created_*)`.
- `processo_financeiro(id uuid, processo_id, item text NOT NULL, valor text NOT NULL DEFAULT '', situacao text NOT NULL DEFAULT '', created_*)`.
- `codigos_contador(prefixo text PRIMARY KEY, ultimo bigint NOT NULL)` — privilégios ao papel `painel_app` conforme a migração 0003.
- Down: documentar que não é reversível sem perda (`RAISE EXCEPTION` se houver linhas).

- [ ] **Step 1: testes que falham** — (a) `codigos.Proximo` devolve `PES. 0000001`, `PES. 0000002`, `PRO. 0000001`; duas transações concorrentes não repetem; rollback não consome número; (b) migração: inserir PF sem `nome_completo` → violação; PJ sem `razao_social` → violação; `situacao='suspenso'` → violação; dois processos com `numero_cnj=''` → permitido; dois com o mesmo CNJ não vazio → violação; (c) o papel `painel_app` consegue `INSERT/UPDATE` nas novas tabelas e **não** consegue `DELETE` em `auditoria`.
- [ ] **Step 2–4:** ver falhar, escrever migração e pacote, passar (`go test -p 1 ./internal/app ./internal/platform/codigos`).
- [ ] **Step 5:** `restore-check` continua passando. **Step 6: commit.**

## Task 2: Domínio Pessoas com os campos do B

**Files:** Modify `internal/pessoas/{model,repo,service,handler}.go` (dividir em arquivos ≤ 300 linhas se preciso: `model.go`, `qualificacao.go`, `repo.go`, `service.go`, `handler.go`, `papeis.go`), `internal/pessoas/templates/*.html`; Test em `internal/pessoas/*_test.go` e `internal/app/pessoas_test.go`.

**Interfaces — Produces:**
```go
type Pessoa struct { ID, Codigo, TipoPessoa, Nome string /* nome_exibicao */; Documento string /* já decifrado ou mascarado */; … todos os campos do B … }
type Categoria string // "cliente","parte_contraria","terceiro_interessado","sem_processo"
// Categoria vem dos processos (por interface, sem importar o domínio):
type PapeisLookup interface {
	PapeisPorPessoa(ctx context.Context, ids []string) (map[string][]PapelEmProcesso, error)
}
type PapelEmProcesso struct { ProcessoID, ProcessoNumero, ProcessoTitulo, Papel string; Cliente bool; Responsavel bool }
func CategoriaDe(p []PapelEmProcesso) Categoria // regra de categoriaPapelPessoa do B: algum Cliente→cliente; algum papel "Terceiro interessado"→terceiro_interessado; algum→parte_contraria; nenhum→sem_processo
type Opcao = webx.Opcao // para selects de pessoa em outros domínios:
func (s *Service) Opcoes(ctx context.Context) ([]Opcao, error) // rótulo igual ao B: "NOME — Física (123.456.789-09)"
```
**Tela e drawer (literais do B):** tópico `Pessoas` / sub "Cadastro de clientes, partes adversas e terceiros interessados — art. 319 do CPC."; botão "Nova pessoa"; busca "Buscar por nome…"; acordeão **Cliente, Parte adversa, Terceiro interessado, Sem processo vinculado** (vários abertos); tabela `Código | Nome | Tipo | Documento | (ações Editar / Excluir)`; estados vazios do B ("Nenhuma pessoa cadastrada.", "Nenhuma pessoa nesta categoria."); drawer "Nova pessoa" — "Qualificação nos termos do art. 319 do CPC." — com **Tipo de pessoa** (radios Física/Jurídica; no editar ficam desabilitados com a dica "Tipo de pessoa não se troca depois de criada — exclua e recrie, se for o caso."), `fieldset` Pessoa física (Nome completo, CPF, RG (+ órgão expedidor), Estado civil, Profissão, Nacionalidade, Data de nascimento), `fieldset` Pessoa jurídica (Razão social, Nome fantasia, CNPJ, Representante legal, CPF do representante, Cargo), `fieldset` Endereço (Logradouro, Número, Complemento, Bairro, CEP + "Buscar CEP", Cidade, UF), E-mail, Telefone, Observações, ações "Cancelar" / "Salvar cadastro" (editar: "Salvar alterações"). Nomes em **maiúsculas** (`data-maiusculo`), máscaras `data-mask`, validação de CPF/CNPJ (`data-validate`). Toast de sucesso "Pessoa criada — código PES. 0000001." e "Pessoa atualizada."; erros do B ("CPF inválido — confira os dígitos informados." etc.).
**Regras do v2:** CPF/CNPJ duplicado → mensagem de conflito (mesma resposta do B para duplicidade: "Já existe uma pessoa com esse documento."); `Reveal` de documento auditado; edição com `version` (concorrência otimista); `data-protege` no formulário.

- [ ] **Step 1: testes que falham** (serviço, com banco): criar PF e PJ; código sequencial; `Nome` em maiúsculas; CPF inválido recusado no servidor também; duplicidade de CPF; `CategoriaDe` com os 4 casos do B; editar não troca o tipo; exclusão lógica some das listas; listagem sem permissão de documento vem mascarada; auditoria registra `pessoa.create/update/delete`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: handler/templates** — fragmentos de drawer (`GET /pessoas/nova`, `GET /pessoas/{id}/editar` com `HX-Request`), página `GET /pessoas` com acordeão e `data-filtro`; `POST` por `data-drawer-form` responde `X-Painel-Ok`. **Step 6:** testes de handler: fragmento começa por `<h2>`; página sem `on*=`/`style=`; só quem escreve vê "Nova pessoa" e botões; CSRF exigido. **Step 7: commit.**

## Task 3: Domínio Processos com os campos e as ações do B

**Files:** Modify `internal/processos/*` (dividir em `model.go`, `partes.go`, `andamento.go`, `pendencias.go`, `repo.go`, `service.go`, `handler.go`, `acoes.go`), templates; Test `internal/processos/*_test.go`, `internal/app/processos_test.go`.

**Interfaces — Consumes:** `pessoas.Service.Opcoes`, `codigos.Proximo`. **Produces:**
```go
type Parte struct{ PessoaID, Papel string; Cliente bool }
type Pendencias func(Processo) []string // lista na ordem fixa da tabela de decisões
func (s *Service) PapeisPorPessoa(ctx context.Context, ids []string) (map[string][]PapelEmProcesso, error) // implementa pessoas.PapeisLookup via adaptador no app.go
func (s *Service) MoverEstado(ctx, ac, id string, version int, novo, motivo string) error
func (s *Service) RegistrarAndamento(ctx, ac, id string, data time.Time, oQue, documento string) error
func (s *Service) AtualizarFinanceiro(ctx, ac, id, item, valor, situacao string) error
func (s *Service) ProximaProvidencia(ctx, ac, id, oQue string, ateQuando *time.Time, quem string) error
func (s *Service) Opcoes(ctx context.Context) ([]Opcao, error) // para o Plano 3 (Tarefas): "PRO. 0000012 · NOME DO CLIENTE"
```
**Tela e drawers (literais do B):** `Processos` / sub do B; botão "Novo processo"; busca "Buscar por número, classe, matéria ou parte…"; acordeão **Ativos, Sobrestados, Extintos, Arquivados definitivamente**; colunas `Código | Nº processo | Classe / Matéria | Cliente | Parte contrária | Valor da causa | Distribuição | Pendências | (ações)` com as ações **Editar, Mover estado, Andamento, Financeiro, Próxima providência, Excluir**; drawer "Novo processo" (subtítulo do B) com `fieldset Partes` (linhas: Pessoa / Papel processual + "Outro" revela texto / "É cliente do escritório" / "Remover parte"; botão "+ Adicionar parte"; hint do B), `fieldset Responsáveis pelo cadastro` (4 selects "Responsável 1…4"), Número do processo (CNJ) + Valor da causa, Sistema (+Outro), Instância (+Outro) com as listas `SISTEMAS_PROCESSO`/`INSTANCIAS_PROCESSO` do B, Classe processual, Matéria / ramo do direito (listas `OPCOES` do B, `05-dominio.js`), Comarca, Juízo, Contingência (radios Ativa/Passiva/Não se aplica), `fieldset Linha do tempo` (Distribuição, Citação, Audiência / Sentença, Recursos, Fase de execução (satisfativa) + hint), Atendimento de origem (opcional; fica **desabilitado com a dica "Disponível quando o módulo Atendimentos for ativado" até o Plano 3**), Observações; editar acrescenta Título da ficha, Polo, Projeto vinculado e a dica "Estado (ativo/sobrestado/...) não se edita aqui — use "Mover estado" na lista.". Drawers de ação com os textos do B: "Mover estado do processo", "Registrar andamento", "Atualizar financeiro", "Próxima providência" (campos, placeholders e datalist idênticos). Cliente/Parte contrária exibidos como no B (primeira parte cliente; demais partes). Toasts do B ("Processo criado.", "Processo atualizado.", "Estado do processo atualizado.", "Andamento registrado.", "Dado financeiro atualizado.", "Próxima providência registrada.", "Processo excluído.").
**Regras do v2:** número CNJ validado (20 dígitos + dígito verificador CNJ — módulo 97) quando informado; valor da causa em `numeric`; `version` para concorrência; auditoria de cada ação; exclusão lógica com o texto de segurança da seção Decisões; processo com tarefas/prazos abertos pede confirmação extra ("Há N tarefa(s)/prazo(s) em aberto vinculados.") — preparação para os Planos 3 e 7.

- [ ] **Step 1: testes que falham:** pendências (processo vazio lista todas, na ordem; processo completo lista nenhuma; "parte cliente" conta); CNJ inválido recusado e válido aceito (usar CNJ de teste com dígito calculado); criar com 2 partes (uma cliente) e 2 responsáveis; `Cliente`/`Parte contrária` derivados; mover estado grava motivo e auditoria; andamento ordenado por data desc; financeiro; próxima providência; editar com versão antiga → conflito; exclusão lógica.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** handlers/templates (fragmentos de drawer para cada ação; página com acordeão e `data-filtro`). **Step 6:** testes de handler (sem inline, permissões, CSRF, "Outro" revela campo via `data-mostra`). **Step 7:** adaptador em `internal/app/app.go` ligando `processos.Service` a `pessoas.PapeisLookup`; `fronteiras_test.go` verde. **Step 8: commit.**

## Task 4: Seed fictício, instantâneos e paridade

- [ ] **Step 1:** `internal/app/seed.go`: 4 pessoas fictícias (PF/PJ), 3 processos (um em cada de 3 estados) com partes/responsáveis, andamentos e financeiro; `seed_test.go` verde.
- [ ] **Step 2:** estender `internal/app/instantaneos_test.go` e `tools/refb/fixtures.py` (**mesmos dados fictícios nos dois lados**) para Pessoas e Processos, com o drawer "Nova pessoa" e o "Novo processo" abertos.
- [ ] **Step 3:** rodar `tools/refb/paridade.js` no B (porta 18787) e nos instantâneos do v2 para: acordeão aberto/fechado, tabelas de Pessoas e de Processos, `.badge-warn` das pendências, `fieldset`/`legend`, `.row2/.row3`, `.radio-row`, `.field label/input/select/textarea`, `.hint`, botões `.btn-text`/`.btn-danger-text`/`.btn-ghost`. **Zero diferença não justificada.** Capturas em 1440 e 400 de largura se o Chrome permitir (senão, dizer que não foi possível).
- [ ] **Step 4:** comparar **texto a texto** os rótulos/placeholders/hints/toasts do B (extraídos por `grep` dos módulos 14 e 09) com os do v2 (extraídos dos templates): script `tools/refb/textos.py` imprime as diferenças; zerar as não justificadas.
- [ ] **Step 5:** `docs/arquivo/plano-fusao/paridade/RELATORIO-2.md`, `docs/arquivo/plano-fusao/NOTAS-PLANO-2.md`, `ACEITE-VISUAL.md`, ficha no acervo.
- [ ] **Step 6: commit.**

## Task 5: Verificação final

- [ ] `gofmt -l .` vazio; `go vet ./...`; `go test -race -p 1 ./...` (em contêiner `golang:1.26.10-bookworm` com Postgres descartável, como no Plano 1B); `staticcheck`/`gosec` se instalados; `scripts/check-lines.sh`.
- [ ] Reconstruir só o app dev (sem `down -v`); a migração 0008 roda sobre o banco dev (vazio de dados do negócio; só o administrador); `GET /login` 200; **não usar credenciais reais**; remover contêineres de teste.
- [ ] Hash da pasta do B inalterado (`sha256sum -c docs/plano-fusao/HASHES-B.txt`).

## Self-review

- Cobertura do B: 14-pessoas (T2), 09-processos (T3), 06 (linhas de parte/responsável em T3), 05 (listas em T3), 03 (campos), 15/07 (drawer/acordeão do Plano 1B).
- Itens do B **não** reproduzidos, por regra do v2: texto "definitivamente" (exclusão lógica), "Nome do arquivo da ficha", efeito de "monitoramento em cadeia" (Plano 6), CEP real (Plano 8). Todos listados nas Decisões.
- Sem placeholders: onde o B tem o texto, o plano manda copiar literalmente e T4 prova por comparação automática.
