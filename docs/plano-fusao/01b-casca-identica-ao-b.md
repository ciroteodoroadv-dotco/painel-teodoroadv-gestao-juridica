# Fusão — Plano 1B: casca idêntica ao painel B

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`). **Antes de escrever qualquer template ou script, leia por inteiro, no B (somente leitura):** `web/index.html`, `web/css/painel.css`, `web/js/02-util.js`, `05-dominio.js`, `07-app-core.js`, `13-indices.js`, `15-drawers.js`, `16-sync-status.js`, `99-init.js`. Onde este plano e o B divergirem, **vale o B na forma**.

**Goal:** Toda tela logada do v2 tem a estrutura, os ids, as classes e os comportamentos do `index.html`/`painel.css`/`07-app-core.js`/`15-drawers.js` do B. Prova: paridade de estilo computado contra o próprio B rodando com dados fictícios.

**Architecture:** `base.html` passa a reproduzir o esqueleto do B (`#app > .sidebar + main.main > .topbar + .stats + #content + #syncStatus`, `#idxTicker`, `#drawerOverlay > #drawer`, `#toast`). Cada página fornece um **Head** (título, subtítulo, chip, botão primário) e, opcionalmente, o bloco **stats** (pílulas). `app.js` reproduz o drawer (com pilha), acordeão, máscaras, linhas repetíveis e toast **sem inline** (`data-*` + delegação). As páginas existentes migram para o novo esqueleto sem mudar regra de negócio.

**Tech Stack:** Go 1.26, html/template, htmx local (só onde já usado), JS puro, Python 3 só para o servidor de referência (`tools/refb/`, fora do binário).

**Spec:** `docs/plano-fusao/ESPECIFICACAO-FUSAO.md`, `docs/plano-fusao/00-SEQUENCIA.md`, `docs/plano-fusao/01-casca-visual.md` (Plano 1, já executado: `painel.css` e `logo.png` copiados do B, `compat.css`).

## Global Constraints

- CSP estrita; sem `on*=`, `style=`, `innerHTML`, `eval` (testes do Plano 1 continuam valendo).
- O B é **somente leitura**: o servidor de referência apenas **serve** os arquivos dele e responde `/api/*` com **dados fictícios** gerados por nós. **Não executar `painel.exe`**, **não abrir `dados/`**, não extrair dado real.
- Ids/classes do B preservados: `app`, `navTabs`, `topbar`, `stats`, `content`, `syncStatus`, `idxTicker`, `drawerOverlay`, `drawer`, `toast`, `.sidebar`, `.brand`, `.seal-logo`, `nav.tabs`, `.count`, `.topbar`, `.title-row`, `.sub`, `.agenda-date-chip`, `.acc-heads`, `.acc-head`, `.acc`, `.acc-item`, `.drawer-sub`, `.drawer-actions`, `.field`, `.row2`, `.row3`, `.radio-row`, `.hint`, `.empty`, `.toolbar`, `.search`, `.badge-*`, `.row-actions`, `.btn`, `.btn-primary`, `.btn-ghost`, `.btn-small`, `.btn-text`, `.btn-danger-text`.
- Título da aba do navegador nas telas logadas: `Teodoro Advocacia Estratégica — Painel Processual` (idêntico ao B). Texto do `alt` da logo: `Teodoro Advocacia Estratégica`.
- O drawer **não fecha por ESC nem por clique fora**; fecha só por botões do próprio drawer; foco devolvido ao elemento de origem; `Cancelar` tem `data-drawer-fechar`.
- Arquivos de produção ≤ 300 linhas (dividir `app.js` em `app.js`, `drawer.js`, `forms.js`, `mascaras.js` se preciso; todos `defer`).
- `gofmt`, `go vet`, `go test -race ./...` verdes. Sem push. Commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.

---

## Task 1: Referência visual do B com dados fictícios (para provar a paridade)

**Files:** Create `tools/refb/servidor.py`, `tools/refb/fixtures.py`, `tools/refb/LEIAME.md`; Create `docs/plano-fusao/paridade/` (saídas).

**Interfaces — Produces:** `python3 tools/refb/servidor.py --porta 18787` serve `GET /` e estáticos de `REFB_WEB` (padrão `/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/painel-teodoro/web`, **leitura**), e responde `/api/*` com fixtures fictícias nos formatos que `web/js/03-api-mapa.js` e `04-api-carga.js` esperam (ler esses dois arquivos inteiros para derivar os formatos): `/api/{pessoas,processos,tarefas,atendimentos,agenda,audiencias,diligencias}` (listas), `/api/indices`, `/api/intimacoes`, `/api/sincronizacao/status`, `/api/events` (SSE vazio mantido aberto), `/api/auth/login`. Qualquer POST/PUT/PATCH/DELETE → 200 `{}` (não persiste nada).

