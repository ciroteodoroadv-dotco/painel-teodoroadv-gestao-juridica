# Fusão — Plano 6: Monitor DataJud e tarefa de monitoramento

> ## EMENDA DO ADVOGADO (04/11/2026) — **prevalece sobre qualquer trecho abaixo que a contradiga**
>
> **E1 — Quem é monitorado:** **todos** os processos cadastrados e não excluídos que tenham **número CNJ válido**, em **qualquer estado** (ativo, sobrestado, extinto e arquivado). (Isso substitui "só ativo/sobrestado" na tabela de Decisões e o teste (j) da Task 4: processo extinto/arquivado **é** consultado.) A **tarefa de monitoramento do B** (criar/baixar em cadeia, Task 5) continua ligada ao ciclo de vida, **apenas para ativo/sobrestado**, como no B.
>
> **E2 — Descrição completa:** cada movimentação/intimação nova traz a **descrição completa**, com a origem **DataJud** ou **DJEN** (ou "DJEN (sigiloso)"). Sem corte de 2000 caracteres (substitui a limitação do Plano 5): guardar o **texto integral** em coluna própria (`texto`), com teto de segurança de 100.000 caracteres (corte por caracteres, marcando "[texto cortado]" se ocorrer). **DataJud:** guardar e exibir **todos** os campos que a API devolver do movimento (grau, órgão julgador, código e nome do movimento, complementos tabelados com nome/valor/descrição, data e hora). **Verdade sobre a fonte:** o DataJud **não fornece o teor da intimação**; a tela deve dizer, no próprio detalhe, "Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação." **DJEN:** texto integral da comunicação. A tabela da categoria Intimações mantém as colunas do B e ganha o botão `btn-text` **"Ver"** que abre o drawer **"Intimação"** (campos: Origem, Tribunal, Órgão, Processo, Data, Tipo, Texto completo, Link). Vale também para as intimações do Plano 5 (migrar o que já existir sem perda).
>
> **E3 — Tarefas de análise e prazo interno de 2 dias:** para **cada nova intimação/movimentação** é criada **uma tarefa de análise** em "A confirmar" **e**, por processo, **uma tarefa consolidada por dia de coleta com novidades** (as duas existem; a consolidada lista as novidades do dia, a individual cobre cada intimação). **Ambas** nascem com **Prazo fatal = data da criação + 2 dias corridos** (data civil de São Paulo), gravado pelo mesmo caminho do prazo fatal da tarefa (`PrazoSync`, Plano 3), e **o selo "A CONFERIR" aparece no mesmo campo**. Esse prazo é **interno, de confirmação** pelo advogado "para evitar perda de prazo": a descrição da tarefa diz, literalmente, "Prazo interno de confirmação (2 dias corridos). NÃO é o prazo processual: o advogado apura o prazo da intimação." O sistema continua **sem calcular nem presumir o prazo processual**. O prazo interno é **encerrado** quando a tarefa sai de "A confirmar" (só advogado move) **ou** quando a intimação é decidida (Confirmar/Cancelar); se vencer, aparece como atrasado nos níveis já existentes do domínio Prazos (faixa/lista de prazos críticos da Agenda). Isso substitui "sem prazo fatal" (Global Constraints, Decisões e o teste (m)) e **também passa a valer para as tarefas de análise do Plano 5** (DJEN): ajustar o Plano 5 já entregue, os testes, `COLETA-DJEN.md` e o glossário. `created_by` do prazo/tarefa automáticos: mesma regra do Plano 5.
>
> **Interpretação a ser registrada em `docs/arquivo/plano-fusao/NOTAS-PLANO-6.md` (o advogado pode corrigir):** "uma por processo e todos os dias" foi entendido como *uma tarefa consolidada por processo em cada dia em que a coleta trouxer novidade*, e não uma tarefa fixa por processo todos os dias sem novidade (o que geraria milhares de tarefas vazias).

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever, leia por inteiro, no B (somente leitura):** `referencia/painel-b/js/16-sync-status.js`, `referencia/painel-b/js/11-agenda.js` (Intimações), `referencia/painel-b/js/09-processos.js` (drawer "Mover estado": texto sobre a tarefa de monitoramento), `referencia/painel-b/js/03-api-mapa.js` (`mapearIntimacaoAPI`); **servidor do B, só contraste**: `servidor/monitorprocessual/{monitor.go,coletor.go,numero.go}`, `servidor/ficha/monitoramento.go`, `servidor/servicos/intimacoes.go`, `servidor/servicos/sincronizacao.go`, `servidor/sincronizador/*.go`; e `docs/plano-fusao/painel-B-inventario.md` §3 (bugs do B: só o 1º `hit`, todo movimento vira tarefa, falha duplica tarefa, "visto" sem registro, data em UTC). Leia também os planos 2, 4C e 5 e os `NOTAS-PLANO-2/4C/5.md`. **Documentação oficial do DataJud:** `https://datajud-wiki.cnj.jus.br/api-publica/` e `/endpoints/` (se o acesso for negado — o swagger do DJEN deu 403 — **não contornar**: anotar em `docs/arquivo/plano-fusao/NOTAS-PLANO-6.md` e deixar o item como pendência de conferência no teste manual). O B vale na forma; o v2 vale na regra.

