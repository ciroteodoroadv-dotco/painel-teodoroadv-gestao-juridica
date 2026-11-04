# Fusão — Plano 8: Índices (rodapé), CEP e fechamento

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever, leia por inteiro, no B (somente leitura):** `web/js/13-indices.js` (ticker do rodapé: ordem, rótulos, textos de estado), `web/js/14-pessoas.js` (somente o trecho de CEP: botão "Buscar CEP", `App.buscarCEP`, textos e o drawer de confirmação "Endereço pelo CEP"), `web/js/04-api-carga.js` (`App_carregarIndices`), `web/js/07-app-core.js` (atualização a cada 30 s), `web/js/15-drawers.js` (`confirmarEmDrawer` com `sobrepor`), `web/css/painel.css` (`.idx-ticker*`); **servidor do B, só contraste:** `servidor/indices/indices.go`, `servidor/cep/cep.go`. Leia também o código atual do v2 (`internal/pessoas` — o botão "Buscar CEP" do Plano 2 hoje responde "Consulta de CEP indisponível neste painel."; `internal/platform/coletas`; `internal/auth` shell/rodapé do Plano 1B/5/6), os `NOTAS-PLANO-*.md` e `docs/ROADMAP-PRAZOS-E-INTEGRACOES.md`. **O B vale na forma; o v2 vale na regra.**

**Goal:** (1) A **faixa de índices** do rodapé passa a mostrar valores **reais e sempre datados** do Banco Central (SGS), com o estado honesto do B quando não houver dado; (2) o botão **"Buscar CEP"** do cadastro de pessoa passa a funcionar (ViaCEP), com a confirmação "Endereço pelo CEP" do B; (3) **fechamento** da fusão: análise estática de segurança, teste de backup/restauração com o esquema final, teste de ponta a ponta no navegador numa pilha isolada, ADRs, documentação e roteiro de aceite para o advogado.

**Architecture:** `internal/indices` (tabela `indices_valores`, cliente BCB isolado, agendador, fragmento HTML do ticker) e `internal/cep` (cliente ViaCEP isolado + rota autenticada que o formulário chama; o navegador **nunca** fala com a internet — a CSP é `default-src 'self'`). As duas integrações vêm **desligadas por padrão** (`PAINEL_INDICES_ATIVO=false`, `PAINEL_CEP_ATIVO=false`) e nenhum teste usa rede. Estado e batimento em `internal/platform/coletas` (módulo `indices`).

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, JS dos planos anteriores; ferramentas de análise (`gosec`, `staticcheck`, `govulncheck`) instaladas **só no contêiner de teste**.

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, planos 1B–7 e `NOTAS-PLANO-*.md`.

## Global Constraints

