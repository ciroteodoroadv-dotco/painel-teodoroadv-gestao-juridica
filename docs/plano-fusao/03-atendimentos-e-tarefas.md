# Fusão — Plano 3: Atendimentos e Tarefas idênticos ao painel B

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever qualquer template, leia por inteiro, linha por linha, no B (somente leitura):** `referencia/painel-b/js/10-atendimentos.js`, `referencia/painel-b/js/08-tarefas.js`, `referencia/painel-b/js/06-helpers-vinculos.js`, `referencia/painel-b/js/03-api-mapa.js`, `referencia/painel-b/js/05-dominio.js`, `referencia/painel-b/js/07-app-core.js`, `referencia/painel-b/js/15-drawers.js`, `referencia/painel-b/js/02-util.js`, `referencia/painel-b/css/painel.css`, e releia `14-pessoas.js` (o drawer "Nova pessoa" é aberto de dentro do atendimento). **O B vale na forma; o v2 vale na regra.** Rótulos, ordem dos campos, dicas, placeholders, textos de subtítulo/toast, opções de select, valores padrão e ordem das colunas são **copiados literalmente**. Use como modelo de método o Plano 2 (`02-pessoas-e-processos.md`, `docs/arquivo/plano-fusao/NOTAS-PLANO-2.md`, `tools/refb/textos.py`).

**Goal:** As telas **Atendimentos** e **Tarefas** do v2 passam a ter os mesmos campos, ordem, rótulos, drawers, acordeão e ações do B. A tarefa **nasce vinculada a um atendimento ou a um processo** (obrigatório), tem 7 estados, prioridade Urgente/Alta/Normal/Baixa, Partes e Responsáveis como listas dinâmicas, **Prazo fatal como campo da tarefa**, "Tarefa vinculada" (filha), documentos e exclusão com reautenticação.

**Architecture:** Migração `0009` cria `atendimentos` e `atendimento_pessoas`, remodela `tarefas` (tabela hoje sem dados de negócio; o v2 começa limpo) e cria `tarefa_partes`/`tarefa_responsaveis`; liga `processos.atendimento_origem_id` ao novo domínio. O **prazo fatal** da tarefa é espelhado no domínio `prazos` por **interface** (`PrazoSync`), de modo que a regra de ouro continua valendo: o selo **A CONFERIR** aparece **no mesmo campo** do prazo fatal enquanto um advogado não confirmar. Domínios só se falam por interfaces (`fronteiras_test.go` verde).

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, JS do Plano 1B (`drawer.js`, `forms.js`, `mascaras.js`).

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `01b-casca-identica-ao-b.md`, `02-pessoas-e-processos.md`.

## Global Constraints

