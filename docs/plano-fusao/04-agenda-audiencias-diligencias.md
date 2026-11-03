# Fusão — Plano 4: Agenda, Audiências e Diligências idênticas ao painel B

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever qualquer template, leia por inteiro, linha por linha, no B (somente leitura):** `web/js/11-agenda.js` (715 linhas — o módulo central), `web/js/05-dominio.js`, `web/js/07-app-core.js`, `web/js/06-helpers-vinculos.js`, `web/js/08-tarefas.js` (tabela de tarefas dentro da Agenda), `web/js/03-api-mapa.js` (`mapearAgendaAPI`, `mapearAudienciaAPI`, `mapearDiligenciaAPI`, `mapearIntimacaoAPI`), `web/js/15-drawers.js`, `web/js/02-util.js`, `web/css/painel.css`. **Leia também as regras do servidor do B**, só leitura: `servidor/agenda/agenda.go`, `servidor/agenda/http.go`, `servidor/audiencias/http.go`, `servidor/diligencias/http.go`, `servidor/ficha/audiencia.go`, `servidor/ficha/diligencia.go` (conflito de horário, "crítico", status da diligência, cancelamento da audiência, mensagens de erro e de aviso). **O B vale na forma; o v2 vale na regra.** Rótulos, ordem dos campos, dicas, placeholders, textos de subtítulo/toast/aviso, opções de select, valores padrão e ordem das colunas são **copiados literalmente**. Método e armadilhas: `02-…md`, `03-…md`, `NOTAS-PLANO-2.md`, `NOTAS-PLANO-3.md`.

**Goal:** A **Agenda** (página inicial), a aba **Audiências** e as **Diligências** do v2 ficam idênticas às do B: pílulas das **7 categorias** (Tarefas, Intimações, Audiências, Diligências, Reuniões, Ligações, **Aulas**) com contador e uma aberta por vez; busca; botão **Novo compromisso** (Reunião, Ligação, Diligência, **Aula**); aba **Audiências** com acordeão pelos 6 tipos de ato; drawers de criar/editar; cancelamento de audiência com senha; conflito de horário pelas regras do B.

**Architecture:** Migração `0010` **substitui** `agenda_eventos`/`agenda_participantes` (o B não tem participantes nem página de detalhe) por três tabelas que espelham as três fichas do B: `agenda_compromissos`, `audiencias`, `diligencias`. Três domínios Go (`agenda`, `audiencias`, `diligencias`); a **página** Agenda é do domínio `agenda` e lê os outros por **interfaces** (`AudienciaLookup`, `DiligenciaLookup`, `TarefaLookup`, `IntimacaoLookup`), ligadas em `internal/app`. A categoria "Intimações" mostra, até o Plano 5, o bloco do próprio B quando não há dado: **"Intimações indisponíveis. Nenhuma sincronização de intimações foi executada."** (`form-block`) — nunca dado inventado.

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, JS dos Planos 1B–3.

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `01b-…`, `02-…`, `03-…`.

## Global Constraints