- [ ] **Step 1:** Ler `03-api-mapa.js` e `04-api-carga.js` por inteiro; listar os campos usados por cada `mapear*API`.
- [ ] **Step 2:** `fixtures.py` com **nomes e números inventados** (ex.: "Maria Exemplo", CNJ de teste `0000000-00.2026.8.26.0000`): ≥ 4 pessoas (física e jurídica), 3 processos, 3 atendimentos, 6 tarefas (uma por estado relevante, com prazo fatal e subtítulo Ligação/Aula), 5 compromissos (Reunião, Ligação, Aula, Prazo Fatal), 2 audiências, 2 diligências, 6 índices, 2 intimações.
- [ ] **Step 3:** `servidor.py` (só biblioteca padrão: `http.server`, `json`) servindo estáticos com `Cache-Control: no-store`. Teste manual: abrir `http://localhost:18787/` no Chrome → a Agenda do B renderiza com os dados fictícios; `read_console_messages` sem erro.
- [ ] **Step 4:** Capturar (1440×900 e 400×800) e salvar em `docs/plano-fusao/paridade/B-*.png`: Agenda (sem categoria aberta e com "Aulas" aberta), Tarefas (acordeão com 2 estados abertos), Processos, Pessoas, Atendimentos, Audiências, drawer "Nova tarefa" aberto, drawer "Novo compromisso" aberto. (Imagens só com dados fictícios.)
- [ ] **Step 5:** Garantir que a pasta do B não mudou: recalcular `sha256sum` de `css/painel.css`, `js/*.js`, `index.html` e comparar com a lista gravada antes (`docs/plano-fusao/HASHES-B.txt`, ampliar o arquivo com esses arquivos **antes** do Step 3).
- [ ] **Step 6: commit** `test(fusao): servidor de referência do painel B com dados fictícios`.

## Task 2: Esqueleto do B em `base.html` + modelo de cabeçalho por página

**Files:** Modify `internal/auth/templates/base.html`, `internal/auth/pages.go` (dados de página), `internal/auth/router.go` ou onde `AddNav` vive; Modify os templates de página (`internal/*/templates/*.html`, `internal/auth/templates/*.html`) para sair de `<h1>`/`.cabeca` e usar o Head; Test `internal/app/shell_test.go`.

**Interfaces — Produces:**
```go
// Head descreve o cabeçalho da página no padrão do B (.topbar).
type Head struct {
	Title, Sub  string // h1 e <p class="sub">
	Chip        string // vazio, ou data de hoje para a Agenda (<time class="agenda-date-chip">)
	ActionLabel string // "Novo compromisso", "Nova tarefa"… vazio = sem botão
	ActionHref  string // caminho da página de formulário (fallback sem JS)
	ActionDrawer bool  // true: o botão ganha data-drawer="<ActionHref>"
}
// NavItem ganha contador (como o <span class="count"> do B).
type NavItem struct{ Label, Href string; Count func(ctx context.Context) (int, error) }
func (a *Auth) AddNavCount(label, href string, perm Perm, count func(context.Context) (int, error))
```
- Menu principal na **ordem alfabética do B**: Agenda, Atendimentos, Audiências, Pessoas, Processos, Tarefas (cada um registrado pelo seu domínio quando existir). Itens que o v2 tem e o B não (Prazos, Documentos, Minha conta, Usuários, Auditoria, Sair) vão para um **segundo `<nav class="tabs tabs-sistema">`** no pé da sidebar (mesmas classes dos itens do B).
- `GET /` renderiza a Agenda (home = Agenda). O widget de prazos do v2 vira a faixa `<div class="notice">` do B no topo de `#content` da Agenda: "**N prazo(s) a conferir** · M vencem em até 5 dias — ver Prazos" (só se N+M > 0; texto honesto, sem cálculo novo).
- Bloco opcional `{{block "stats" .}}{{end}}` dentro de `<div class="stats" id="stats">` (pílulas do acordeão).
- Telas sem sessão (login/MFA/setup/recuperação) mantêm o cartão central, usando os tokens do B.

- [ ] **Step 1: teste que falha** (`shell_test.go`, usando o harness): `GET /agenda` logado contém, nesta ordem: `id="app"`, `class="sidebar"`, `id="navTabs"` com os links "Agenda", "Atendimentos"… na ordem alfabética (apenas os já existentes), `class="topbar"`, `id="stats"`, `id="content"`, `id="syncStatus"`, `id="idxTicker"`, `id="drawerOverlay"`, `id="drawer"`, `id="toast"`; `<title>` é o do B; **não** contém `<header class="topo">` nem `cabeca`.
- [ ] **Step 2:** ver falhar. **Step 3:** implementar o esqueleto e migrar **todas** as páginas existentes para o Head (ler cada template antes de editar; manter formulários e textos).
- [ ] **Step 4:** teste de papéis: usuário sem `PermAgendaRead` não vê o item Agenda; contador aparece como `<span class="count">N</span>`.
- [ ] **Step 5:** `go test ./...` verde; `TestTemplatesSemInline` verde.
- [ ] **Step 6: commit** `feat(ui): esqueleto do painel B (topbar, stats, sidebar com contadores) em todas as telas`.