- Tudo dos planos anteriores (CSP estrita; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...` em contêiner; B somente leitura e hashes conferidos; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; teste **vermelho real** antes de implementar; nada se apaga).
- **Nunca valor presumido** (regra do B, mantida): o rodapé só mostra número que veio da fonte, **com a data de referência e o estado** ("sincronizado", "parcial", "falha", "não sincronizado", "bloqueado"). Valor com mais de `PAINEL_INDICES_VALIDADE_DIAS` (padrão 40, pois IPCA/INPC/IGP-M são mensais) aparece marcado "desatualizado". Índice **não** substitui consulta à fonte para cálculo judicial.
- **Rede só por clientes isolados** (URL base fixa, `https`, timeout, resposta limitada, sem redirecionamento a outro host, `User-Agent` genérico, sem e-mail do usuário). Testes só com `httptest`/`RoundTripper` falso; com a chave desligada, **zero rede** (teste com `RoundTripper` que falha se chamado).
- **CEP é dado pessoal indireto** (endereço de uma pessoa cadastrada): a consulta envia **só os 8 dígitos do CEP** ao ViaCEP, nada mais; a rota exige sessão e permissão de escrita em Pessoas; limite de taxa por usuário; cache curto; **sem** log do CEP; erro mostra o motivo, nunca preenche campo com dado inventado.
- **Fechamento não pode alterar regra de negócio** sem teste vermelho antes; achados de segurança são corrigidos com teste de regressão.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Séries (ordem e rótulos do B, `ORDEM_INDICES`) | "Selic (meta Copom)" (SGS 432, % a.a.), "IPCA" (433, % mensal), "INPC" (188, % mensal), "IGP-M" (189, % mensal), "Dólar (PTAX)" (1, R$), "Salário mínimo" (1619, R$). Endpoint `https://api.bcb.gov.br/dados/serie/bcdata.sgs.{codigo}/dados/ultimos/1?formato=json` (**confirmar na documentação do BCB**; se o acesso for negado, não contornar — pendência em `NOTAS-PLANO-8.md`). O **significado exato** de cada série (variação mensal %, meta etc.) é conferido na fonte e escrito em `docs/operacao/INDICES-E-CEP.md`. |
| Rodapé (forma do B) | `<div id="idxTicker" class="idx-ticker" role="status" aria-label="Índices econômicos">` com `.idxt-item` (rótulo + `<b>valor</b>`) por série e `.idxt-status`: "Ref.: {data da 1ª série} · {estado}"; sem dados: "Índices econômicos: {estado} — {mensagem}". Atualização **a cada 30 s** relendo o **último estado gravado** (nunca dispara coleta), como no B, por `data-refresh` + fragmento `GET /fragmentos/indices`. |
| Formato dos valores | pt-BR com unidade (ex.: "0,52 %", "R$ 5,43", "R$ 1.518,00"), a **data de referência** de cada série visível no `title`. Nenhum arredondamento além do que a fonte entrega. |
| Coleta | 1 requisição por série, intervalo mínimo entre séries, 1 execução por dia no agendador (e botão "Atualizar índices agora" na página do administrador); parcial se alguma série falhar (as que deram certo são gravadas; as que falharam mantêm o **último valor com a data antiga e a marca "desatualizado"**, nunca um valor novo inventado). |
| CEP | `POST /pessoas/cep` (CSRF) devolve JSON `{logradouro,bairro,municipio,uf,fonte}` ou `{erro}`; fonte exibida **como no B**: "Fonte consultada: ViaCEP (público, não oficial)". O drawer da pessoa mantém o botão "Buscar CEP" (`btn btn-ghost btn-small`), os toasts do B ("Informe um CEP com oito dígitos.", "Endereço preenchido nos campos vazios.", "Endereço preenchido e campos existentes substituídos.") e a confirmação **sobreposta** "Endereço pelo CEP — Há campos de endereço preenchidos. Deseja substituí-los pelos dados do CEP?" [Substituir]. CEP inexistente (`erro:true` do ViaCEP) → "CEP não encontrado." |
| Desligado | Com a integração desligada, o botão responde "Consulta de CEP desligada neste ambiente." e o rodapé mostra "Índices econômicos: bloqueado — Coleta desligada neste ambiente." |

---

## Task 1: Índices do BCB no rodapé

**Files:** Create `migrations/0015_indices.sql`, `internal/indices/{model,repo,service,coleta,agendador,fragmento}.go`, `internal/indices/bcb/{cliente,series}.go`, `internal/indices/templates/ticker.html`, testes; Modify `internal/auth/shell.go`/`base.html` (rodapé vem do serviço), `internal/platform/config` (`PAINEL_INDICES_ATIVO`, `_INTERVALO`, `_VALIDADE_DIAS`, `_URL`), `web/static/forms.js` (`data-refresh` a cada 30 s no ticker, se ainda não houver), `internal/app/adaptadores*.go`.

**Esquema:** `indices_valores(serie_codigo int pk, rotulo text not null, valor numeric(20,6) not null, unidade text not null, referencia date not null, obtido_em timestamptz not null, fonte text not null default 'BCB SGS')` (último valor por série; histórico não é necessário aqui). Privilégios do `painel_app`. Módulo `indices` em `coletas` (tentativa/sucesso/estado).
**Interfaces — Produces:**
```go
package bcb
type Serie struct{ Codigo int; Rotulo, Unidade string }
var Series = []Serie{ /* as 6 na ordem do B */ }
type Ponto struct{ Data time.Time; Valor decimal /* sem float */ }
func (c *Cliente) Ultimo(ctx context.Context, codigo int) (Ponto, error)
```
- [ ] **Step 1: testes que falham:** (a) 6 séries lidas por `httptest` com o formato do SGS (`[{"data":"dd/mm/aaaa","valor":"0.52"}]`; **vírgula decimal e ponto** tratados, sem `float`); (b) `HTTP 404/429/500`, JSON inválido, série vazia, resposta gigante, redirecionamento a outro host → erro **sem** gravar; (c) coleta parcial grava só as boas e mantém o antigo das ruins com a marca desatualizado; (d) desligado → `RoundTripper` falso não chamado e estado `bloqueado`; (e) **fragmento do ticker**: ordem dos itens = a do B; `.idxt-status` com o texto do B; sem dados → "Índices econômicos: não sincronizado"; valor com mais de 40 dias → marca "desatualizado"; **nunca** HTML com valor sem referência; (f) formatação pt-BR de cada unidade; (g) `GET /fragmentos/indices` exige sessão e responde só o fragmento; (h) HTML sem `on*=`/`style=`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 2: CEP (ViaCEP) no cadastro de pessoa

**Files:** Create `internal/cep/{cliente,handler}.go` e testes; Modify `internal/pessoas` (rota e formulário), `web/static/forms.js` (ou novo `cep.js` ≤ 300 linhas) — **sem `on*=`**: botão `data-buscar-cep`; Test `internal/cep/*_test.go`, `internal/app/cep_tela_test.go`.
- [ ] **Step 1: testes que falham:** CEP com 8 dígitos → consulta; menos/mais/letras → "Informe um CEP com oito dígitos." **sem** chamar a rede; resposta do ViaCEP com `{"erro":true}` → "CEP não encontrado."; `5xx`/timeout/JSON inválido → mensagem do motivo e **nenhum campo preenchido**; só os 8 dígitos saem na requisição (inspecionar a URL falsa; nenhum cabeçalho com dado do usuário); desligado → "Consulta de CEP desligada neste ambiente." e zero rede; exige sessão e `PermPessoasWrite`; limite de taxa por usuário; cache curto por CEP; nada de CEP em log; fonte "ViaCEP (público, não oficial)".
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: JS** (executar no Chrome): campos vazios são preenchidos; com campo ocupado abre a confirmação **sobreposta** do B e só substitui se confirmar; UF em maiúsculas; botão desabilitado durante a consulta. **Step 6: commit.**

## Task 3: Análise estática de segurança

**Files:** `scripts/analise-estatica.sh` (roda no contêiner `golang:1.26.10-bookworm`: instala `gosec`, `staticcheck`, `govulncheck` com `go install` de versão fixa; executa nas ferramentas e grava o relatório em `docs/plano-fusao/ANALISE-ESTATICA.md`); correções com teste de regressão.
- [ ] **Step 1:** rodar as três; **registrar** a saída resumida (contagem por gravidade).
- [ ] **Step 2:** corrigir **todo** achado de gravidade alta/média do `gosec` e do `govulncheck` (atualizar dependência quando houver correção; justificar por escrito o que for falso positivo com `#nosec` comentado); `staticcheck` zerado ou justificado.
- [ ] **Step 3:** CI (`.github/workflows`) passa a rodar o mesmo script (já usa gosec/staticcheck; acrescentar `govulncheck` se faltar); **não** publicar nada.
- [ ] **Step 4: commit.**

## Task 4: Backup e restauração com o esquema final

- [ ] **Step 1:** rodar `deploy/teste-backup-local.sh` e `restore-check` com o banco **populado por dados fictícios de todos os domínios** (pessoas, processos, atendimentos, tarefas, agenda, audiências, diligências, intimações, movimentos, prazos, calendário, documentos cifrados): backup → restaurar em banco novo → comparar contagens e **hash de um documento cifrado** → chave errada falha → backup adulterado falha.
- [ ] **Step 2:** conferir que `PAINEL_*_ATIVO` e as chaves **não** entram no backup nem no log; `deploy/secrets/` continua ignorado.
- [ ] **Step 3:** registrar o resultado em `docs/plano-fusao/RELATORIO-8.md`; corrigir o que falhar com teste de regressão. **Step 4: commit.**

## Task 5: Teste de ponta a ponta no navegador (pilha isolada, usuários de teste)

**Objetivo:** provar por **clique real** o que até aqui só foi coberto por testes HTTP: envio de formulários, drawers, cancelamentos, duplicidade, prazo "A CONFERIR".
- [ ] **Step 1:** subir uma pilha **isolada** (porta 18081, banco próprio descartável, `PAINEL_ENV=dev`, `SeedDemo`) — **nunca** a pilha do advogado (18080) nem dado real. Criar usuários **de teste** só no seed da pilha isolada (advogado, assistente, administrador) com a forma de 2º fator mais simples **que o próprio sistema permitir em `dev`**; se o 2º fator impedir a automação, usar o TOTP fixo **apenas do seed de teste** (documentar) — **não** enfraquecer o login em produção.
- [ ] **Step 2:** com o Chrome (`mcp__claude-in-chrome__*`), executar e registrar (capturas só com dados fictícios): login; criar pessoa PF e PJ (máscaras, CPF inválido recusado, CEP desligado → mensagem); criar processo e **duplicado** (marcador + justificativa obrigatória); atendimento com "+ Novo cliente" sobreposto; tarefa vinculada com **prazo fatal** (como assistente: selo A CONFERIR; como advogado: "Confirmar prazo"); mover tarefa a "Cancelada" (sem exclusão); Agenda: abrir **Aulas**, criar compromisso, cancelar com motivo; audiência cancelada com senha; Intimações (seed): ver, Confirmar, Cancelar; calculadora de prazo (resultado A CONFERIR + criar prazo preenchido); rodapé de coletas e de índices (seed).
- [ ] **Step 3:** cada falha vira teste de regressão + correção; `docs/plano-fusao/paridade/E2E-8.md` com o que passou, o que falhou e o que não foi possível. **Step 4:** derrubar **somente** a pilha isolada (`docker compose -p <projeto-isolado> down -v` **apenas** nela; **nunca** na do advogado). **Step 5: commit.**

## Task 6: Fechamento documental e roteiro de aceite

- [ ] **Step 1:** ADRs (`docs/adr/`): 0009 "Visual do painel B sobre o v2" (decisão, o que mudou, modo escuro removido, `compat.css`); 0010 "Nada se apaga" (cancelar com motivo, exceções: nenhuma); 0011 "Integrações externas desligadas por padrão" (lista de chaves `PAINEL_*_ATIVO`, como ligar, o que o teste manual exige); 0012 "Prazo: o sistema detecta, sugere e o advogado confirma" (prazo interno de 2 dias, calculadora sempre A CONFERIR).
- [ ] **Step 2:** `docs/operacao/INDICES-E-CEP.md` (leigo) e atualizar `RUNBOOK.md`/`COMECAR-USO-REAL.md` com as novas chaves e telas; `ESPECIFICACAO.md` com um adendo "Fusão com o painel B" apontando para `docs/plano-fusao/`.
- [ ] **Step 3: `docs/plano-fusao/ACEITE-FINAL.md` — roteiro para o advogado, em linguagem leiga, passo a passo,** com caixas de marcar: primeiro login (senha provisória, 2º fator, códigos de recuperação); conferir cada tela contra o painel B (lista de telas e o que olhar); testes manuais orientados (1 processo de cliente no monitor, 1 OAB no DJEN, chave do DataJud, ligar uma integração por vez); **validação de `REGRAS-DE-PRAZO.md`** (Plano 7) e decisão dos itens `[VERIFICAR]`; decisões pendentes (campo "Nome do arquivo da ficha"; preenchimento retroativo da tarefa de monitoramento; mensagens de recarga; Semana Santa/Carnaval no calendário da Justiça Federal); como guardar a chave de backup.
- [ ] **Step 4:** consolidar as pendências de **todos** os `NOTAS-PLANO-*.md` em uma tabela única (`docs/plano-fusao/PENDENCIAS.md`: item, plano, quem decide, risco). Termos novos em `GLOSSARIO-PLANO-8.md`.
- [ ] **Step 5:** `gofmt`, `go vet`, `go test -race -p 1 ./...` (contêiner, **rodada final**), `scripts/check-lines.sh`, `sha256sum -c HASHES-B.txt`; reconstruir só o app dev depois de `psql` (socket local, sem ler credenciais) confirmar que as tabelas afetadas não têm dado de negócio; `GET /login` 200; CSP intacta; todas as `PAINEL_*_ATIVO` ausentes/`false`; remover contêineres de teste; atualizar a ficha do projeto no acervo (estado final) e `ACEITE-VISUAL.md`; commit.

## Ordem se faltar crédito

1º Task 1 e 2 (funcionalidade); 2º Task 4 (backup com o esquema final); 3º Task 3 (análise estática); 4º Task 6 (documentos e roteiro); 5º Task 5 (ponta a ponta, a mais cara). Cada tarefa termina **commitada e verde**; o relatório diz exatamente onde parou.

## Self-review

- Cobertura do B: ticker `13-indices.js` e atualização de 30 s (T1), CEP de `14-pessoas.js` (T2). Bugs do B tratados: `cep` prometia "Correios/contingência" que não existia (agora a fonte declarada é só ViaCEP, como o B já exibe), valores como string solta sem unidade/data, MD5 como hash de idempotência (não necessário aqui), séries sem significado conferido.
- Regras do v2: nunca valor presumido; integrações desligadas por padrão; só 8 dígitos do CEP saem; cliente isolado com limites; nada se apaga; E2E só em pilha isolada.
- Fora do plano: cálculo de correção monetária/atualização de débitos com índices (o ticker é informativo), notificações (Fase F), segredo de justiça por credencial própria (Fase D), F13 (escopo por equipe/cliente).