- Tudo dos Planos 1B–3 (CSP estrita; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...` em contêiner; `codigos.Proximo` para `AUD. 0000001` e `DIL. 0000001`; exclusão/cancelamento auditados; `version`; pasta do B somente leitura; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; teste **vermelho de verdade** antes de cada implementação).
- **Fuso:** `America/Sao_Paulo`. "Em aberto" = data ≥ hoje (data civil em SP), como `todayISO()` do B.
- **Chip da data** no topo da Agenda: `<time class="agenda-date-chip" datetime="AAAA-MM-DD">` com `Intl.DateTimeFormat('pt-BR',{weekday:'short',day:'2-digit',month:'2-digit',year:'numeric'})` — no v2, formatado no servidor com o mesmo resultado (ex.: `ter., 03/11/2026`).
- **Seis tipos de ato (slugs e rótulos do B):** `audiencia-ad-monitoria` "Audiência Ad Monitoria"; `conciliacao-e-mediacao` "Conciliação e Mediação"; `inicial-justica-do-trabalho` "Inicial — Justiça do Trabalho"; `instrucao-e-julgamento` "Instrução e Julgamento"; `sustentacao-oral` "Sustentação Oral"; `una-justica-do-trabalho` "Una — Justiça do Trabalho".
- Acordeão da **Agenda**: **um painel por vez**, nenhum aberto de início; **Audiências**: vários abertos. Busca da Agenda filtra **ao digitar** (campo `#buscaAgenda`, placeholder "Buscar por título, tipo, cliente ou responsável…"); a de Audiências também ("Buscar por título, processo, cliente ou juízo…").
- **Nada de "Prazo Fatal" como categoria ou tipo** (decisão do advogado de 29/10, registrada no B).

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Participantes e página de detalhe de evento (existiam no v2, **não existem no B**) | Removidos junto com `agenda_eventos` (tabelas sem dados). Registrar em `NOTAS-PLANO-4.md`. |
| Conflito de horário | **Regras lidas no servidor do B** (`agenda.go`, `http.go` dos três módulos) e reproduzidas: mesmo dia e **mesma hora exata**; "crítico" = audiência; audiência nunca é bloqueada (apenas gera `aviso` quando coincide com outro crítico); compromisso/diligência são bloqueados quando coincidem com item crítico (confirmar a regra exata no código do B e copiá-la). Resposta de bloqueio = erro com a mensagem do B ("Bloqueado: conflito de horário com um compromisso existente.") exibida como toast de erro. |
| "Processo vinculado" em audiência/diligência | Campo de texto com máscara CNJ, **idêntico ao B**. Regra do v2: se o número existir em `processos`, grava `processo_id` (vínculo real) além do texto; senão guarda só o texto. A tabela mostra o número como no B. |
| Compromisso: Cliente / Atendimento vinculado / Processo vinculado (selects com busca "Sem vínculo") | Idênticos; os selects usam as listas dos domínios Pessoas, Atendimentos e Processos (interfaces). |
| Responsável da **diligência** | **Texto livre**, como no B ("Responsável pela diligência"/"Responsável"). Não é usuário nem pessoa. |
| Status da diligência | Valor e opções **como no `ficha/diligencia.go` do B**; "Status não se edita por aqui." |
| Cancelar audiência | Mesmo drawer ("Cancelar audiência", "A ficha é preservada com o motivo do cancelamento. Confirme com a sua senha.", campos "Motivo do cancelamento *" e "Senha *", botões "Voltar"/"Cancelar audiência"); o servidor **reautentica** a senha do usuário logado (`auth.Reautenticar`), exige `PermAudienciaCancel`, grava motivo, quem e quando, **audita** e mantém a ficha (selo `badge-neutral` "Cancelada" no lugar das ações). Toast: "Audiência cancelada; a ficha foi preservada." |
| "Excluir" de compromisso | O B só mostra `toastNaoImplementado()` ("Ainda não implementado nesta fase do painel.") — **reproduzir o mesmo botão e o mesmo toast**; nenhuma exclusão. |
| Tarefas dentro da Agenda | Via `TarefaLookup` (Plano 3): categoria **Tarefas** = todas as tarefas em aberto (estados fora de concluída/cancelada); **Reuniões/Ligações/Aulas** = compromissos **+** tarefas em aberto cujo subtítulo, normalizado (sem acento, minúsculas), esteja em `ligacao|ligacoes`, `reuniao|reunioes`, `aula|aulas`. Colunas e selos exatamente como `tabelaTarefasCategoriaAgenda`/`computeTarefaStatus` do B; o **prazo fatal** exibido leva o selo **A CONFERIR** quando o prazo ligado estiver `a_conferir` (regra do v2, no mesmo campo). |
| Faixa de prazos | A faixa `.notice` e a lista de prazos críticos do Plano 1B permanecem no topo da Agenda (regra do v2). |
| Home | `GET /` redireciona (302) para `/agenda`; a Agenda é a página inicial, como no B. |

---

## Task 1: Migração 0010

**Files:** Create `migrations/0010_agenda_audiencias_diligencias_do_b.sql`; Test `internal/app/migracoes_0010_test.go`.

**Esquema** (nomes literais; todas com `id uuid pk`, `excluido_em`, `created_*`/`updated_*`, `version`; classificação LGPD no cabeçalho — título/descrição/local = PESSOAL):
- `agenda_compromissos`: `tipo text check in ('Reunião','Ligação','Aula')`, `descricao text not null`, `data date not null`, `hora time not null`, `local text not null default ''`, `cliente_pessoa_id uuid null`, `atendimento_id uuid null`, `processo_id uuid null`, `criado_por_pessoa_id uuid null`; índice `(data, hora)`.
- `audiencias`: `codigo text unique not null`, `tipo_ato text check in (os 6 slugs)`, `titulo text not null`, `processo_numero text not null default ''`, `processo_id uuid null`, `cliente_pessoa_id uuid null`, `data date not null`, `hora time not null`, `juizo text not null default ''`, `local text not null default ''`, `descricao text not null default ''`, `cancelada boolean not null default false`, `cancelamento_motivo text not null default ''`, `cancelada_em timestamptz null`, `cancelada_por uuid null references users`; check: cancelada ⇒ motivo não vazio.
- `diligencias`: `codigo text unique not null`, `titulo text not null`, `processo_numero text not null default ''`, `processo_id uuid null`, `cliente_pessoa_id uuid null`, `local text not null default ''`, `data_limite date not null`, `hora time null`, `responsavel text not null default ''`, `status text not null` com o conjunto e o padrão do B.
- `DROP TABLE agenda_participantes, agenda_eventos` **só se vazias** (`DO $$ … RAISE EXCEPTION`).
- Permissões RBAC novas (em código, não em SQL): `PermAgendaRead/Write`, `PermAudienciaRead/Write/Cancel`, `PermDiligenciaRead/Write`.

- [ ] **Step 1: testes que falham** — inserções válidas e inválidas (tipo fora da lista; `tipo_ato` fora dos 6; cancelada sem motivo; `data_limite` nula); `painel_app` insere/atualiza e não apaga `auditoria`; as tabelas antigas deixam de existir.
- [ ] **Step 2–4:** ver falhar, escrever, passar; **Step 5:** `restore-check`; **Step 6: commit.**

## Task 2: Compromissos (Reunião, Ligação, Aula) + conflito de horário

**Files:** Modify/Create em `internal/agenda/` (`model.go`, `repo.go`, `service.go`, `conflito.go`, `handler.go`, `formulario.go`, `templates/{drawer,editar}.html`); remover o código de eventos/participantes e o domínio de detalhe; Test `internal/agenda/*_test.go`, `internal/app/agenda_test.go` (reescrever).

**Interfaces — Produces:**
```go
var TiposCompromisso = []string{"Reunião", "Ligação", "Diligência", "Aula"} // ordem do B (OPCOES.agendaTipo); "Diligência" cria uma diligência (Task 4)
type Compromisso struct{ ID, Tipo, Descricao, Local string; Data time.Time; Hora string; ClienteID, AtendimentoID, ProcessoID, CriadoPorID string; Version int }
type ErrConflito struct{ Com string }   // mensagem do B
func (s *Service) CriarCompromisso(ctx, ac webx.Actor, in CompromissoInput) (aviso string, err error)
func (s *Service) AtualizarCompromisso(ctx, ac webx.Actor, id string, version int, in CompromissoInput) error
```
**Drawers (literais do B, `openAgendaForm`/`openAgendaEditForm`):** "Novo compromisso" — "Audiência nunca é bloqueada por conflito de horário; os demais compromissos podem ser. Prazo fatal é campo da tarefa (aba Tarefas), não compromisso."; **Tipo \*** (select "Selecione…" + 4 tipos), **Descrição \***, linha `row2` **Data \*** / **Hora \***, **Local ou link da aula/reunião** (+ hint "Somente registro do local/link. Não inicia gravação, download ou transcrição."), bloco **Responsável pela diligência** (aparece só quando o tipo é Diligência, por `data-mostra-quando="Diligência"`; quando Diligência, os campos "Atendimento vinculado" e "Criado por" ficam ocultos), `fieldset` de vínculos **Cliente (opcional)** / **Atendimento vinculado (opcional)** / **Processo vinculado (opcional)** (cada um com campo de busca "Buscar nome ou código PES…", "Buscar código ATE, cliente ou objeto…", "Buscar código PRO, número ou título…" e select "Sem vínculo"; filtro por `data-filtra-opcoes`), **Criado por**; "Cancelar"/"Salvar compromisso". "Editar compromisso": subtítulo = descrição; sem bloco de diligência; "Salvar alterações". Toasts: "Compromisso criado." / "Compromisso criado — {aviso}" / "Compromisso atualizado." / "Descrição, data e hora são obrigatórios." / "Selecione o tipo do compromisso." / "Falha ao criar o compromisso.".

- [ ] **Step 1: testes que falham:** criar Reunião/Ligação/Aula; tipo inválido recusado; campos obrigatórios; conflito conforme a regra do B (cobrir os casos lidos no `agenda.go`); `Diligência` no formulário **não** cria compromisso (cria diligência — Task 4); editar com versão antiga → conflito; vínculos inexistentes recusados; auditoria `compromisso.criado/atualizado`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** handlers (fragmento por `HX-Request`; `POST` por `data-drawer-form`; `forms.js`: `data-filtra-opcoes`, `data-mostra-quando` já existem — reutilizar). **Step 6: commit.**

## Task 3: Audiências

**Files:** Create `internal/audiencias/{model,repo,service,handler,formulario,cancelar}.go`, `templates/{pagina,drawer,cancelar}.html`; Modify `internal/app/app.go`; Test `internal/audiencias/*_test.go`, `internal/app/audiencias_test.go`.

**Tela e drawers (literais do B):** aba **Audiências** / sub "Audiências marcadas, agrupadas por tipo de ato."; botão "Nova audiência"; busca; acordeão pelos 6 tipos, ordenado por data+hora, vários abertos; tabela `Tipo do ato | Código | Título | Data | Hora | Local | Processo | (ações)` (o selo `badge-danger` "Audiência" com o rótulo do tipo em `hint`); vazio "Nenhuma audiência em aberto nesta categoria."; ações **Editar** e **Cancelar audiência** (ou selo "Cancelada"). Drawer "Nova audiência" — "Audiência é item crítico: nunca é bloqueada por conflito de horário, só avisa se coincidir com outro item crítico." — **Tipo do ato \***, **Título \*** (placeholder "Ex.: Audiência de instrução — fulano x banco"), **Processo vinculado** (máscara CNJ), **Cliente**, `row2` **Data \*** / **Hora \***, **Juízo**, **Local ou link da aula/reunião** (+hint), **Observações**; "Cancelar"/"Salvar audiência". Edição: "Tipo do ato" desabilitado + hint "Tipo do ato não se edita aqui (é a pasta da ficha)." (no v2 o texto entre parênteses é mantido literalmente) ; "Salvar alterações". Toasts: "Audiência criada — código AUD. 0000001." (ou "Audiência criada — {aviso}"), "Audiência atualizada.", "Selecione o tipo do ato.", "Título, data e hora são obrigatórios.", "Audiência cancelada; a ficha foi preservada.".

- [ ] **Step 1: testes que falham:** criar com código sequencial; 6 tipos aceitos e outro recusado; aviso quando coincide com outra audiência (sem bloquear); processo resolvido por número existente; editar não troca o tipo; cancelar exige motivo e senha correta (senha errada → erro e nada muda); cancelada some das ações e permanece listada com selo; sem `PermAudienciaCancel` → 403; auditoria; busca sem acento.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** handlers/templates; contador do menu "Audiências". **Step 6: commit.**

## Task 4: Diligências

**Files:** Create `internal/diligencias/{model,repo,service,handler,formulario}.go`, `templates/editar.html`; Modify `internal/agenda` (criação a partir do "Novo compromisso"); Test `internal/diligencias/*_test.go`.

**Regras/telas (literais do B):** criada pelo drawer "Novo compromisso" com tipo Diligência (campos: Descrição→Título, Data→Data-limite, Hora, Local, Responsável pela diligência, Processo vinculado, Cliente); toast "Diligência criada — código DIL. 0000001."; erros "Falha ao criar a diligência."; edição — drawer "Editar diligência" (subtítulo = título; **Título \***, **Processo vinculado** (máscara CNJ), **Cliente**, `row2` **Data-limite \*** / **Hora**, **Local**, **Responsável**, hint "Status não se edita por aqui.", "Cancelar"/"Salvar alterações"); toast "Diligência atualizada."; tabela na Agenda `Código | Título | Data-limite | Hora | Local | Responsável | Status | Processo | (Editar)`; vazio "Nenhuma diligência em aberto.".

- [ ] **Step 1: testes que falham:** criar/editar; código `DIL.`; conflito conforme o B; status padrão; "em aberto" = data-limite ≥ hoje; auditoria.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 5: Página Agenda (7 categorias) e home

**Files:** Create `internal/agenda/{pagina,categorias,tabelas}.go`, `templates/{pagina,tabelas}.html`; Modify `internal/auth/shell.go`/`app.go` (home → `/agenda`; contador "Agenda" = compromissos + audiências + diligências, como `renderNav` do B); Modify `web/static/forms.js` (busca ao digitar da Agenda por `data-filtro`, acordeão `agenda` um por vez — já existem; apenas usar); Test `internal/agenda/pagina_test.go`, `internal/app/agenda_pagina_test.go`.

**Interfaces — Consumes:**
```go
type AudienciaLookup interface{ EmAberto(ctx context.Context, hoje time.Time, q string) ([]AudienciaLinha, error) }
type DiligenciaLookup interface{ EmAberto(ctx context.Context, hoje time.Time, q string) ([]DiligenciaLinha, error) }
type TarefaLookup interface{
	EmAberto(ctx context.Context, q string) ([]TarefaLinha, error)                       // categoria Tarefas
	PorSubtitulo(ctx context.Context, normalizados []string, q string) ([]TarefaLinha, error) // Ligações/Reuniões/Aulas
}
type IntimacaoLookup interface{ Lista(ctx context.Context) (linhas []IntimacaoLinha, disponivel bool, mensagem string, err error) } // Plano 5 implementa; até lá: disponivel=false
```
**Produces:** `GET /agenda` com as 7 pílulas **nesta ordem**: Tarefas, Intimações, Audiências, Diligências, Reuniões, Ligações, Aulas, cada uma `<button class="acc-head" data-acc="agenda:CHAVE">` com `<span>rótulo</span><span class="count">N</span>` dentro de `<div class="acc-heads">` em `#stats`; painéis `.acc-item` com as tabelas e vazios do B ("Nenhum item em aberto nesta categoria."; "Nenhuma tarefa em aberto."; "Nenhuma intimação registrada no histórico."); topo: `h1 Agenda`, chip da data, sub "Painel principal: compromissos e tarefas em aberto, por categoria.", botão "Novo compromisso"; busca `#buscaAgenda` filtrando todas as tabelas ao digitar (sem acento).

- [ ] **Step 1: testes que falham:** ordem e rótulos exatos das 7 pílulas (**incluindo "Aulas"**); contagens (2 aulas futuras + 1 passada + 1 tarefa "aula" aberta + 1 concluída → Aulas = 3); Ligações/Reuniões idem; Tarefas = só abertas; Intimações mostra o bloco "Intimações indisponíveis." quando indisponível; chip da data em `pt-BR`; `/` → 302 `/agenda`; sem `PermAgendaRead` → negado; HTML sem `on*=`/`style=`; a faixa de prazos do v2 presente; prazo fatal `a_conferir` aparece com o selo **A CONFERIR** na tabela de tarefas.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** adaptadores em `internal/app/adaptadores.go` ligando `audiencias`, `diligencias`, `tarefas` ao `agenda`; `fronteiras_test.go` verde. **Step 6: commit.**

## Task 6: Seed, paridade B × v2 e textos

- [ ] **Step 1:** seed fictício (3 compromissos: Reunião, Ligação, Aula; 2 audiências de tipos diferentes, 1 cancelada; 2 diligências) e a **mesma massa** em `tools/refb/` (fixtures do B nos formatos de `mapearAgendaAPI`/`mapearAudienciaAPI`/`mapearDiligenciaAPI`).
- [ ] **Step 2:** instantâneos: Agenda (sem painel aberto; com Aulas aberta; com Tarefas aberta; com Audiências aberta), Audiências (2 tipos abertos), drawers "Novo compromisso", "Editar compromisso", "Nova audiência", "Editar audiência", "Cancelar audiência", "Editar diligência".
- [ ] **Step 3:** `tools/refb/paridade.js` no B (porta 18787) e no v2: `.stats .acc-heads`, `.acc-head(.open)`, `.agenda-date-chip`, `#buscaAgenda`, `.agenda-table-scroll`, `.agenda-tarefas`, `.badge-*`, `.row-actions`, `.empty`, `.form-block`, `.field`, `.row2`, `.hint`, `.drawer-actions`. **Zero diferença não justificada.**
- [ ] **Step 4:** `tools/refb/textos4.py` (módulo 11 e 05): zerar diferenças não justificadas; listar as justificadas.
- [ ] **Step 5:** `RELATORIO-4.md`, `NOTAS-PLANO-4.md`, `ACEITE-VISUAL.md`; ficha do projeto no acervo. **Step 6: commit.**

## Task 7: Verificação final

- [ ] `gofmt -l .` vazio; `go vet ./...`; `go test -race -p 1 ./...` (contêiner, rodada **final** após qualquer mudança de CSS/JS); `scripts/check-lines.sh`; `sha256sum -c docs/plano-fusao/HASHES-B.txt`.
- [ ] Antes de reconstruir o app dev: confirmar (via `psql` na socket do contêiner do banco dev) que `agenda_eventos`/`agenda_participantes` estão **vazias**; senão PARAR e reportar. Reconstruir só o app (sem `down -v`); `GET /login` 200; CSP intacta; sem credenciais reais; remover contêineres de teste.

## Self-review

- Cobertura do B: `11-agenda.js` inteiro (T2–T5), `05` (tipos/6 atos), `03` (mapeamentos), `06`/`08` (tabela de tarefas), `07`/`15` (casca).
- Não reproduzido por regra do v2 (todos nas Decisões): senha de administrador → reautenticação; vínculo real do processo por número; participantes/detalhe removidos por não existirem no B.
- "Aula" presente em: pílulas (T5), tipos do "Novo compromisso" (T2), tarefas por subtítulo (T5).