## Task 3: `app.js` do B sem inline — drawer com pilha, acordeão, repetidores, máscaras, toast

**Files:** Modify/Create `web/static/{app.js,drawer.js,forms.js,mascaras.js}`; Modify `base.html` (scripts `defer`); Modify `web/tema_test.go`; Create `web/static/teste-js.html`? **Não** (sem página de teste em produção) — usar o servidor de referência / Chrome para verificar.

**Contrato (para os planos 2–8):**
- **Drawer** (`15-drawers.js`): `[data-drawer="URL"]` faz `GET URL` com `HX-Request: true`; o fragmento (que começa por `<h2>`) vai para `#drawer`; `h2` recebe `id="drawerTitle"`; abre `#drawerOverlay` (remove `hidden`, `aria-hidden=false`, `body.drawer-aberto`); foco no primeiro foco válido. `[data-drawer-sobrepor]` empilha o conteúdo atual (pilha como `App.drawerStack`); `[data-drawer-fechar]` fecha (desempilha se houver pilha; senão fecha e devolve o foco ao elemento de origem). **Sem ESC e sem clique fora.**
- **Formulário no drawer:** `<form data-drawer-form action="/x" method="post">` envia por `fetch` (FormData + `X-CSRF-Token`); resposta `200` com HTML → substitui `#drawer` (erro de validação mantém o drawer aberto); resposta com `HX-Redirect`/`X-Painel-Ok: <mensagem>` → fecha o drawer, mostra toast e recarrega o `#content` (`fetch(location.href)` e troca `#content`, `#stats` e contadores do menu). Botão de envio vira "Salvando…" e desabilitado durante o envio (como `App.saveTarefa`).
- **Acordeão** (`renderAcordeao`/`toggleAcc`): `<button class="acc-head" data-acc="GRUPO:CHAVE">` alterna `.open` no botão e no `.acc-item[data-acc="GRUPO:CHAVE"]`; grupo `agenda` abre **um por vez**, os demais abrem **vários**.
- **Linhas repetíveis:** `<button type="button" class="btn-text" data-add-row="#idTemplate" data-row-target="#partesRows">+ adicionar parte</button>` clona o `<template>`; `<button data-remove-row=".row-parte">Remover parte</button>` remove o ancestral. (Substitui `adicionarLinhaParteTarefa` e `onclick="this.closest(...).remove()"`.)
- **Máscaras** (`02-util.js`): `<input data-mask="cpf|cnpj|cep|phone|cnj|currency">` aplica `maskCPF/maskCNPJ/maskCEP/maskPhone/maskCNJ/maskCurrencyInput` (portar as funções **literalmente** do B) no evento `input`; `isValidCPF/isValidCNPJ` portadas e usadas por `data-validate="cpf|cnpj"` (mensagem em `setCustomValidity`).
- **"Outro" revela campo:** `<select data-mostra-quando="Outro" data-mostra="#idCampo">` (substitui `onchange="…style.display=…"`; usar `hidden`, não `style`).
- **Toast:** `PainelToast(msg, erro)` idêntico a `showToast` (3,2 s; classe `error`).
- **Busca:** `<input data-filtro="#idTabela">` filtra linhas `tr` por texto sem acento/maiúscula (para Agenda, Pessoas…); `<form data-busca-servidor>` envia por GET (Tarefas busca ao enviar, como no B).
- **Aviso ao sair de formulário sujo e CSRF do htmx:** manter o que já existe.

- [ ] **Step 1: testes que falham** em `web/tema_test.go`: os quatro arquivos existem; nenhum contém `innerHTML`, `eval(`, `new Function`, `document.write`, `.onclick =`, `setAttribute("on`; `mascaras.js` contém as funções `maskCPF`, `maskCNPJ`, `maskCEP`, `maskPhone`, `maskCNJ`, `maskCurrencyInput`, `isValidCPF`, `isValidCNPJ`; cada arquivo ≤ 300 linhas; `drawer.js` **não** contém `Escape` nem `keydown`.
- [ ] **Step 2:** ver falhar; **Step 3:** implementar portando a lógica do B (cada função de máscara/validação **idêntica**); **Step 4:** passar.
- [ ] **Step 5: verificação no navegador** com a página de referência do B e o v2 lado a lado, via `javascript_tool` **somente leitura** (sem alterar dados): aplicar máscara de CPF digitando `12345678909` → `123.456.789-09` nos dois; drawer: abrir, pressionar ESC e clicar fora → continua aberto; `Cancelar` fecha e devolve o foco.
- [ ] **Step 6: commit** `feat(ui): comportamentos do painel B (drawer com pilha, acordeão, máscaras, linhas repetíveis, toast) sem inline`.

