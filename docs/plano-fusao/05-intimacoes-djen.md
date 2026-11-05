# Fusão — Plano 5: Intimações (DJEN) no visual do B, com a regra do v2

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever, leia por inteiro, no B (somente leitura):** `referencia/painel-b/js/11-agenda.js` (trechos de Intimações: `tarefaDaIntimacao`, `statusIntimacao*`, `tabelaIntimacoes`, `atualizarStatusIntimacao`, e a categoria "Intimações" em `viewAgenda`), `referencia/painel-b/js/03-api-mapa.js` (`mapearIntimacaoAPI`), `referencia/painel-b/js/04-api-carga.js` (`App_carregarIntimacoes`), `referencia/painel-b/js/16-sync-status.js`, `referencia/painel-b/js/13-indices.js` (`statusOperacionalLabel`), `referencia/painel-b/js/07-app-core.js`, `referencia/painel-b/css/painel.css` (`.sync-status`, `.form-block`); **servidor do B, só para contraste** (a lógica é reimplementada com testes, nunca copiada às cegas): `servidor/intimacoes/intimacoes.go`, `servidor/servicos/intimacoes.go`, `servidor/servicos/sincronizacao.go`, `servidor/sincronizador/*.go`, `servidor/rotina.go`, e `docs/INVENTARIO-B` do v2 (`docs/plano-fusao/painel-B-inventario.md` §3: bugs conhecidos do coletor do B). **Leia também a documentação pública do DJEN:** `https://comunicaapi.pje.jus.br/swagger/djen.yml` (parâmetros, paginação, limites). O B vale na forma; o v2 vale na regra.

**Goal:** Receber **intimações do DJEN** pelas OABs dos advogados parceiros e mostrá-las na categoria **Intimações** da Agenda, no visual do B (tabela, selos, ações Confirmar/Descartar), criando para cada intimação nova uma **tarefa de análise em "A confirmar"**. O sistema **detecta e sugere; o advogado confirma** (decisão de 02/11). A integração vem **desligada por padrão** e nada sai para a internet nos testes.

**Architecture:** Novo domínio `internal/intimacoes` (tabelas `oab_monitoradas`, `intimacoes`) e infraestrutura `internal/platform/coletas` (tabela `coletas` com última tentativa, último sucesso, status e mensagem — alimenta o rodapé **"Estado das coletas"** do B e a faixa de alerta do v2). O coletor (`internal/intimacoes/djen`) é um cliente HTTP isolado, com URL base fixa por configuração, tempo limite, limite de tamanho de resposta, espera de 1 min no `429`, paginação completa e **divisão da janela por dia** quando o teto de 10.000 resultados é atingido. Registrar a intimação e criar a tarefa acontece **numa única transação** (corrige o "visto sem registro" do B). A tarefa nasce no estado `a-confirmar`, **sem prazo fatal** (o v2 não presume prazo).

**Tech Stack:** Go 1.26 (`net/http`, `httptest` nos testes), PostgreSQL 17 + goose, html/template, JS dos Planos anteriores.

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `ROADMAP-PRAZOS-E-INTEGRACOES.md` (Fase B), planos 1B–4 e `NOTAS-PLANO-*.md`.

## Global Constraints