- Tudo dos Planos 1B e 2: CSP estrita (sem `on*=`, `style=`, `innerHTML`, `eval`); arquivos de produção ≤ 300 linhas (`scripts/check-lines.sh`); `gofmt`, `go vet`, `go test -race -p 1 ./...` (contêiner `golang:1.26.10-bookworm` + Postgres descartável); código de ficha por `codigos.Proximo` (`ATE. 0000001`, `TAR. 0000001`); exclusão lógica auditada; `version` para concorrência otimista; pasta do B somente leitura; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.
- **Estados da tarefa (slugs e rótulos do B):** `a-confirmar` "A confirmar", `pendente` "Pendente", `em-execucao` "Em execução", `revisao` "Revisão", `concluidas-com-sucesso` "Concluída com sucesso", `concluidas-sem-sucesso` "Concluída sem sucesso", `canceladas` "Cancelada". Selos (`ROTULOS_ESTADO_TAREFA`): a-confirmar/pendente `badge-warn`; em-execucao/revisao `badge-neutral`; concluída `badge-ok`; sem êxito `badge-neutral`; cancelada `badge-danger`.
- **Prioridade:** Urgente, Alta, Normal (padrão), Baixa. Classe do selo: Urgente→`prioridade-alta`, Alta→`prioridade-media`, Baixa→`prioridade-baixa`, demais→`prioridade-normal` (função `prioridadeBadgeClass` do B).
- Acordeão de Tarefas: **vários painéis abertos ao mesmo tempo**, nenhum aberto de início (como `state.accOpen={}`); contador por estado.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| **Prazo fatal da tarefa** | Campo "Prazo fatal" idêntico ao do B. Ao salvar com data, o serviço chama `PrazoSync.Sincronizar` → cria/atualiza o `prazo` ligado à tarefa. **Advogado** salva → prazo `confirmado`; **não advogado** → `a_conferir` (regra de ouro existente). Na tabela, o selo do B `Prazo fatal: dd/mm/aaaa` (`badge-danger`) ganha, **dentro do mesmo selo/linha**, o selo `A CONFERIR` (`badge-warn`) enquanto a situação for `a_conferir`; quem tem `PermPrazoConfirm` vê um botão `btn-text` "Confirmar prazo" ao lado, que usa o fluxo de confirmação do domínio Prazos. Data apagada, tarefa cancelada/concluída ou excluída → prazo é cancelado/cumprido/removido conforme o estado, sempre pelo domínio Prazos. O sistema **não calcula** a data. |
| Mover para "a-confirmar" → outro estado | Só quem tem `PermPrazoConfirm` (advogado) tira uma tarefa de "A confirmar" (a tarefa criada por automação — Plano 5 — nasce aí). Demais recebem a mensagem "Somente advogado confirma tarefas a confirmar." |
| Excluir tarefa (B: "Senha de administrador") | Mesmo drawer e mesmos textos do B, **exceto** o rótulo do campo e a frase de confirmação, que passam a "Sua senha *" e "Excluir esta tarefa? Ela deixa de aparecer nas listas e a exclusão fica registrada na auditoria. Confirme com a sua senha." O servidor **reautentica** a senha do usuário logado (Argon2id), exige `PermTarefaDelete` (advogado/administrador), registra na auditoria e faz exclusão lógica. Não existe "senha de administrador" única no v2. |
| "Documentos do caso" (PDF) | Campo e posição idênticos; o arquivo vai para o domínio `documentos` (cifrado, tipo conferido pelo conteúdo, auditado), ligado ao processo (se o vínculo for processo) ou ao atendimento (se for atendimento) e à tarefa. A dica do B ("…guardado em 01-documentos do caso e convertido para .md.") vira "O PDF será guardado, cifrado, junto ao caso." (a conversão em `.md` não existe no v2). O envio é imediato ao escolher o arquivo (como no B), por `data-upload`; toast "Documento guardado." O fluxo "dados extraídos do recibo → abre Novo processo" **não é reproduzido** (anotar em `docs/arquivo/plano-fusao/NOTAS-PLANO-3.md`). |
| "Tarefa vinculada" (filha) | Idêntico: abre "Nova tarefa" **sobreposta** (`data-drawer-sobrepor`) com vínculo, partes e responsáveis do pai; guarda `tarefa_pai_id`; a lista mostra o código do pai quando houver. |
| Responsáveis / Partes / Criado por | **Pessoas** (selects de pessoa do Plano 2, rótulo igual ao B), como no B; "Criado por" com a dica "Pessoa que criou este registro. Não define quem executará a atividade." |
| "Nome do arquivo"/slug | Não existe no B para Tarefa/Atendimento além do técnico `nomeArquivo` gerado no código; **não** há campo na tela → nada a omitir. |
| Atendimento: tipo/status | O B mantém `atendimento_tipo`/`status` só no mapeamento; a tela **não** os exibe → o v2 guarda `tipo` ('Extrajudicial'/'Triagem') e `status` ('Em triagem') com valores padrão, sem campo na tela. |
| Botão "+ Novo cliente"/"+ Nova parte adversa" no Atendimento | Abre o drawer "Nova pessoa" **sobreposto**, preserva o rascunho do atendimento e, ao salvar a pessoa, volta ao atendimento com a nova pessoa já selecionada (`App.voltarPessoaAtendimento`). Cancelar volta ao rascunho. |
| Processo → "Atendimento de origem" | Passa a funcionar: lista `ATE. 0000001 · tipo · CLIENTE · dd/mm/aaaa`; some a dica de "desabilitado" do Plano 2. |