**Goal:** Acompanhar, processo a processo, as **movimentações no DataJud (CNJ)** dos processos cadastrados que estão **em andamento** (estados `ativo` e `sobrestado`), com a **tarefa de monitoramento** criada e baixada pelo ciclo de vida do processo (como no B), e a camada de **processo em segredo de justiça pelo DJEN** (consulta pela OAB **filtrada pelo número do processo**, nunca a pauta inteira). Cada movimentação nova aparece na categoria **Intimações** da Agenda (origem "DataJud") e gera **uma tarefa de análise consolidada em "A confirmar"**. O rodapé "Estado das coletas" mostra DataJud e "DJEN por processo sigiloso".

**Architecture:** Novo domínio `internal/monitor` (tabelas `monitor_processos`, `processo_movimentos`, `monitor_sigilosos`), cliente `internal/monitor/datajud` isolado (URL base fixa, chave só por segredo, limites, `httptest` nos testes), tabela de tribunais **verificada contra a fonte oficial**, e reaproveitamento de `internal/platform/coletas`, de `internal/intimacoes` (linhas e decisão), de `internal/tarefas` (tarefa de análise e tarefa de monitoramento) e do cliente DJEN do Plano 5 (com filtro por número). Integração **desligada por padrão** (`PAINEL_DATAJUD_ATIVO=false`); nenhum teste faz requisição de rede.

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, JS dos planos anteriores.

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `ROADMAP-PRAZOS-E-INTEGRACOES.md` (Fases C e D), planos 1B–5 e 4C.

## Global Constraints