- Tudo dos planos anteriores (CSP estrita; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...` em contêiner; pasta do B somente leitura; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; teste **vermelho real** antes de implementar).
- **Integração externa desligada por padrão:** `PAINEL_DJEN_ATIVO=false`. Com a chave desligada **nenhuma requisição de rede é feita** (teste com `http.RoundTripper` que falha o teste se chamado). Nenhum teste fala com `comunicaapi.pje.jus.br`; só `httptest.Server`.
- **Quem é monitorado:** só OABs cadastradas em `oab_monitoradas` com **data de consentimento** preenchida e `ativa=true`. Sem consentimento, a coleta daquela OAB é **recusada** (erro e registro na auditoria).
- **Coleta educada:** no máximo 1 requisição por vez, intervalo mínimo configurável entre requisições (padrão 1 s), `429` → espera 60 s (relógio injetável nos testes), teto de páginas e de resultados respeitado, `User-Agent` genérico (`painel-teodoro`), **sem e-mail do usuário** em cabeçalho/URL.
- **LGPD:** o texto da intimação pode conter dado pessoal → `resumo` é PESSOAL, limitado a 2000 **caracteres** (corte por runes, nunca por bytes); logs só com contagens e códigos de erro, nunca texto de intimação; `link` só aceito se `https://`.
- **"Sem resultado" ≠ "sem prazo"**: a tela e o rodapé nunca dizem "nenhuma intimação nova" como garantia; dizem o estado da última coleta e que a conferência nas fontes oficiais continua necessária.
- **Heartbeat:** se a coleta estiver ativa e o último sucesso tiver mais de `PAINEL_DJEN_ALERTA_HORAS` (padrão 6), a Agenda mostra a faixa `.notice` "**Coleta de intimações sem sucesso desde …** — confira o DJEN manualmente."
- O DJEN **não é tempo real** e não promete segredo de justiça: a documentação do sistema (`docs/operacao/COLETA-DJEN.md`) diz isso com essas palavras.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Status da intimação | `triagem`, `confirmada`, `descartada` (rótulos "Em triagem", "Confirmada", "Descartada"; classes `badge-warn`, `badge-ok`, `badge-neutral`) — como na tela do B. |
| Confirmar / Descartar | Botões e toasts do B ("Intimação confirmada.", "Intimação descartada e mantida no histórico."). Só **advogado** (`PermIntimacaoDecide`); descartar exige o motivo? **Não** (B não pede) — o servidor registra quem/quando na auditoria. Intimação descartada **permanece** no histórico. |
| Tarefa de análise | Criada na mesma transação, estado `a-confirmar`, título "Analisar intimação — {tipo} — {processo}", subtítulo "Análise de decisão" (sugestão do B), prioridade Normal, **sem prazo fatal**, descrição = "Prazo a apurar pelo advogado. O sistema não calculou prazo." + link. Vínculo: o **processo** do v2 se o número existir; senão `origem_tipo='intimacao'` (migração 0011 amplia o CHECK de `tarefas`: `origem_tipo IN ('atendimento','processo','intimacao')` + `intimacao_id`). Na tela de Tarefas o campo **Origem** mostra "DJEN". |
| Prazo sugerido | **Não calculado neste plano.** O campo `rascunho_prazo` do B existe no formato, mas fica vazio até o Plano 7 (calculadora com calendário forense), e mesmo então só como sugestão "A CONFERIR". |
| "Tarefa de análise" na tabela | Idêntica ao B: título em `<strong>` + estado em `hint`; "Tarefa não localizada" se faltar. |
| WhatsApp (a fila do B) | **Não reproduzido** (F13/integração futura; ADR). Anotar em `docs/arquivo/plano-fusao/NOTAS-PLANO-5.md`. |
| Segredo de justiça | Só o que a API pública devolver pela OAB; **sem** camada por credencial própria nem provedor comercial neste plano (o roteiro as trata na Fase D). O rodapé diz "DJEN (público): processos em segredo de justiça podem não aparecer". |
| Cadastro das OABs | Página **"Coleta DJEN"** no grupo "sistema" (só administrador; não existe no B — usa o mesmo vocabulário visual: `topbar`, `.field`, `.row2`, tabela, `.badge`, drawer). Campos: Nome do advogado, Número da OAB, UF, Data do consentimento (obrigatória), Ativa. Mostra o estado da integração ("Coleta desligada neste ambiente" / "Coleta ativa") e o botão **"Executar coleta agora"** (só se ativa). |
| Rodapé "Estado das coletas" | Passa de texto fixo para a lista do B (`16-sync-status.js`): "DJEN: {estado}. Tentativa: {dd/mm/aaaa hh:mm:ss (Brasília)}. Sucesso: {…}. {mensagem}" com `.sync-error` quando não for sucesso; títulos e textos **idênticos** (`estados`: sucesso → "última coleta concluída", parcial → "coleta parcial", falha → "erro", bloqueado → "bloqueado", nao_sincronizado → "sem tentativa registrada"). Renderizado no servidor; atualização por `fetch` do próprio fragmento a cada 60 s e ao voltar à aba (`data-refresh`), como o B. |

---

## Task 1: Migração 0011 e infraestrutura de coletas

**Files:** Create `migrations/0011_intimacoes_e_coletas.sql`, `internal/platform/coletas/{coletas,coletas_test}.go`; Test `internal/app/migracoes_0011_test.go`.

**Esquema:** `oab_monitoradas(id, advogado_nome, numero_oab, uf char(2), consentimento_em date not null, ativa bool, created_*/updated_*/version, unique (numero_oab, uf))`; `intimacoes(id, hash text unique not null, origem text not null default 'DJEN', oab_id uuid references oab_monitoradas, tribunal, orgao, processo_numero text not null default '', processo_id uuid null, tipo text, data_publicacao date, resumo text (≤2000 caracteres), link text, status text check in ('triagem','confirmada','descartada') default 'triagem', decidida_por uuid null, decidida_em timestamptz null, created_at)`; `ALTER TABLE tarefas` amplia `origem_tipo` e adiciona `intimacao_id uuid null references intimacoes` com o CHECK de vínculo exclusivo atualizado; `coletas(modulo text primary key, origem text, ultima_tentativa timestamptz, ultimo_sucesso timestamptz, status text check in ('sucesso','parcial','falha','bloqueado','nao_sincronizado'), mensagem text, janela_desde date)`.
**Interface — Produces:**
```go
package coletas
type Estado struct{ Modulo, Origem, Status, Mensagem string; UltimaTentativa, UltimoSucesso *time.Time; JanelaDesde *time.Time }
func (s *Store) Registrar(ctx context.Context, e Estado) error   // atualiza tentativa; só avança UltimoSucesso/JanelaDesde se Status==sucesso
func (s *Store) Listar(ctx context.Context) ([]Estado, error)
func (s *Store) Obter(ctx context.Context, modulo string) (Estado, bool, error)
```
- [ ] **Step 1: testes que falham:** inserções válidas/inválidas (status; `hash` duplicado; resumo > 2000 caracteres; OAB sem consentimento); tarefa com `origem_tipo='intimacao'` exige `intimacao_id`; `coletas.Registrar` **não** avança `UltimoSucesso` em falha/parcial e avança em sucesso; `painel_app` não apaga auditoria.
- [ ] **Step 2–4:** ver falhar, escrever, passar. **Step 5:** `restore-check`. **Step 6: commit.**

## Task 2: Cliente DJEN isolado (sem rede nos testes)

**Files:** Create `internal/intimacoes/djen/{cliente,janela,normaliza}.go` e testes; Test com `httptest.Server`.

**Interfaces — Produces:**
```go
type Config struct{ BaseURL string /* https fixo */; Timeout, IntervaloMinimo, EsperaTaxa time.Duration; MaxBytesResposta int64; MaxPaginas, MaxResultados int; Agora func() time.Time; Dormir func(time.Duration) }
type Comunicacao struct{ ID, Tribunal, Orgao, Processo, Tipo string; DataPublicacao time.Time; Texto, Link string }
type Resultado struct{ Itens []Comunicacao; Parcial bool; Motivo string }
func (c *Cliente) Coletar(ctx context.Context, oab, uf string, desde, ate time.Time) (Resultado, error)
func JanelaDesde(agora time.Time, ultimoSucesso *time.Time, diasPadrao int) (desde, ate time.Time) // data civil de São Paulo; recua 7 dias sobre o último sucesso; nunca usa UTC
func NormalizarResumo(s string) string // corte por runes, ≤ 2000, UTF-8 válido
func Hash(c Comunicacao) string        // id do DJEN; sem id → SHA-256 de tribunal|órgão|processo|data|tipo|link|resumo
```
**Regras (lidas no swagger e nos bugs do B):** paginação até esgotar; `429` → `Dormir(EsperaTaxa)` e repete **a mesma página** até N vezes; página repetida ou vazia encerra; **teto de 10.000** → refaz a coleta **dia a dia** para a janela e marca `Parcial` só se algum dia ainda estourar; `5xx`/timeout/JSON inválido → erro **sem** devolver parciais como se fossem completos; resposta limitada por `MaxBytesResposta`; `BaseURL` precisa ser `https` (exceto em teste via opção explícita); sem seguir redirecionamento para outro host.
- [ ] **Step 1: testes que falham:** (a) 3 páginas → todos os itens; (b) `429` na página 2 → dorme 60 s (fake) e conclui; (c) página repetida não entra em laço; (d) teto: servidor falso que devolve 10.000 → cliente divide por dia; (e) `500` no meio → erro, **nenhum** resultado parcial aceito como sucesso; (f) resposta gigante → erro; (g) `JanelaDesde` usa data civil de SP (caso 21h de 02/11 em SP = 00h UTC de 03/11) e recua 7 dias; (h) `NormalizarResumo` com acentos de 3 bytes não corta no meio do caractere; (i) `Hash` estável e sem colisão entre campos (separador); (j) redirecionamento a outro host recusado; (k) nenhum cabeçalho/URL contém `@`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 3: Serviço — coleta, dedupe e tarefa de análise na mesma transação

**Files:** Create `internal/intimacoes/{model,repo,service,coleta,agendador}.go` e testes; Modify `internal/tarefas` (criação automática em `a-confirmar`, interface `AutomacaoTarefa`); Modify `internal/app/adaptadores*.go`, `internal/platform/config` (`PAINEL_DJEN_ATIVO`, `PAINEL_DJEN_INTERVALO`, `PAINEL_DJEN_ALERTA_HORAS`).

**Interfaces — Produces:**
```go
type TarefaAutomatica interface {
	CriarAnaliseDeIntimacao(ctx context.Context, tx pgx.Tx, in AnaliseIn) (tarefaID string, err error) // idempotente por intimacao_id
}
type ProcessoPorNumero interface{ IDPorNumero(ctx context.Context, numero string) (string, bool, error) }
func (s *Service) ColetarAgora(ctx context.Context, ac webx.Actor) (Resumo, error) // só administrador; registra auditoria
func (s *Service) Decidir(ctx context.Context, ac webx.Actor, id string, status string) error // confirmada|descartada; só advogado
func (s *Service) Lista(ctx context.Context) ([]Linha, disponivel bool, mensagem string, err error) // implementa agenda.IntimacaoLookup (Plano 4)
```
**Regras:** para cada OAB ativa **com consentimento**: `JanelaDesde` → `Coletar` → para cada comunicação **nova** (hash ainda inexistente) abre **uma transação** que insere a intimação, resolve o processo por número, cria a tarefa `a-confirmar` e só então confirma; falha em qualquer passo desfaz **tudo daquele item** e a janela **não** avança; `coletas.Registrar` com sucesso/parcial/falha por OAB e agregado "DJEN"; coleta desligada (`PAINEL_DJEN_ATIVO=false`) → `bloqueado` com a mensagem "Coleta desligada neste ambiente." e **zero** rede; execução concorrente impedida por `pg_try_advisory_lock`; agendador em goroutine só com a chave ligada, respeitando intervalo e parando no `ctx.Done()`.
- [ ] **Step 1: testes que falham (com banco + servidor falso):** (a) coleta cria intimação `triagem` **e** tarefa `a-confirmar` sem prazo fatal; (b) segunda coleta com os mesmos itens não duplica nada; (c) falha forçada na criação da tarefa → intimação **não** fica gravada, janela não avança, próxima rodada registra; (d) processo existente vira vínculo real; inexistente → `origem_tipo='intimacao'`; (e) OAB sem consentimento → recusada e auditada; (f) chave desligada → `RoundTripper` falso **não** é chamado e o estado é `bloqueado`; (g) duas coletas simultâneas → só uma executa; (h) `Decidir` por assistente → 403; por advogado muda o status e audita; descartada continua em `Lista`; (i) heartbeat: último sucesso há 7 h → `Alerta()` verdadeiro; (j) resumo longo é cortado por caracteres; (k) tarefa criada **não** aparece como prazo no domínio `prazos`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** adaptadores ligando `intimacoes` a `tarefas`, `processos` e `agenda.IntimacaoLookup`; `fronteiras_test.go` verde. **Step 6: commit.**

## Task 4: Telas — categoria Intimações, ações e rodapé "Estado das coletas"

**Files:** Modify `internal/agenda/{categorias,tabelas}.go` e `templates/tabelas.html` (categoria Intimações real); Create `internal/intimacoes/{handler,formulario}.go` e `templates/{tabela,oabs,oab-drawer}.html`; Modify `internal/auth/shell.go` e `base.html` (rodapé dinâmico + faixa de alerta); Modify `web/static/forms.js` (`data-refresh`); Test `internal/intimacoes/handler_test.go`, `internal/app/intimacoes_tela_test.go`.

**Tela (literais do B):** categoria **Intimações** da Agenda: tabela `Origem | Processo | Data | Status | Tarefa de análise | (ações)`; ações **Confirmar** (`btn-text`) e **Descartar** (`btn-danger-text`) só quando `triagem` e só para quem decide; "—" nos demais; contador = **todas** as intimações (histórico); indisponível → `form-block` "**Intimações indisponíveis.** …" com a mensagem real da última coleta; vazio "Nenhuma intimação registrada no histórico."; toasts do B. Rodapé `#syncStatus`: `<strong>Estado das coletas</strong>`, `<p>Histórico local. Não comprova agendamento ativo nem substitui a conferência nas fontes oficiais.</p>`, `<ul>` com uma linha por coleta (`DJEN`), classe `sync-error` quando não for sucesso; `data-refresh` recarrega o fragmento. Página **Coleta DJEN** (administrador): lista de OABs, "Nova OAB" (drawer), "Executar coleta agora", estado.
- [ ] **Step 1: testes que falham:** tabela com as 6 colunas na ordem; Confirmar/Descartar presentes só para advogado e só em `triagem`; clicar (POST) muda status e responde `X-Painel-Ok` com o texto do B; categoria Intimações da Agenda mostra o contador do histórico; indisponível mostra o bloco do B; rodapé lista o estado do `coletas` com os textos do B e `sync-error` em falha; faixa de alerta aparece só com coleta ativa e sem sucesso há mais de N horas; página de OABs só para administrador, exige data de consentimento; HTML sem `on*=`/`style=`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 5: Documentação, seed, paridade e textos

- [ ] **Step 1:** `docs/operacao/COLETA-DJEN.md` (leigo): o que a coleta faz e **não** faz (não é tempo real; segredo de justiça não garantido; "sem resultado" ≠ "sem prazo"); como cadastrar a OAB com consentimento; como ligar (`PAINEL_DJEN_ATIVO=true`) **só depois** do teste manual orientado; o que fazer se aparecer o alerta; limites e coleta educada. Glossário: DJEN, OAB, hash, janela, consentimento (`99-SISTEMA/docs/05-GLOSSARIO.md` no acervo).
- [ ] **Step 2:** seed fictício: 1 OAB (nome inventado, número de teste), 3 intimações (uma por status) com tarefas de análise; `coletas` com um sucesso e uma falha.
- [ ] **Step 3:** instantâneos e **paridade B × v2** para a categoria Intimações e o rodapé (`tools/refb/fixtures.py` do B com `/api/intimacoes` e `/api/sincronizacao/status` fictícios); **zero diferença não justificada**. `tools/refb/textos5.py` (11-agenda: Intimações; 16-sync-status).
- [ ] **Step 4:** `docs/arquivo/plano-fusao/paridade/RELATORIO-5.md`, `docs/arquivo/plano-fusao/NOTAS-PLANO-5.md`, `ACEITE-VISUAL.md`; ficha do projeto no acervo.
- [ ] **Step 5: commit.**

## Task 6: Verificação final

- [ ] `gofmt`, `go vet`, `go test -race -p 1 ./...` (contêiner, rodada **final**), `scripts/check-lines.sh`, `sha256sum -c HASHES-B.txt`.
- [ ] Antes de reconstruir o app dev: conferir que `tarefas`/`prazos` não têm dado de negócio; migração `0011` aplica; `GET /login` 200; CSP intacta; **`PAINEL_DJEN_ATIVO` continua `false`** no ambiente dev; sem credenciais reais; remover contêineres de teste.
- [ ] **Teste manual orientado (fora do código, com o advogado):** lista de OABs parceiras + consentimentos; só então ligar a coleta e acompanhar a primeira rodada.

## Self-review

- Cobertura do B: tela de Intimações (T4), rodapé 16 (T4), status/ações (T3–T4), coletor com paginação/limites (T2), dedupe e tarefa (T3). Bugs do B tratados: janela em UTC, página única, "visto sem registro", tarefa criada com prazo fatal presumido, corte de resumo por bytes, falha duplicando tarefas.
- Regra do v2 preservada: detectar → sugerir → advogado confirma; tarefa em "A confirmar"; sem prazo presumido; integração desligada por padrão; consentimento por OAB; "sem resultado ≠ sem prazo".
- Fora do plano (explícito): camada de segredo de justiça (Fase D), WhatsApp, DataJud (Plano 6), cálculo de prazo (Plano 7).