---

## Task 1: Migração 0009

**Files:** Create `migrations/0009_atendimentos_e_tarefas_do_b.sql`; Test `internal/app/migracoes_0009_test.go`.

**Esquema:**
- `atendimentos(id uuid pk, codigo text unique not null, tipo text not null default 'Extrajudicial' check in ('Extrajudicial','Triagem'), status text not null default 'Em triagem' check in ('Em triagem','Em andamento','Concluído','Convertido em processo'), descricao text not null, observacoes text not null default '', data_abertura date not null, criado_por_pessoa_id uuid null references pessoas, excluido_em, created_*/updated_*/version)`.
- `atendimento_pessoas(atendimento_id, pessoa_id, papel text check in ('cliente','parte_adversa'), ordem int, primary key (atendimento_id, pessoa_id, papel))`.
- `ALTER TABLE processos ADD CONSTRAINT … FOREIGN KEY (atendimento_origem_id) REFERENCES atendimentos(id)`.
- `tarefas` **recriada** (`DROP` + `CREATE`, só se vazia — `DO $$ … RAISE EXCEPTION` se houver linhas): `id, codigo unique, titulo (2..300), subtitulo (≤80, default ''), estado text check in (os 7 slugs) default 'pendente', origem_tipo text check in ('atendimento','processo'), atendimento_id uuid null, processo_id uuid null, check ((origem_tipo='atendimento' and atendimento_id is not null and processo_id is null) or (origem_tipo='processo' and processo_id is not null and atendimento_id is null)), tarefa_pai_id uuid null references tarefas, criado_por_pessoa_id uuid null, descricao text not null default '', observacoes text not null default '', data_base date, prazo date, prazo_fatal date, prioridade text check in ('Urgente','Alta','Normal','Baixa') default 'Normal', excluido_em, created_*/updated_*/version`; índices por `estado` e `(processo_id)`, `(atendimento_id)`.
- `tarefa_partes(tarefa_id, pessoa_id, papel text, ordem int)`; `tarefa_responsaveis(tarefa_id, pessoa_id, ordem int)`.
- Se a tabela `prazos` exigir `processo_id`, torná-lo **opcional** e adicionar `tarefa_id uuid null references tarefas` (índice único parcial `(tarefa_id) where tarefa_id is not null and excluido_em is null`); `documentos` ganha `atendimento_id uuid null` e `tarefa_id uuid null` (FKs). Conferir os nomes reais nas migrações 0004–0007.
- Privilégios do papel `painel_app` como na 0003; classificação LGPD documentada no cabeçalho (descricao/observacoes = PESSOAL).

- [ ] **Step 1: testes que falham** — inserções válidas e inválidas (tarefa sem vínculo; com os dois vínculos; estado fora da lista; prioridade fora da lista; subtítulo > 80); `prazos` aceita linha sem processo e com `tarefa_id`; `documentos` aceita `atendimento_id`; `painel_app` insere/atualiza e **não** apaga `auditoria`.
- [ ] **Step 2–4:** ver falhar, escrever, passar. **Step 5:** `restore-check` continua passando. **Step 6: commit.**

## Task 2: Domínio Atendimentos

**Files:** Create `internal/atendimentos/{model,repo,service,handler,formulario}.go` e `templates/{lista,drawer}.html`; Modify `internal/app/app.go` (registro, `AddNavCount`, permissões `PermAtendimentoRead/Write` no RBAC: advogado e assistente escrevem; leitura para todos os papéis que leem processos); Test `internal/atendimentos/*_test.go`, `internal/app/atendimentos_test.go`.