## Task 4: Rodapé do B — `#syncStatus` e `#idxTicker` honestos

**Files:** Modify `base.html`, `internal/auth` (dados de página), `web/static/app.js`; Test `shell_test.go`.

- [ ] **Step 1:** `<section id="syncStatus" class="sync-status" aria-label="Estado das coletas">` com o texto inicial do B: **"Estado das coletas — Consultando histórico local. Isso não ativa atualizações automáticas."** Enquanto não houver coletor (planos 5/6), o servidor devolve no próprio HTML: "**Estado das coletas** — Nenhuma coleta configurada neste painel. Não presuma que houve atualização; confira as fontes oficiais." (mesmo `<strong>` + `<p>` do B).
- [ ] **Step 2:** `<div id="idxTicker" class="idx-ticker" role="status" aria-label="Índices econômicos">` com o texto de estado do B quando não há dados: "Índices econômicos: não sincronizado" (classe `idxt-status`). **Nunca** valor presumido.
- [ ] **Step 3:** teste: ambos presentes em toda página logada; o texto de "não sincronizado" aparece.
- [ ] **Step 4: commit.**

## Task 5: Paridade de estilo computado (B × v2)

**Files:** Create `tools/refb/paridade.js` (executado via `javascript_tool` no Chrome, só leitura), `docs/plano-fusao/paridade/RELATORIO-1B.md`; Create `internal/app/instantaneos_test.go` (gera HTML do v2 com seed fictício em `$PARIDADE_OUT`, ignorado se a variável não existir).

- [ ] **Step 1:** o teste de instantâneos grava `agenda.html`, `tarefas.html`, `processos.html`, `pessoas.html` (páginas logadas do harness com **dados do seed fictício**) e copia `web/static/*`; um `python3 -m http.server` em `$PARIDADE_OUT` serve para o Chrome (sem credenciais, sem dados reais).
- [ ] **Step 2:** `paridade.js` percorre os seletores `.sidebar, .brand .seal-logo, nav.tabs a (ou button), nav.tabs .count, .topbar, .topbar h1, .topbar .sub, .btn-primary, .stats, .acc-head, .acc-head.open, .acc-head .count, table, th, td, .badge, .empty, .search, .sync-status, .idx-ticker, #drawer, .toast` e devolve JSON de `getComputedStyle` para: `font-family, font-size, font-weight, line-height, color, background-color, border-top-color, border-radius, padding-top/right/bottom/left, margin-top/bottom, width, height, letter-spacing, box-shadow, display, text-transform`.
- [ ] **Step 3:** rodar no B (porta 18787) e no v2 (porta do snapshot), mesma largura 1440×900; comparar; **toda diferença** vira item da tabela do relatório com decisão (corrigir o v2 ou justificar por regra de segurança/dado real vs. fictício). Meta: **zero diferenças não justificadas** nas propriedades acima.
- [ ] **Step 4:** capturas v2 em `docs/plano-fusao/paridade/V2-*.png` (mesmas telas e larguras do Task 1) e revisão visual lado a lado pelo agente que executa; anotar divergências remanescentes.
- [ ] **Step 5:** relatório + atualizar `ACEITE-VISUAL.md` (Plano 1B) e a ficha do projeto no acervo.
- [ ] **Step 6: commit** `test(fusao): paridade computada entre o painel B e o v2 (casca)`.

## Task 6: Limpeza e verificação final

- [ ] `gofmt -l .` vazio; `go vet ./...`; `go test -race ./...`; `staticcheck`/`gosec` se instalados; `wc -l` ≤ 300 nos arquivos de produção tocados.
- [ ] Reconstruir só o app (`deploy/`: `PAINEL_DEV_PORT=18080 docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d app`), `GET /login` 200, CSP intacta (`curl -sI`).
- [ ] **Não** usar credenciais reais; **não** apagar nem alterar dados do banco dev; remover contêineres de teste que o próprio trabalho criar.

## Self-review

- Cobertura do B: esqueleto (T2), comportamentos 15/02/07 (T3), 13/16 (T4), paridade (T5). Itens 05/99 aplicados em T2/T3 (ordem das abas, init).
- Consistência: ids/classes iguais aos do B; contrato `data-*` definido em T3 é o que os planos 2–8 usam.
- Sem placeholders: tudo que depende de leitura do B manda **ler o arquivo indicado antes de escrever**.