- Tudo dos planos anteriores (CSP estrita; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...` em contêiner; B somente leitura e hashes conferidos; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; teste **vermelho real** antes de implementar; **nada se apaga** — cancelar com motivo; "A CONFERIR" onde houver prazo).
- **Só processos cadastrados no painel** são consultados (decisão do advogado: são de clientes do escritório); a consulta usa o **número CNJ** do cadastro. Sem número válido → o processo **não é consultado** e o estado dele diz "sem número CNJ válido".
- **Chave do DataJud**: segredo em arquivo/variável (`PAINEL_DATAJUD_KEY` via secret), **nunca** em log, URL de tela, banco ou repositório; ausente → coleta `bloqueado` com mensagem "Chave do DataJud não configurada."
- **Coleta educada:** 1 requisição por vez, intervalo mínimo configurável (padrão 1 s) entre processos, timeout, resposta limitada em tamanho, `429` → espera e repete, sem seguir redirecionamento a outro host; `User-Agent` genérico; **sem e-mail do usuário** em cabeçalho/URL.
- **"Sem resultado" ≠ "sem prazo" e ≠ "sem movimentação"**: consulta sem nenhum `hit` vira estado **parcial** com a mensagem "Sem registro no DataJud para este número (pode haver segredo de justiça, atraso na publicação ou número incorreto). Confira nas fontes oficiais." — nunca "sem novidades".
- **O DataJud não traz o teor das intimações**: o painel registra *movimentações* (data, código e nome do movimento); não presume prazo, não calcula prazo, não cria prazo fatal.
- **LGPD:** movimentações podem conter nome de parte → texto limitado (corte por caracteres) e só em tabela protegida; logs só com contagens e códigos de erro.
- **Heartbeat:** coleta ligada e último sucesso há mais de `PAINEL_DATAJUD_ALERTA_HORAS` (padrão 36, pois o DataJud não é tempo real e a frequência de atualização não é garantida) → faixa `.notice` na Agenda.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Quem é monitorado | Processos **não excluídos** com estado `ativo` ou `sobrestado` **e** número CNJ válido (20 dígitos com dígito verificador). Estado `extinto`/`arquivado` → sai do monitoramento. |
| Tarefa de monitoramento (herdada do B) | Criada **ao cadastrar** processo ativo/sobrestado e **ao mover** para ativo/sobrestado; **baixada em cadeia** (estado "Concluída com sucesso" + nota) ao mover para extinto/arquivado; idempotente (nunca duas abertas); se só existir baixada, cria **nova** (a baixada é histórico). Título "Monitoramento processual — {número}" (ou o título da ficha sem número); subtítulo "Monitoramento"; prioridade Normal; estado Pendente; **sem prazo**; descrição com os **textos do B** ("Tarefa criada automaticamente no cadastro do processo…", "ATENÇÃO: ficha sem número CNJ…"). O drawer "Mover estado" já traz o texto do B (Plano 2): agora o **efeito passa a valer**. |
| Várias instâncias | O DataJud devolve um `hit` por grau/instância; o v2 **lê todos** (corrige o `Hits[0]` do B) e guarda o grau em cada movimento; chave do movimento = `grau|data|código|nome|complemento-resumido`. |
| Quantidade de tarefas | **Uma tarefa de análise consolidada por processo e por dia de coleta**, em "A confirmar", sem prazo; lista os movimentos novos (até 20 + "e mais N") e remete ao processo. Cada movimento novo é **uma linha** na categoria Intimações (origem "DataJud", status `triagem`), como no B; Confirmar/Cancelar valem por linha. Isso corta o ruído do B (uma tarefa por movimento) sem esconder movimento. |
| Primeira leitura de um processo | Registra os movimentos existentes como **base** (`base=true`, não aparecem como novos nem geram tarefa) e só alerta pelo que surgir depois. A tela da coleta mostra "linha de base registrada em {data}". |
| Falha | Movimento só é marcado como visto **na mesma transação** que cria a linha de Intimações e a tarefa consolidada; falha em qualquer passo desfaz o item e **não** avança o estado do processo. |
| Segredo de justiça | Cadastro na página **Coleta DJEN** (v2; o B usa um `segredos.json`): lista "Processos em segredo de justiça monitorados" com **número do processo** (precisa existir no cadastro) + **OAB** (de `oab_monitoradas`). A consulta usa o cliente DJEN do Plano 5 com **filtro por número**; resultado vira linha em Intimações com origem "DJEN (sigiloso)". O rodapé mostra "DJEN por processo sigiloso". Não há camada por credencial própria nem provedor comercial (Fase D do roteiro). |
| Tabela de tribunais | Reescrita **a partir da fonte oficial**: TJs (J=8, 27 UFs, `07 → tjdft`), TRFs (J=4, TR 01–06; verificar 6 e eventuais novos), TRTs (J=5, TR 01–24; TR 00 → TST), TREs (J=6, TR 01–27 → `tre-<uf>`; TR 00 → TSE), TJMs (J=9: MG, RS, SP) e STM (J=7), STJ (J=3). Conferir cada alias no endpoint oficial; **o teste compara com uma lista independente escrita à mão da Res. CNJ 65 (Anexo VI), não com a própria tabela**. Tribunal sem endpoint → processo `bloqueado` com mensagem explicativa. |
| Página "Coleta DJEN" | Ganha duas seções (mesmo vocabulário visual do B): **Monitor de processos (DataJud)** — estado da integração (ligada/desligada), botão "Executar monitor agora" (só ligada, só administrador), tabela `Processo | Tribunal | Último resultado | Última consulta | Base` e a mensagem por processo — e **Processos em segredo de justiça**. Admin; auditado. |
| Rodapé | `rotulos = {intimacoes:'DJEN', monitor:'DataJud', 'monitor-segredos':'DJEN por processo sigiloso'}` e os textos de estado do B (Plano 5). |

---

## Task 1: Migração 0013

**Files:** Create `migrations/0013_monitor_datajud.sql`; Test `internal/app/migracoes_0013_test.go`.

**Esquema:** `monitor_processos(processo_id uuid pk references processos, tribunal_alias text, status text check in ('sucesso','parcial','falha','bloqueado','nao_sincronizado') default 'nao_sincronizado', mensagem text not null default '', ultima_consulta timestamptz, ultimo_sucesso timestamptz, base_registrada_em timestamptz)`; `processo_movimentos(id uuid pk, processo_id, fonte text check in ('DataJud','DJEN'), grau text not null default '', chave text not null, data date, codigo text not null default '', nome text not null, complemento text not null default '' (≤ 500 caracteres), base boolean not null default false, intimacao_id uuid null references intimacoes, created_at, unique (processo_id, fonte, chave))`; `monitor_sigilosos(id, processo_id unique references processos, oab_id references oab_monitoradas, ativo boolean, created_*/updated_*/version)`; `intimacoes.origem` aceita `'DJEN'`, `'DataJud'`, `'DJEN (sigiloso)'` (CHECK ou domínio) e `intimacoes.oab_id` passa a ser **opcional**; `tarefas` ganha `monitoramento boolean not null default false` com índice único parcial `(processo_id) where monitoramento and estado not in ('concluidas-com-sucesso','concluidas-sem-sucesso','canceladas')` (garante uma aberta por processo). Privilégios do `painel_app`; classificação LGPD no cabeçalho.

- [ ] **Step 1: testes que falham:** duas tarefas de monitoramento abertas para o mesmo processo → violação; uma aberta + uma concluída → ok; movimento duplicado `(processo, fonte, chave)` → violação; `base` e `intimacao_id`; `monitor_sigilosos` exige processo existente.
- [ ] **Step 2–4:** ver falhar, escrever, passar. **Step 5:** `restore-check`. **Step 6: commit.**

## Task 2: Número CNJ e tabela de tribunais verificada

**Files:** Create `internal/monitor/cnj/{numero,tribunais}.go` e testes; Test `internal/monitor/cnj/tribunais_oficial_test.go`.

**Interfaces — Produces:**
```go
func Digitos(s string) string
func Valido(numero string) bool                 // 20 dígitos + dígito verificador (módulo 97, ISO 7064) — mesma regra do Plano 2
func Formatar(numero string) string             // NNNNNNN-DD.AAAA.J.TR.OOOO
func AliasDataJud(numero string) (alias string, err error)
```
- [ ] **Step 1: testes que falham:** (a) validação com números de teste de dígito calculado **à mão no teste** (não gerados pela função); (b) `AliasDataJud` para **todos** os 27 TJs, 6 TRFs, 24 TRTs + TST, 27 TREs + TSE, 3 TJMs, STM e STJ contra uma **lista escrita à mão no teste a partir da Res. CNJ 65/Anexo VI e do wiki do DataJud** (anotar a fonte e a data de consulta no comentário do teste); (c) STF, CNJ, J inexistente, TR fora de faixa → erro explicativo; (d) entrada com pontuação ou espaços.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** se a fonte oficial não puder ser lida, o teste usa a lista conhecida e o plano registra **pendência de conferência** em `docs/arquivo/plano-fusao/NOTAS-PLANO-6.md`. **Step 6: commit.**

## Task 3: Cliente DataJud isolado (sem rede nos testes)

**Files:** Create `internal/monitor/datajud/{cliente,resposta,limites}.go` e testes com `httptest.Server`.

**Interfaces — Produces:**
```go
type Config struct{ BaseURL string /* https fixo */; Chave func() string; Timeout, Intervalo time.Duration; MaxBytes int64; EsperaTaxa time.Duration; Dormir func(time.Duration) }
type Movimento struct{ Grau, Codigo, Nome, Complemento string; Data time.Time }
type Processo struct{ Grau, Tribunal string; Movimentos []Movimento }
type Resultado struct{ Processos []Processo /* um por grau */ ; SemRegistro bool }
func (c *Cliente) Consultar(ctx context.Context, alias, numeroCNJ string) (Resultado, error)
func Chave(m Movimento) string // grau|data|código|nome|complemento resumido — sem colisão entre campos
```
Regras: `POST {BaseURL}/api_publica_{alias}/_search` com corpo `match numeroProcesso` (confirmar o formato no wiki; se não puder, o formato do coletor do B, marcado para conferência), cabeçalho `Authorization: APIKey <chave>`; **todos os `hits`**; `429` → `Dormir(EsperaTaxa)` e repete; `5xx`/timeout/JSON inválido → erro **sem** parciais; chave **nunca** aparece em erro, log ou URL; BaseURL `https` (exceto opção explícita de teste); redirecionamento a outro host recusado; resposta limitada.
- [ ] **Step 1: testes que falham:** vários `hits` (G1 e G2) → ambos lidos (regressão do `Hits[0]`); zero `hits` → `SemRegistro`; `429` com espera fake; erro 500; JSON inválido; resposta gigante; chave ausente → erro "não configurada" **sem** chamar a rede; chave não vaza em mensagem de erro nem no `String()` da config; redirecionamento recusado; movimentos com acentos de vários bytes cortados por caracteres; `Chave` estável.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 4: Serviço do monitor — movimentos, base, linhas de Intimações e tarefa consolidada

**Files:** Create `internal/monitor/{model,repo,service,coleta,sigilosos,agendador}.go` e testes; Modify `internal/intimacoes` (linhas com origem DataJud / DJEN (sigiloso), `oab_id` opcional), `internal/tarefas` (tarefa consolidada por processo/dia, `a-confirmar`, idempotente), `internal/platform/config` (`PAINEL_DATAJUD_ATIVO`, `PAINEL_DATAJUD_INTERVALO`, `PAINEL_DATAJUD_ALERTA_HORAS`), `internal/app/adaptadores*.go`.

**Interfaces — Produces:**
```go
type ProcessosEmAndamento interface{ Listar(ctx context.Context) ([]ProcessoRef, error) } // ativo+sobrestado, não excluídos (implementado por processos)
type AnaliseConsolidada interface{ CriarDoDia(ctx context.Context, tx pgx.Tx, in ConsolidadaIn) (tarefaID string, err error) } // idempotente por (processo, dia)
func (s *Service) ColetarAgora(ctx context.Context, ac webx.Actor) (Resumo, error) // só administrador; auditado
func (s *Service) ColetarProcesso(ctx context.Context, processoID string) error
```
Regras: lista processos em andamento; **sem número válido** → `bloqueado` com mensagem e segue; alias por `AliasDataJud`; `Consultar`; **primeira leitura** → grava tudo como `base` e `base_registrada_em`; leituras seguintes → só movimentos com chave ainda inexistente viram **uma transação** (inserir `processo_movimentos` + linha em `intimacoes` origem "DataJud" + tarefa consolidada do dia), falha desfaz o item e **não** avança `ultimo_sucesso`; `SemRegistro` → status `parcial` com a mensagem fixa; status por processo e agregado em `coletas` (módulos `monitor` e `monitor-segredos`); coleta **desligada** → `bloqueado` "Coleta desligada neste ambiente." e **zero rede** (teste com `RoundTripper` que falha se chamado); trinco `pg_try_advisory_lock`; agendador só com a chave ligada; segredo de justiça: para cada `monitor_sigilosos` ativo → cliente DJEN com filtro por número → linhas com origem "DJEN (sigiloso)" pela mesma transação.
- [ ] **Step 1: testes que falham (banco + servidor falso):** (a) primeira leitura só grava base, sem tarefa nem linhas; (b) segunda leitura com 2 movimentos novos → 2 linhas de Intimações + **1** tarefa consolidada em "A confirmar" sem prazo; (c) terceira leitura igual → nada novo; (d) dois graus com movimentos → ambos tratados; (e) falha forçada ao criar a tarefa → nada gravado e `ultimo_sucesso` não avança; reexecução registra; (f) zero `hits` → `parcial` com a mensagem fixa e **nenhuma** frase "sem novidades"; (g) processo sem número válido → `bloqueado`, sem rede; (h) chave desligada → `RoundTripper` falso não chamado; (i) duas coletas simultâneas → só uma; (j) processo `extinto`/`arquivado`/excluído **não** é consultado; (k) sigiloso: consulta com filtro por número (a URL falsa recebe `numeroProcesso`), nunca sem filtro; (l) administrador-only e auditoria; (m) nenhuma tarefa criada tem prazo fatal nem aparece em `prazos`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** adaptadores; `fronteiras_test.go` verde. **Step 6: commit.**

## Task 5: Tarefa de monitoramento no ciclo de vida do processo

**Files:** Modify `internal/processos/{service,acoes}.go` e `internal/tarefas/monitoramento.go` (novo); Test `internal/tarefas/monitoramento_test.go`, `internal/app/monitoramento_ciclo_test.go`.

**Interface — Produces:** `type Monitoramento interface{ Garantir(ctx, tx pgx.Tx, ac webx.Actor, p ProcessoRef) (criada bool, err error); Baixar(ctx, tx pgx.Tx, ac webx.Actor, processoID, motivo string) (encontrada bool, err error) }` — chamadas **dentro da mesma transação** de criar/mover estado do processo.
- [ ] **Step 1: testes que falham:** criar processo `ativo` → 1 tarefa "Monitoramento processual — {número}" em Pendente, subtítulo "Monitoramento", sem prazo, com a descrição do B (e o bloco "ATENÇÃO: ficha sem número CNJ…" quando vazio); criar `extinto` → nenhuma; mover `ativo→sobrestado` → continua a mesma (idempotente); `ativo→extinto` → tarefa vai a "Concluída com sucesso" com a nota de encerramento e **continua listada**; `extinto→ativo` → **nova** tarefa (a antiga permanece); duas abertas impossíveis (índice); falha na tarefa desfaz a mudança de estado; auditoria registra `monitoramento.criado/baixado`; o motivo do "Mover estado" segue indo ao andamento (Plano 2).
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 6: Telas — Intimações com origem DataJud, rodapé e página de coletas

**Files:** Modify `internal/agenda/templates/tabelas.html` (origem exibida), `internal/intimacoes/templates/*` (seções novas da página **Coleta DJEN**), `internal/auth/shell.go`/`base.html` (rodapé com os três módulos), `web/static/compat.css` (se a medição exigir); Test `internal/intimacoes/handler_test.go`, `internal/app/monitor_tela_test.go`.

- [ ] **Step 1: testes que falham:** categoria Intimações mostra linhas com Origem "DataJud" e "DJEN (sigiloso)" no mesmo formato do B (colunas `Origem | Processo | Data | Status | Tarefa de análise | ações`); Confirmar/Cancelar iguais aos do Plano 5/4C; rodapé lista `DJEN`, `DataJud` e `DJEN por processo sigiloso` com os textos de estado do B e `sync-error` fora de sucesso; página de coletas: seção do monitor mostra a tabela por processo, a mensagem de "sem registro" e "linha de base registrada"; botão "Executar monitor agora" só se ligada e só administrador; cadastro de sigiloso exige processo existente e OAB ativa com base da autorização; HTML sem `on*=`/`style=`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 7: Documentação, seed, paridade e verificação final

- [ ] **Step 1:** `docs/operacao/MONITOR-DATAJUD.md` (leigo): o que o monitor faz e **não** faz (movimentações, não o teor da intimação; não é tempo real; segredo de justiça pode não aparecer; "sem registro" ≠ "sem movimentação"; "sem resultado" ≠ "sem prazo"), como configurar a chave (arquivo de segredo), como ligar **só depois** do teste manual orientado, o que significam "linha de base" e "tarefa consolidada". Termos novos em `docs/arquivo/plano-fusao/GLOSSARIO-PLANO-6.md`.
- [ ] **Step 2:** seed fictício (2 processos ativos com número de teste válido, 1 sobrestado, 1 extinto; movimentos de exemplo incluindo um segundo grau; 1 sigiloso) e o **mesmo** conjunto em `tools/refb/` (`/api/intimacoes` com origem DataJud; `/api/sincronizacao/status` com os três módulos).
- [ ] **Step 3:** paridade B × v2 no Chrome da categoria Intimações com origens novas e do rodapé com 3 linhas (zero diferença não justificada); `tools/refb/textos6.py` (módulos `16-sync-status` e o trecho de Intimações). **Executar no Chrome** o que for JS novo; dizer o que não foi verificado.
- [ ] **Step 4:** `gofmt -l .`; `go vet ./...`; `go test -race -p 1 ./...` (contêiner, **rodada final**); `scripts/check-lines.sh`; `sha256sum -c HASHES-B.txt`. Antes de reconstruir o app dev: `psql` na socket do contêiner do banco dev (sem ler credenciais) para confirmar que `processos`/`tarefas` não têm dado de negócio (se tiverem, a migração não pode apagar nada); `GET /login` 200; CSP intacta; `PAINEL_DATAJUD_ATIVO` ausente/`false`; sem credenciais reais; remover contêineres de teste.
- [ ] **Step 5:** `docs/arquivo/plano-fusao/paridade/RELATORIO-6.md`, `docs/arquivo/plano-fusao/NOTAS-PLANO-6.md`, `ACEITE-VISUAL.md`; ficha do projeto no acervo; commit.
- [ ] **Step 6 (fora do código, com o advogado):** **teste manual orientado**: configurar a chave pública do DataJud em arquivo de segredo; ligar o monitor; acompanhar a primeira rodada com **um** processo de cliente; conferir a linha de base e o formato real da resposta contra a documentação oficial **antes** de ampliar.

## Self-review

- Cobertura do B: tarefa de monitoramento e baixa em cadeia (T5), monitor DataJud processo a processo (T3–T4), camada de segredo por DJEN com filtro por número (T4), rodapé com os três módulos (T6). Bugs do B tratados: só o 1º `hit`, tarefa por movimento, "visto" sem registro e falha duplicando tarefa, tabela de tribunais sem independência de teste, TRE/TJM/STM ignorados.
- Regras do v2: integração desligada por padrão; só processos cadastrados; chave só por segredo; sem prazo presumido; "sem resultado" ≠ "sem prazo"; nada se apaga; tudo em transação e auditado.
- Fora do plano (explícito): camada de segredo por credencial própria/provedor comercial (Fase D), cálculo de prazo (Plano 7), WhatsApp.
