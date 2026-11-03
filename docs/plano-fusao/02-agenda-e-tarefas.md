# Fusão — Plano 2: Agenda e Tarefas no visual e na lógica do painel B

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`). Este plano fixa **o que** e **como se prova**; o código dos handlers é escrito na execução, seguindo os padrões já existentes de `internal/agenda` e `internal/tarefas` (`handler→service→repo`, `auth.Router.Handle` com permissão, `webx`, `html/template` via `auth.BaseTemplate()`).

**Goal:** A tela **Agenda** vira o "painel principal" do B: data de hoje, pílulas com contador das **7 categorias** (Tarefas, Intimações, Audiências, Diligências, Reuniões, Ligações, **Aulas**), uma categoria aberta por vez, busca, botão **Novo compromisso** em drawer. **Tarefas** ganha subtítulo, prioridade e o estado "em revisão", em drawer.

**Architecture:** Migração `0007` amplia os tipos de evento e os campos de tarefa. O domínio `agenda` ganha a visão por categorias e recebe as tarefas por uma **interface** (`TarefaLookup`), ligada no ponto de composição (`internal/app`), sem importar `tarefas` (teste de fronteiras). Formulários em drawer reutilizam o contrato do Plano 1 (`data-drawer`, `HX-Request`, toast).

**Tech Stack:** Go 1.26, PostgreSQL 17 (goose), html/template, htmx local.

**Spec:** `docs/plano-fusao/ESPECIFICACAO-FUSAO.md` §5; inventário `docs/plano-fusao/painel-B-inventario.md`. Referência de comportamento do B (somente leitura): `web/js/11-agenda.js:22-45,46-110,274-330`, `web/js/08-tarefas.js`, `web/js/06-helpers-vinculos.js:212-218`.

## Global Constraints

- Tudo do Plano 1 (CSP estrita, sem `on*=`/`style=`, arquivos ≤ 300 linhas, `go vet`/`gofmt`/testes verdes).
- Prazo: a Agenda **nunca calcula nem sugere** data. "Prazo fatal" **não é categoria** (decisão do advogado, 29/10): vive no domínio `prazos` (regra de ouro).
- Fuso: `America/Sao_Paulo` (`webx.SP()`); "em aberto" = início ≥ 00:00 de hoje em SP.
- RBAC: leitura exige `PermAgendaRead`/`PermTarefaRead`; criar/editar exige as de escrita; botões de escrita só aparecem a quem pode.
- Domínios só se falam por interface; `internal/app/fronteiras_test.go` continua verde.
- Pasta do painel B somente leitura.
- Commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`. Sem push.

## Decisões de herança do B (e o que NÃO herdamos)

| Herda | Como no v2 |
|---|---|
| 7 categorias com contador, uma aberta por vez | pílulas `a.acc-head` (`open` na ativa), `?cat=` na URL |
| Botão "Novo compromisso": Reunião, Ligação, Diligência, **Aula** | drawer; Audiência segue no formulário completo até o Plano 4 |
| Tarefa com subtítulo Ligação/Reunião/Aula aparece também na categoria | subtítulo livre (≤ 80), comparado sem acento/maiúscula: `ligacao\|ligacoes`, `reuniao\|reunioes`, `aula\|aulas` |
| Conflito de horário: mesmo instante; **audiência nunca é bloqueada** | serviço `agenda`, escopo do mesmo responsável |
| Estados da tarefa, prioridade (baixa/normal/média/alta) | `status` ganha `em_revisao`; `prioridade` nova; "a-confirmar" do B = `origem=automacao_a_conferir` do v2 |
| Prazo **não** é categoria | mantido |
| **Não herdamos agora** | PDF anexado à tarefa (v2 anexa documento a processo/pessoa; reavaliar no Plano 3); baixa automática de intimação (Plano 5); coluna "prazo fatal" na tarefa (vive em `prazos`) |

---

## Task 1: Migração 0007

**Files:** Create `migrations/0007_agenda_tipos_tarefas_campos.sql`; Test `internal/app/migracoes_0007_test.go` (padrão do `harness_test.go`).

- [ ] **Step 1: teste que falha** — inserir evento `tipo='aula'` e `'ligacao'` (esperado: ok), tarefa com `status='em_revisao'`, `prioridade='alta'`, `subtitulo='Ligação'` (ok), `prioridade='urgente'` (violação), `subtitulo` de 81 caracteres (violação).
- [ ] **Step 2: rodar e ver falhar** — `go test ./internal/app -run Migracoes0007`.
- [ ] **Step 3: escrever a migração**