**Interfaces — Produces:**
```go
type Atendimento struct{ ID, Codigo, Tipo, Status, Descricao, Observacoes string; DataAbertura time.Time; CriadoPorID, CriadoPor string; Clientes, PartesAdversas []PessoaRef; Version int }
type PessoaRef struct{ ID, Codigo, Nome string }
type PessoaLookup interface { Opcoes(ctx) ([]Opcao, error); Existem(ctx, ids []string) (map[string]bool, error); Nomes(ctx, ids []string) (map[string]PessoaRef, error) }
func (s *Service) Opcoes(ctx context.Context) ([]Opcao, error) // p/ Processos e Tarefas: "ATE. 0000001 · NOME DO CLIENTE"
func (s *Service) Existe(ctx context.Context, id string) (bool, error)
func (s *Service) Resumo(ctx context.Context, id string) (codigo, cliente string, err error)
```
**Tela e drawer (literais do B, módulo 10):** `Atendimentos` / sub "Registro das demandas e dos atendimentos do escritório."; botão "Novo atendimento"; busca `Buscar por código, cliente, parte adversa ou objeto…` (filtra ao digitar, sem acento); tabela `Código | Cliente(s) | Parte(s) adversa(s) | Descrição / objeto | Data do cadastro | (Editar)` (classe `tabela-atendimentos`, nomes em MAIÚSCULAS sem "(CPF…)"; "Criado por: …" como `hint` sob a descrição); vazios "Nenhum atendimento encontrado nesta busca." / "Nenhum atendimento cadastrado. Use Novo atendimento para registrar uma demanda."; drawer "Novo atendimento"/"Editar atendimento" — "Registro da demanda. Cada novo objeto deve ter seu próprio atendimento." — com **Criado por**, `fieldset Cliente(s)` e `fieldset Parte(s) adversa(s)` (campo de busca "Buscar pessoa por nome ou código PES…", linhas com select "Selecione uma pessoa" + "Remover", botões "+ Adicionar cliente"/"+ Novo cliente" e equivalentes de parte adversa), **Descrição / objeto** (obrigatória), **Data do cadastro** (obrigatória; padrão hoje), **Observações complementares**; "Cancelar"/"Salvar atendimento" ("Salvar alterações" ao editar). Validação "Selecione ao menos um cliente."; toasts "Atendimento criado." / "Atendimento atualizado.". **Sem exclusão** (o B não tem).

- [ ] **Step 1: testes que falham** (serviço com banco): criar com 1 cliente + 1 parte adversa; código `ATE. 0000001`; sem cliente → erro; pessoa inexistente → erro; editar substitui as listas; `version` antiga → conflito; busca sem acento; auditoria `atendimento.criado/atualizado`; permissões.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** handlers/templates: `GET /atendimentos` (página), `GET /atendimentos/novo` e `/{id}/editar` (fragmento com `HX-Request`), `POST`s por `data-drawer-form`; `forms.js`: contexto de pessoa criada de dentro do atendimento (`data-nova-pessoa="clientes|partesAdversas"`: salva o rascunho em memória, abre "Nova pessoa" sobreposta, e ao receber `X-Painel-Pessoa-Id` volta ao rascunho com a pessoa adicionada). **Step 6:** testes de handler (sem `on*=`/`style=`; só quem escreve vê botões; CSRF). **Step 7:** habilitar o select "Atendimento de origem" no processo (Plano 2) por interface `AtendimentoLookup`; teste. **Step 8: commit.**

## Task 3: Domínio Tarefas — modelo, serviço e prazo fatal

**Files:** Modify `internal/tarefas/*` (dividir: `model.go`, `estados.go`, `prioridade.go`, `repo.go`, `repo_listas.go`, `service.go`, `prazo.go`, `exclusao.go`, `documentos.go`); Modify `internal/prazos` (interface de sincronização) e `internal/documentos` (vínculo com tarefa/atendimento); Test `internal/tarefas/*_test.go`, `internal/app/tarefas_test.go`.

**Interfaces — Produces:**
```go
type Tarefa struct{ ID, Codigo, Titulo, Subtitulo, Estado, Prioridade, OrigemTipo, OrigemID, OrigemCodigo, ClienteNome, Descricao, Observacoes, TarefaPaiCodigo string
	DataBase, Prazo, PrazoFatal *time.Time; PrazoFatalSituacao string /* ""|"a_conferir"|"confirmado"|"cumprido"|"cancelado" */
	CriadoPorID, CriadoPor string; Partes []ParteTarefa; Responsaveis []PessoaRef; Version int }
type Estado struct{ Chave, Rotulo, Classe string }
var Estados = []Estado{…os 7 na ordem do B…}
func PrioridadeClasse(p string) string // regra do B
// Contratos com outros domínios (interfaces; implementações ligadas em internal/app):
type PrazoSync interface {
	Sincronizar(ctx context.Context, ac webx.Actor, tarefaID, processoID, descricao string, data *time.Time) (situacao string, err error)
	Situacoes(ctx context.Context, tarefaIDs []string) (map[string]string, error)
	Encerrar(ctx context.Context, ac webx.Actor, tarefaID, motivo string) error // concluída/cancelada/excluída
}
type VinculoLookup interface { // atendimentos + processos
	Resumo(ctx context.Context, tipo, id string) (codigo, cliente string, err error)
	Opcoes(ctx context.Context, tipo string) ([]Opcao, error)
}
type DocumentoSink interface { Anexar(ctx context.Context, ac webx.Actor, tarefaID, origemTipo, origemID, nome string, conteudo io.Reader) error }
type Reauth interface { Confirmar(ctx context.Context, ac webx.Actor, senha string) error }
func (s *Service) AbertasPorSubtitulo(ctx, q string, limite int) ([]Resumo, error) // para a Agenda (Plano 4)
```
**Regras:** criar exige título e vínculo (tipo + id existente); `estado` inicial padrão `pendente`; mover exige a regra de "A confirmar"; concluída/cancelada chamam `PrazoSync.Encerrar`; `prazo_fatal` ≠ nulo chama `Sincronizar`; apagar a data cancela o prazo; filha herda vínculo/partes/responsáveis só como **pré-preenchimento** (a tela os envia; o servidor valida de novo); busca textual em título, subtítulo, código, cliente, origem, responsáveis, partes, descrição, observações (sem acento/caixa); exclusão exige `Reauth.Confirmar` + `PermTarefaDelete`.

- [ ] **Step 1: testes que falham:** (a) criar tarefa vinculada a atendimento e a processo; (b) sem vínculo → erro; vínculo inexistente → erro; (c) advogado salva com prazo fatal → situação `confirmado`; assistente → `a_conferir`; mesmo prazo não duplica ao editar; (d) assistente não tira de `a-confirmar`; advogado tira; (e) concluir/cancelar encerra o prazo; (f) apagar a data cancela o prazo; (g) filha grava `tarefa_pai_id` e mostra `TAR.` do pai; (h) exclusão com senha errada falha e **não** exclui; com senha certa exclui logicamente e audita; sem permissão → 403; (i) busca sem acento encontra por cliente e por código; (j) `PrioridadeClasse` com os 4 valores; (k) `AbertasPorSubtitulo` ignora concluídas/canceladas/excluídas e compara sem acento ("ligacao"/"ligacoes", "reuniao"/"reunioes", "aula"/"aulas").
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** adaptadores em `internal/app/app.go` ligando `prazos`, `documentos`, `atendimentos`, `processos` e `auth` (reautenticação) às interfaces; `fronteiras_test.go` verde. **Step 6: commit.**

## Task 4: Tarefas — tela, drawers e ações (idênticos ao B)