```sql
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
```
(Conferir o nome real das constraints com `\d agenda_eventos` antes; ajustar se diferirem.)
- [ ] **Step 4: passar** — `go test ./internal/app ./migrations/...`; aplicar também o restore-check (`go run ./cmd/painel restore-check` conforme RUNBOOK) para provar que a migração não quebra restauração.
- [ ] **Step 5: commit** `feat(db): migração 0007 — tipos Ligação/Aula, em revisão, subtítulo e prioridade`.

## Task 2: Tarefas — subtítulo, prioridade, em revisão

**Files:** Modify `internal/tarefas/{model,repo,service,handler}.go`, `internal/tarefas/templates/{form,lista}.html`; Create `internal/tarefas/templates/drawer.html`; Test `internal/tarefas/service_test.go` (ou `internal/app/processos_tarefas_test.go`, onde já há testes do domínio).

**Interfaces — Produces:**
- `Tarefa` ganha `Subtitulo, Prioridade string`; `Prioridades []Opcao` = baixa/normal/média/alta; método `PrioridadeRotulo()`, `PrioridadeClasse()` → `prioridade-alta|media|normal|baixa` (classes já existem em `painel.css`).
- `Input` ganha `Subtitulo, Prioridade`; validação: subtítulo ≤ 80 runes, prioridade ∈ enum (vazio → `normal`), status inclui `em_revisao`.
- `Aberta()` continua true para `aberta`, `em_andamento`, `em_revisao`.
- `(*Service).AbertasPorSubtitulo(ctx, q string, limite int) ([]Tarefa, error)` — tarefas abertas, ordenadas por vencimento (nulos por último), filtro opcional `q` no título; usada pelo adaptador da Agenda.

- [ ] **Step 1: testes que falham** — (a) cria tarefa com subtítulo "Ligação" e prioridade "alta" → lê de volta igual; (b) prioridade inválida → erro de validação; (c) subtítulo de 81 runes → erro; (d) `em_revisao` conta como aberta; (e) `AbertasPorSubtitulo` não devolve concluídas/canceladas/excluídas.
- [ ] **Step 2:** rodar, ver falhar. **Step 3:** implementar (model → repo SQL → service → handler/form). **Step 4:** passar.
- [ ] **Step 5: drawer** — `GET /tarefas/nova` e `GET /tarefas/{id}/editar` com cabeçalho `HX-Request` devolvem só o fragmento `drawer.html` (começa por `<h2 id="drawer-titulo">`, botão `data-drawer-fechar`, form `hx-post` + `hx-target="#drawer-corpo"`); sem o cabeçalho, a página completa de hoje. Sucesso: `HX-Redirect: /tarefas?salvo=1`; erro de validação: reexibe o fragmento com a mensagem (status 200 para o htmx trocar).
- [ ] **Step 6: lista** — coluna "Prioridade" com `<span class="badge prioridade-…">`, subtítulo em `<small>` sob o título; os links "Nova tarefa"/"Editar" ganham `data-drawer="/tarefas/nova"` (mantendo `href` como fallback sem JS).
- [ ] **Step 7: teste de handler** — fragmento não contém `<html`; `POST` com `HX-Request` válido devolve `HX-Redirect`; usuário sem permissão de escrita recebe 403 no POST e não vê "Nova tarefa".
- [ ] **Step 8: commit.**

## Task 3: Agenda — tipos Ligação/Aula e conflito de horário

**Files:** Modify `internal/agenda/{model,service,repo}.go`, `internal/agenda/templates/form.html`; Test `internal/agenda/service_test.go` (ou `internal/app/agenda_test.go`).

**Interfaces — Produces:** `Tipos` = audiência, reunião, **ligação**, diligência, **aula**, outro (ordem do B: Reunião, Ligação, Diligência, Aula no drawer; Audiência e Outro só no formulário completo). `(*Service).Create/Update` devolvem `ErrConflito{Com string}` (título e hora do compromisso conflitante) quando:
- o novo evento **não** é `audiencia`, **e**
- existe evento não excluído com o **mesmo `inicio`** (ao minuto) e o **mesmo responsável** (ou ambos sem responsável).
Audiência **nunca** é bloqueada (o conflito vira só aviso exibido na tela de detalhe). Em `Update`, o próprio evento é ignorado na checagem.

- [ ] **Step 1: testes que falham** — tipos `ligacao` e `aula` aceitos; reunião no mesmo instante e mesmo responsável → `ErrConflito`; mesma hora, outro responsável → ok; audiência no mesmo instante → ok; editar o próprio evento sem mudar horário → ok.
- [ ] **Step 2–4:** ver falhar, implementar (`repo.ExisteNoInstante(ctx, inicio, responsavelID, ignorarID)`), passar.
- [ ] **Step 5: commit.**

## Task 4: Agenda — visão por categorias

**Files:** Create `internal/agenda/categorias.go` (≤ 300 linhas), `internal/agenda/templates/painel.html`, `internal/agenda/categorias_test.go`; Modify `internal/agenda/handler.go` (rota), `internal/app/app.go` (ligação do adaptador); Move: a listagem atual (`lista.html`, filtros tipo/período/paginação) passa a `GET /agenda/todos`.

**Interfaces — Consumes:** `webx.SP()`, `ProcessoLookup`, `UserLookup` existentes.
**Interfaces — Produces:**
```go
// Categoria é uma pílula da Agenda.
type Categoria struct {
	Chave, Rotulo string // "tarefas","intimacoes","audiencias","diligencias","reunioes","ligacoes","aulas"
	Tipo          string // tipo de evento correspondente ("" para tarefas/intimações)
	Subtitulos    []string // formas normalizadas aceitas no subtítulo de tarefa
}
var Categorias = []Categoria{ /* as 7, nessa ordem */ }

// TarefaLookup é o que a Agenda precisa do domínio de tarefas (implementado no app.go por adaptador).
type TarefaLookup interface {
	AbertasPorSubtitulo(ctx context.Context, q string, limite int) ([]TarefaResumo, error)
}
type TarefaResumo struct {
	ID, Titulo, Subtitulo, Prioridade, PrioridadeRotulo, ProcessoNumero, Responsavel, Status string
	Vencimento *time.Time
	AConferir  bool
}
func NormalizarSubtitulo(s string) string // minúsculas, sem acento, sem espaços nas pontas
type Resumo struct{ Categoria; Quantidade int; Aberta bool }
func (s *Service) Painel(ctx context.Context, cat, q string, agora time.Time) (PainelView, error)
```
`PainelView` traz: `Hoje` (extenso pt-BR: "terça-feira, 3 de novembro de 2026" e `dd/mm/aaaa`), `Resumos []Resumo`, `Eventos []Evento` (da categoria aberta, em aberto, ordenados por início), `Tarefas []TarefaResumo` (para tarefas e para ligações/reuniões/aulas), `Q`.

**Regras:**
- Contagem: Tarefas = todas as abertas; Audiências/Diligências/Reuniões/Ligações/Aulas = eventos do tipo com `inicio ≥ hoje 00:00 SP`, **mais**, para Reuniões/Ligações/Aulas, as tarefas abertas cujo subtítulo normalizado esteja em `Subtitulos`; Intimações = 0 com aviso "As intimações do DJEN chegam com o coletor de intimações (Plano 5)." (não inventar dado).
- Nenhuma categoria vem aberta por padrão (como no B); `?cat=` inválido → nenhuma.
- Busca `q` filtra título (eventos e tarefas), sem acento.
- Tela: cabeçalho com `agenda-date` e `agenda-date-chip`, campo de busca, linha de pílulas `acc-heads` com `<a class="acc-head [open]" aria-current>` + `<span class="count">`, painel da categoria (tabela de eventos: Tipo (badge: audiência = `badge-danger`, demais `badge-neutral`), Compromisso, Data, Hora, Local, Processo; tabela de tarefas: Tarefa, Processo, Responsável, Vencimento, Prioridade, Situação com selo "A CONFERIR" quando `AConferir`); estado vazio com `.empty`.
- Botão **Novo compromisso** (só `PermAgendaWrite`) com `data-drawer="/agenda/novo"`; link "Todos os eventos" → `/agenda/todos`.

- [ ] **Step 1: testes que falham** (`categorias_test.go`, sem banco quando possível, com `TarefaLookup` falso):
  1. `NormalizarSubtitulo("  Ligação ") == "ligacao"`; "REUNIÕES" → "reunioes"; "Aula" → "aula".
  2. 7 categorias, ordem e rótulos exatos (inclui **"Aulas"**).
  3. contagem: 2 aulas futuras + 1 passada + 1 tarefa com subtítulo "aula" aberta + 1 concluída → Aulas = 3.
  4. `cat=aulas` devolve só aulas e tarefas "aula"; `cat=xyz` → nenhuma aberta.
  5. Intimações = 0 e traz o aviso do Plano 5.
  6. extenso pt-BR de 2026-11-03 = "terça-feira, 3 de novembro de 2026" (mapas fixos de dia/mês; não depender de locale).