**Files:** Modify `internal/tarefas/{handler,formulario,lista}.go` e `templates/{lista,drawer,excluir}.html`; Modify `web/static/forms.js` (+ novos arquivos ≤ 300 linhas se preciso: `upload.js`); Test `internal/tarefas/handler_test.go`, `internal/app/tarefas_tela_test.go`.

**Tela (literais do B, módulos 08 e 06):** `Tarefas` / sub "Toda tarefa nasce de um atendimento (ordem de serviço) ou de um processo."; botão "Nova tarefa"; barra de busca `<form class="toolbar tarefa-search">` com `input.search` "Buscar por título, cliente, código ou responsável…" e botão "Buscar" (`btn btn-primary btn-small`) — **busca ao enviar** (GET `?q=`); acordeão dos 7 estados (vários abertos); tabela `Código | Tarefa | Ações` (`scope="col"`); célula Tarefa: `<strong>título</strong>`, `.tarefa-subtitulo`, "Criado por: …" (`hint`), `.tarefa-client` (nome do cliente do vínculo), grade `.tarefa-meta` com **Origem** (código ATE./PRO.), **Responsável** (nomes separados por vírgula), **Prazo**, **Prioridade** (selo), selo `badge badge-danger` "Prazo fatal: dd/mm/aaaa" **+ `A CONFERIR`** conforme a decisão, descrição como `hint`; ações: `select` "Mover para..." (envia ao mudar: `data-submit-on-change`) com os outros 6 estados, "Editar", "Tarefa vinculada", "Excluir" (`btn-danger-text`). Vazios: "Nenhuma tarefa neste estado." e, sem nenhuma tarefa, `h3` "Nenhuma tarefa cadastrada" + `Crie a primeira tarefa pelo botão "Nova tarefa" acima.`.
**Drawer "Nova tarefa"** (ordem exata do B): subtítulo "Toda tarefa nasce vinculada a um atendimento ou a um processo já existente."; **Vínculo \*** (radios Atendimento / Processo; um é desabilitado se não há registros; o inicial é Atendimento se houver, senão Processo), **Vínculo específico \*** (select `ATE. … · CLIENTE`/`PRO. … · CLIENTE`; troca ao mudar o radio por `data-origem-opcoes`), `tarefaPai` oculto, **Título \*** (placeholder "Ex.: Protocolar embargos à execução"), **Subtítulo** (datalist: Ligação, Reunião, Aula, Análise de decisão, Monitoramento; hint "Tarefas de Ligação, Reunião ou Aula aparecem também na categoria correspondente da agenda."), `fieldset Partes` (linhas Pessoa/Papel(+Outro)/"Remover parte"; botão "+ adicionar parte" `btn-text`), **Descrição**, linha `row2`: **Criado por** (com a dica), **Data base**, **Prazo de entrega**; **Prazo fatal**; `fieldset Responsáveis existentes` (linhas "Responsável" + "Remover"; "+ adicionar responsável"); `row2`: **Status** (padrão Pendente), **Prioridade** (padrão Normal); **Observações**; ações "Cancelar" / "Salvar tarefa". **Drawer "Editar tarefa"**: subtítulo = título; **Vínculo específico** como texto fixo (`hint`), Título, Subtítulo (hint do B), Partes, Descrição (placeholder "Registre aqui as atualizações da tarefa"), Responsáveis existentes, Criado por, Data base, `row2` Prazo de entrega / Prazo fatal, `row2` Prioridade / Status, Observações, **Documentos do caso** (`input type=file accept=.pdf,application/pdf`, dica adaptada), "Cancelar" / "Salvar alterações". **Drawer "Excluir tarefa"** conforme Decisões. Toasts do B: "Tarefa criada.", "Tarefa atualizada.", "Estado da tarefa atualizado.", "Tarefa excluída.", "Título é obrigatório.", "Selecione o vínculo (Atendimento ou Processo).", "Selecione o vínculo específico.", "Falha ao criar a tarefa: …", "Não foi possível excluir a tarefa: …".