- [ ] **Step 2–4:** ver falhar, implementar, passar.
- [ ] **Step 5: adaptador** em `internal/app/app.go`: tipo que implementa `agenda.TarefaLookup` chamando `tarefas.Service.AbertasPorSubtitulo` e convertendo para `agenda.TarefaResumo`. `fronteiras_test.go` deve continuar verde (agenda não importa tarefas).
- [ ] **Step 6: teste de handler** — `GET /agenda?cat=aulas` com usuário `assistente` → 200, contém "Aulas", não contém `on…=` nem `style=`; `GET /agenda` sem sessão → redireciona ao login; usuário só-leitura não vê "Novo compromisso".
- [ ] **Step 7: commit.**

## Task 5: Drawer "Novo compromisso" e toast de retorno

**Files:** Create `internal/agenda/templates/drawer.html`; Modify `internal/agenda/handler.go`, `web/static/app.js` (flash); Test no handler.

**Contrato:** `GET /agenda/novo` e `GET /agenda/{id}/editar` com `HX-Request` → fragmento com `<h2 id="drawer-titulo">Novo compromisso</h2>`, subtítulo "Audiência nunca é bloqueada por conflito de horário; os demais compromissos podem ser. Prazo fatal é campo do módulo Prazos, não compromisso.", seletor de **Tipo** com Reunião/Ligação/Diligência/**Aula**, título, data, hora, término, local, processo (busca existente), responsável; `hx-post="/agenda"`, `hx-target="#drawer-corpo"`. Sucesso: `HX-Redirect: /agenda?cat=<categoria do tipo>&salvo=1`. `ErrConflito` → fragmento com `.form-error` "Já existe «título» às HH:MM para este responsável." e os dados preservados. Em `app.js`: se `location.search` tem `salvo=1`, chamar `PainelToast("Salvo.")` e remover o parâmetro com `history.replaceState`.

- [ ] **Step 1: testes que falham** — fragmento sem `<html`; `POST` com `HX-Request` cria "aula" e devolve `HX-Redirect` com `cat=aulas&salvo=1`; conflito devolve o fragmento com a mensagem e **não** cria; `tema_test` continua verde (app.js sem `innerHTML`).
- [ ] **Step 2–4:** ver falhar, implementar, passar.
- [ ] **Step 5: commit.**

## Task 6: Seed, auditoria, aceite e ficha

- [ ] **Step 1:** `internal/app/seed.go` (dados fictícios): 1 aula, 1 ligação, 1 reunião futuras e uma tarefa com subtítulo "Ligação"; `seed_test.go` verde.
- [ ] **Step 2:** conferir que criar/editar compromisso e tarefa gera evento de auditoria (testes existentes do `audit` cobrem `agenda.create`/`tarefa.create`; acrescentar asserção para o tipo `aula`).
- [ ] **Step 3: verificação completa** — `gofmt -l .` vazio; `go vet ./...`; `go test -race ./...`; `staticcheck ./...` e `gosec ./...` se instalados (CI já usa); `wc -l` de arquivos de produção tocados ≤ 300.
- [ ] **Step 4:** reconstruir só o app (`docker compose … up --build -d app`) e conferir `GET /login` 200; **não** usar credenciais reais.
- [ ] **Step 5:** `docs/plano-fusao/ACEITE-VISUAL.md`: acrescentar "Plano 2: pendente de conferência do advogado após login (Agenda com 7 pílulas, Aulas, Novo compromisso, drawer, toast)". Atualizar a ficha do projeto no acervo.
- [ ] **Step 6: commit** `docs(fusao): plano 2 executado`.

## Self-review

- **Cobertura:** 7 categorias incl. Aulas (T4), botão Novo compromisso com Aula (T5), tarefas por subtítulo na categoria (T2+T4), conflito/audiência (T3), prioridade/estado (T2), prazo fora das categorias (T4 regras), Intimações honesta (T4).
- **Tipos consistentes:** `TarefaResumo`/`TarefaLookup` definidos em T4 e consumidos em T4 passo 5; `AbertasPorSubtitulo` definido em T2.
- **Sem placeholders:** a única conferência externa é o nome real das constraints (T1 passo 3).