- [ ] **Step 1: testes que falham** (handlers): página `GET /tarefas` contém, **nesta ordem**, os 7 `acc-head` com os rótulos exatos; busca `?q=` filtra; fragmento de "Nova tarefa" contém os rótulos do B **na ordem do B** (teste que extrai todos os `<label>`/`<legend>` e compara com a lista literal); "Editar" não permite trocar o vínculo; "Tarefa vinculada" pré-preenche; "Mover para…" por POST muda o estado e responde `X-Painel-Ok: Estado da tarefa atualizado.`; upload de PDF válido anexa e PDF falso (extensão `.pdf` com conteúdo de imagem) é recusado; HTML sem `on*=`/`style=`; sem `PermTarefaWrite` não há botões; assistente vê `A CONFERIR` no prazo fatal que ele mesmo salvou; advogado vê "Confirmar prazo".
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** `forms.js`: `data-submit-on-change`, `data-origem-opcoes`, `data-upload`, `data-drawer-sobrepor` na "Tarefa vinculada" (reusa o contrato do Plano 1B). **Step 6:** contador do menu "Tarefas"/"Atendimentos". **Step 7: commit.**

## Task 5: Seed, paridade B × v2 e comparação de textos

- [ ] **Step 1:** seed fictício (3 atendimentos; 6 tarefas, uma por estado relevante, com prazo fatal, subtítulo Ligação/Aula e uma filha), e a **mesma massa** em `tools/refb/fixtures.py` (o B já tem fixtures de tarefa/atendimento do Plano 1B; ajustar aos formatos de `mapearTarefaAPI`/`mapearAtendimentoAPI`).
- [ ] **Step 2:** instantâneos do v2: Atendimentos (página, "Novo", "Editar", "Nova pessoa" sobreposta), Tarefas (página com 2 estados abertos, "Nova", "Editar", "Excluir", "Tarefa vinculada").
- [ ] **Step 3:** `tools/refb/paridade.js` no B (porta 18787) e no v2: acordeão, `.tarefa-info`, `.tarefa-meta*`, `.badge*`, `.prioridade-*`, `.row-actions select`, `.toolbar.tarefa-search`, `.tabela-atendimentos`, `.atendimento-pessoa-linha`, `fieldset/legend`, `.row2`, `.radio-row`, `.field label/input/select/textarea`, `.hint`, `.drawer-actions`. **Zero diferença não justificada.**
- [ ] **Step 4:** `tools/refb/textos.py` para os módulos 10 e 08 (e 06 nas partes de tarefa): zerar diferenças não justificadas; listar as justificadas (recibo, "senha de administrador", dica do PDF).
- [ ] **Step 5:** `docs/arquivo/plano-fusao/paridade/RELATORIO-3.md`, `docs/arquivo/plano-fusao/NOTAS-PLANO-3.md`, `ACEITE-VISUAL.md`; ficha do projeto no acervo.
- [ ] **Step 6: commit.**

## Task 6: Verificação final

- [ ] `gofmt -l .` vazio; `go vet ./...`; `go test -race -p 1 ./...` (contêiner); `scripts/check-lines.sh`; `sha256sum -c docs/plano-fusao/HASHES-B.txt` (B intacto).
- [ ] Reconstruir só o app dev (sem `down -v`): a migração 0009 recria `tarefas`; **antes**, conferir que o banco dev não tem tarefas/prazos de negócio (se tiver, PARAR e reportar). `GET /login` 200; CSP intacta; sem credenciais reais; remover contêineres de teste.

## Self-review

- Cobertura do B: 10-atendimentos (T2), 08-tarefas e 06 (T3–T4), 05 (estados/prioridade/subtítulos), 02/15/07 (Plano 1B).
- Itens do B **não** reproduzidos por regra do v2, todos nas Decisões: "senha de administrador", conversão do PDF em `.md`, fluxo de recibo, texto "definitivamente".
- Regra de ouro preservada: prazo fatal da tarefa → domínio Prazos, `A CONFERIR` no mesmo campo, confirmação só por advogado.
