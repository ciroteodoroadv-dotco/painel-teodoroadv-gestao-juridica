# Fusão — Plano 7: Calculadora de prazo e calendário forense

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. **Antes de escrever, leia por inteiro, no B (somente leitura):** `web/js/12-prazo.js` (o drawer inteiro: textos, campos, resultado), `web/js/07-app-core.js` (o botão "Calculadora de prazo" na sidebar, inserido **depois de Audiências**), `web/js/15-drawers.js`, `web/css/painel.css` (`.form-block`, `.radio-row`, `.radio-opt`, `.field`, `.hint`); **servidor do B, só contraste**: `servidor/motorprazo/{motorprazo.go,http.go}` e os bugs de `docs/plano-fusao/painel-B-inventario.md` §3 (feriados únicos para o país; Quarta-feira de Cinzas e Corpus Christi tratados como feriado **sem fonte** — o vencimento sai **depois** do real, contra o advogado; Semana Santa incompleta; sem dobro; sem termo inicial por juntada/citação; tela dizia "Prazo fatal" sem aviso; erro do servidor em texto simples que a tela não lia). Leia também `docs/ROADMAP-PRAZOS-E-INTEGRACOES.md` (Fase E e princípios), `internal/prazos/*` atual e os `NOTAS-PLANO-*.md`. **Fontes legais a consultar** (só leitura; se o acesso for negado, não contornar — anotar pendência): CPC (Lei 13.105/2015) arts. 216, 219, 220, 224, 231, 180, 183, 186, 229; CPP art. 798; Código Civil art. 132; Código Penal art. 10; Lei 5.010/66 art. 62; Lei 662/49 e Lei 14.759/2023 (feriados nacionais). **O B vale na forma; o v2 vale na regra.**

**Goal:** A **Calculadora de prazo** do B (drawer aberto pelo botão da sidebar) calcula o vencimento com **calendário forense verificável por tribunal**, mostra **todas as premissas** e sempre rotula o resultado como **"A CONFERIR"**. O **Calendário forense** é cadastro do escritório, com **fonte citada por data** e quem verificou. **Nenhum prazo é criado sozinho** e o cálculo nunca substitui a conferência do advogado.

**Architecture:** Pacote **puro** `internal/prazos/calculo` (sem banco, sem relógio global: recebe a data e o calendário), tabelas `calendarios` e `calendario_dias`, serviço `internal/prazos/calendario` (cadastro auditado), handler da calculadora no domínio `prazos` (htmx: o resultado troca só `#resultadoPrazo`) e página **Calendário forense** no grupo "sistema". O calendário **nacional** vem semeado a partir da lei (com fonte); calendários **por tribunal** começam **vazios** — enquanto não existirem, o resultado diz "Sem calendário verificado deste tribunal: considerados só os feriados nacionais".

**Tech Stack:** Go 1.26, PostgreSQL 17 + goose, html/template, htmx local, JS dos planos anteriores.

**Spec:** `ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `ROADMAP-PRAZOS-E-INTEGRACOES.md`, planos 1B–6 e 4C.

## Global Constraints

- Tudo dos planos anteriores (CSP estrita sem `on*=`/`style=`/`innerHTML`/`eval`; arquivos ≤ 300 linhas; `gofmt`, `go vet`, `go test -race -p 1 ./...` em contêiner; B somente leitura e hashes conferidos; dados fictícios; sem credenciais reais; sem push; commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`; teste **vermelho real** antes de implementar; **nada se apaga** — dia de calendário errado é **inativado com motivo**, não apagado).
- **Resultado sempre "A CONFERIR"** (selo `badge-warn`), nunca "Prazo fatal" sem qualificação; mostra **premissas** (termo inicial, dias, regra, calendário e **cada dia excluído com sua fonte**).
- **Regra jurídica é do advogado:** o código implementa o que está em `docs/juridico/REGRAS-DE-PRAZO.md` (tabela regra × norma × comportamento × "validado pelo advogado em __/__/____"). Enquanto `PAINEL_CALCULADORA_REGRAS_VALIDADAS` não for `true`, o resultado traz o bloco `.form-block` vermelho "**Regras de cálculo ainda não validadas pelo advogado — não use para prazo real.**" Itens duvidosos ficam marcados `[VERIFICAR]` no documento e **no resultado** (nunca resolver dúvida jurídica por conta própria).
- **Nunca errar a favor de um prazo mais longo:** na dúvida entre dois vencimentos, o sistema mostra o **mais curto** e lista a alternativa nas premissas ("se o tribunal adotar X, o vencimento seria Y").
- **Honestidade sobre o que não cobre:** o resultado lista, em texto fixo, o que **não** foi considerado (termo inicial por juntada de AR/mandado, citação, carga, consulta tácita de 10 dias, indisponibilidade do sistema do tribunal, suspensões por portaria não cadastradas, ponto facultativo não cadastrado, feriado municipal não cadastrado).
- Fuso `America/Sao_Paulo`; datas civis; nenhum `time.Now()` dentro do pacote de cálculo.

## Decisões (B na forma, v2 na regra)

| Tema | Decisão |
|---|---|
| Botão e drawer | Botão "Calculadora de prazo" na sidebar **logo depois de Audiências** (item de ação sem contador, abre o drawer). Drawer idêntico ao B: `h2` "Calculadora de prazo"; subtítulo "Ferramenta solta, não vinculada a ficha nenhuma — só calcula e mostra o resultado."; **Data do evento** (padrão hoje; hint "Data da disponibilização no Diário ou da ciência/intimação direta."); **Modo de contagem** (radios "Diário oficial (DJe)" marcado / "Ciência / intimação direta"); **Regra do prazo** (radios "CPC (dias úteis)" marcado / "CPP (prazo processual penal — dias corridos)" / "Prazo material cível (dias corridos)" / "Prazo material penal (dias corridos)"); **Número de dias**; `#resultadoPrazo`; botões "Fechar" / "Calcular". Erros do B: "Número de dias inválido." / "Falha ao calcular o prazo." (agora o servidor devolve a **mensagem do motivo**, corrigindo o B). |
| Campos **extras** (v2) | Depois do "Número de dias", dois campos opcionais, **identificados como extras do v2** em `hint`: **Calendário** (select: "Nacional (sem calendário local verificado)" + calendários cadastrados) e **Prazo em dobro** (select: "Não" / "Fazenda Pública (CPC art. 183)" / "Ministério Público (art. 180)" / "Defensoria Pública (art. 186)" / "Litisconsortes com procuradores distintos (art. 229 — não se aplica a processo eletrônico, §2º)"). Não alteram a ordem nem o texto dos campos do B. |
| Resultado (forma do B + regra do v2) | `.form-block`: `<strong>Vencimento calculado: dd/mm/aaaa</strong>` + selo `A CONFERIR`, a frase do B ("faltam N dia(s) útil(eis)" / "vence hoje" / "venceu há N dia(s) útil(eis)"), a **lista de premissas** (publicação, início da contagem, regra, calendário usado, recesso, cada dia excluído com a fonte) e o aviso fixo "Cálculo de apoio. O advogado confere o prazo no processo e no calendário do tribunal." |
| "Criar prazo com este cálculo" | Botão `btn btn-ghost` no resultado → abre o formulário de Prazos (`/prazos/novo`) **já preenchido** (data, fundamento = resumo das premissas, fonte "calculadora") — **continua A CONFERIR** até confirmação por advogado (regra de ouro). Nada é criado sem esse clique. |
| Texto na tela Prazos | O aviso atual ("o sistema ainda não calcula dia útil nem feriado forense") passa a: "Cálculo de apoio disponível na **Calculadora de prazo**; o resultado é sempre **A CONFERIR**." |
| Calendários por tribunal | Cadastro auditado: `calendarios(id, sigla, nome, ambito)` (ex.: "Nacional", "Justiça Federal (TRF1–6)", "TJSP", "TRT-2"); `calendario_dias(calendario_id, data, tipo ['feriado','suspensao_prazo','recesso','sem_expediente','ponto_facultativo'], conta_como_nao_util boolean, descricao, fonte_citacao text not null, fonte_url, verificado_por, verificado_em, ativo, motivo_inativacao)`. **Sem fonte não grava.** Ponto facultativo **não** é feriado por padrão (`conta_como_nao_util=false` até o advogado marcar com fonte do tribunal). Só advogado/administrador cadastram; tudo auditado. |
| Semente nacional (com fonte) | Fixos: 01/01, 21/04, 01/05, 07/09, 12/10, 02/11, 15/11, 25/12 (Lei 662/49 e atualizações), **20/11** (Lei 14.759/2023 — **a partir de 2024**; antes disso **não** é feriado nacional) e **Sexta-feira Santa** (móvel, por Páscoa) — cada um com a **norma** na `fonte_citacao`; **suspensão 20/12 a 20/01** (CPC art. 220) como regra de recesso. **Não** entram no nacional: Quarta-feira de Cinzas, Corpus Christi, Carnaval, Quarta e Quinta-feira Santa (dependem do tribunal; o calendário "Justiça Federal" traz Semana Santa de quarta a domingo, segunda e terça de Carnaval, 11/08 e 08/12 pela **Lei 5.010/66 art. 62**, **com a fonte** e marcado `[VERIFICAR]` até o advogado validar). |
| Contagem | Dias úteis (CPC art. 219): **exclui o dia do começo, inclui o do vencimento (art. 224)**; início no **primeiro dia útil seguinte**; no modo DJe a **publicação é o primeiro dia útil seguinte à disponibilização** e o início é o dia útil seguinte à publicação (art. 224 §§2º–3º); no modo ciência o início é o primeiro dia útil seguinte à ciência; **suspensão de 20/12 a 20/01** não conta; vencimento em dia não útil **prorroga** para o próximo dia útil (art. 224 §1º). **CPP (art. 798):** corridos; não se computa o dia do começo, computa o do vencimento; prorroga se vencer em domingo ou feriado; recesso **não aplicado por padrão** e marcado `[VERIFICAR]`. **Material cível (CC art. 132):** exclui o dia do começo, inclui o do vencimento, **prorroga** se cair em dia não útil. **Material penal (CP art. 10):** **inclui** o dia do começo, sem prorrogação. Prazo em dobro: multiplica os dias **antes** da contagem. |
| Escopo desta fatia | Termo inicial por juntada/citação/carga, consulta tácita (10 dias) e prazos em horas/meses/anos **ficam fora** e aparecem na lista "não considerado". |

---

## Task 1: Migração 0014 e semente do calendário nacional

**Files:** Create `migrations/0014_calendario_forense.sql` (+ semente em Go `internal/prazos/calendario/semente.go` que insere, na primeira execução, os dias **fixos** do ano corrente e dos 2 seguintes; os móveis são calculados no cálculo, não gravados); Test `internal/app/migracoes_0014_test.go`.
- [ ] **Step 1: testes que falham:** `calendario_dias` sem `fonte_citacao` → violação; `conta_como_nao_util` em `ponto_facultativo` aceito só com `verificado_por`; unicidade `(calendario_id, data, tipo)`; `painel_app` não apaga `auditoria` nem remove linhas de `calendario_dias` (só inativa); a semente **não duplica** ao rodar duas vezes e traz `fonte_citacao` em todo registro; 20/11 só aparece a partir de 2024.
- [ ] **Step 2–4:** ver falhar, escrever, passar. **Step 5:** `restore-check`. **Step 6: commit.**

## Task 2: Pacote puro de cálculo (o coração; teste independente)

**Files:** Create `internal/prazos/calculo/{regras,calendario,pascoa,contagem,dobro,premissas}.go` e testes.

**Interfaces — Produces:**
```go
type Regra string // "cpc-uteis","cpp-corridos","material-civel","material-penal"
type Modo string  // "dje","ciencia"
type DiaNaoUtil struct{ Data time.Time; Tipo, Descricao, Fonte string }
type Calendario interface{ NaoUtil(d time.Time) (DiaNaoUtil, bool) } // sábado/domingo tratados dentro do cálculo
type Entrada struct{ Evento time.Time; Modo Modo; Regra Regra; Dias int; Dobro string; Cal Calendario; Hoje time.Time }
type Resultado struct{ Publicacao, Inicio, Vencimento *time.Time; DiasUteisAteVencimento int; Premissas []string; Excluidos []DiaNaoUtil; Alternativas []string; NaoConsiderado []string; Duvidas []string }
func Calcular(e Entrada) (Resultado, error) // erros com motivo em português ("Número de dias inválido.", "Data do evento inválida.", …)
func Pascoa(ano int) time.Time
```
**Regras:** as da tabela "Contagem". A Páscoa pelo algoritmo anônimo gregoriano; Sexta-feira Santa = Páscoa − 2 dias. O calendário **nunca** trata como não útil o que não tiver **fonte** no registro. Se o calendário escolhido não tiver nenhum dia cadastrado para o ano, acrescenta a premissa "Sem calendário verificado deste tribunal" e uma **dúvida**.
- [ ] **Step 1: testes que falham (casos dourados calculados à mão, no comentário de cada teste, com a norma — nunca gerados pela própria função):**
  1. DJe: disponibilização sexta 2026-11-06, 15 dias úteis, calendário nacional → mostrar o cálculo manual passo a passo (publicação segunda 09/11; início terça 10/11; exclui 20/11 feriado; saltar fins de semana) e comparar;
  2. disponibilização na véspera do recesso (18/12) e vencimento **depois** de 20/01 (dia 21/01 é o primeiro dia contável);
  3. vencimento caindo em sábado → prorroga para segunda; caindo em feriado móvel (Sexta Santa) → prorroga;
  4. ciência direta numa sexta → início segunda;
  5. CPP corridos: 5 dias a partir de quinta, vencendo domingo → segunda; recesso **não** aplicado e `Duvidas` cita `[VERIFICAR]`;
  6. material cível: exclui o dia do começo e prorroga; material penal: **inclui** o dia do começo e **não** prorroga;
  7. prazo em dobro (Fazenda): 15 → 30 dias úteis;
  8. 20/11/2023 **não** é feriado e 20/11/2024 **é**;
  9. Quarta-feira de Cinzas e Corpus Christi **não** são excluídos no calendário nacional (regressão do bug do B) e **são** quando o calendário do tribunal os traz com fonte;
  10. `Pascoa` para 2024 (31/03), 2025 (20/04), 2026 (05/04), 2027 (28/03);
  11. entradas inválidas (dias 0 ou negativo, data zero, regra desconhecida) → erro com a frase certa;
  12. **propriedades:** vencimento ≥ início; vencimento nunca em dia não útil (regras com prorrogação); aumentar `Dias` nunca antecipa o vencimento; mesma entrada → mesmo resultado (determinismo);
  13. `DiasUteisAteVencimento`: 0 no dia, positivo antes, negativo depois (com os textos do B montados na tela).
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 3: Cadastro do calendário forense (serviço, auditoria, página)

**Files:** Create `internal/prazos/calendario/{model,repo,service,handler}.go`, `templates/{pagina,drawer}.html`; Modify `internal/auth` (RBAC: `PermCalendarioRead` para quem lê prazos; `PermCalendarioWrite` só advogado/administrador), `internal/app`; Test `internal/prazos/calendario/*_test.go`, `internal/app/calendario_test.go`.

**Tela (mesmo vocabulário visual do B; página do grupo "sistema", fora do B):** `Calendário forense` / sub "Feriados e suspensões com a fonte de cada data. Sem fonte, não entra no cálculo."; botão "Novo dia"; filtro por calendário e ano; tabela `Data | Tipo | Descrição | Conta como não útil | Fonte | Verificado por | (Inativar)`; drawer "Novo dia de calendário" (Calendário, Data, Tipo, Descrição, **Fonte (norma/portaria) \***, Link da fonte, "Conta como dia não útil", Observação) e "Novo calendário" (sigla, nome, âmbito); **Inativar** pede **Motivo \*** (nada se apaga; dia inativado deixa de contar e permanece listado com selo "Inativo"). Marca `[VERIFICAR]` visível nos itens ainda não validados pelo advogado.
- [ ] **Step 1: testes que falham:** sem fonte → recusa; sem permissão → 403; criar/inativar audita (`calendario.dia_criado/inativado`); inativar exige motivo e mantém a linha; dia inativo **não** entra no `Calendario` entregue ao cálculo; duplicidade recusada; leitura só de quem lê prazos; HTML sem `on*=`/`style=`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5:** adaptador `Calendario` (banco → interface do cálculo) com cache curto e invalidação ao gravar. **Step 6: commit.**

## Task 4: Calculadora na tela (drawer do B, resultado do v2)

**Files:** Create `internal/prazos/calculadora.go`, `templates/{calculadora,resultado}.html`; Modify `internal/auth/shell.go` (item de ação "Calculadora de prazo" depois de Audiências), `internal/prazos/templates/lista.html` (texto), `internal/platform/config` (`PAINEL_CALCULADORA_REGRAS_VALIDADAS`), `web/static/forms.js` (se necessário); Test `internal/prazos/calculadora_test.go`, `internal/app/calculadora_tela_test.go`.
- [ ] **Step 1: testes que falham:** `GET /prazos/calculadora` (com `HX-Request`) devolve o fragmento do drawer com **os rótulos do B na ordem do B** (extrair `<label>`/`legend`/`h2`/`hint` e comparar com a lista literal) e os dois extras depois do "Número de dias"; `POST /prazos/calcular` com dados válidos devolve só o bloco de resultado com o selo **A CONFERIR**, a lista de premissas e a frase "faltam/vence hoje/venceu há…" do B; dias inválidos → "Número de dias inválido." com o motivo; regra desconhecida → mensagem do motivo (não texto genérico); `PAINEL_CALCULADORA_REGRAS_VALIDADAS=false` → bloco vermelho de aviso presente; `true` → ausente; o resultado **nunca** contém a expressão "Prazo fatal:" sem "A CONFERIR"; botão "Criar prazo com este cálculo" leva a `/prazos/novo` preenchido e **não** cria nada por si; exige sessão e `PermPrazoRead`; auditoria `prazo.calculado` (parâmetros, sem dado de cliente); sem `on*=`/`style=`.
- [ ] **Step 2–4:** ver falhar, implementar, passar. **Step 5: commit.**

## Task 5: Regras para validação, docs, seed, paridade e verificação final

- [ ] **Step 1:** `docs/juridico/REGRAS-DE-PRAZO.md` (para o advogado validar): uma linha por regra (norma, o que o sistema faz, exemplo numérico, `[VERIFICAR]` quando houver dúvida, campo "Validado por / data"). `docs/operacao/CALCULADORA-DE-PRAZO.md` (leigo): o que calcula, o que **não** calcula, por que é sempre "A CONFERIR". Termos novos em `GLOSSARIO-PLANO-7.md` (dia útil, recesso, termo inicial, prorrogação, prazo em dobro, calendário forense, ponto facultativo).
- [ ] **Step 2:** seed fictício: calendário "Justiça Federal (TRF1–6)" **marcado `[VERIFICAR]`** com os dias da Lei 5.010/66 e um calendário de teste "TRIBUNAL DE TESTE" com 2 dias; instantâneos e **paridade B × v2** do drawer da calculadora (rótulos, `.radio-row`, `.field`, `.form-block`, `.drawer-actions`) no Chrome — **zero diferença não justificada** nos campos do B; os extras e o selo A CONFERIR são as diferenças justificadas; `tools/refb/textos7.py` (módulo 12).
- [ ] **Step 3:** executar no Chrome o JS novo (troca do `#resultadoPrazo` por htmx; botão "Criar prazo…"); dizer o que não foi verificado.
- [ ] **Step 4:** `gofmt -l .`; `go vet ./...`; `go test -race -p 1 ./...` (contêiner, **rodada final**); `scripts/check-lines.sh`; `sha256sum -c HASHES-B.txt`; conferir via `psql` (socket local, sem ler credenciais) que as tabelas afetadas estão sem dado de negócio antes de reconstruir o app dev; `GET /login` 200; CSP intacta; `PAINEL_CALCULADORA_REGRAS_VALIDADAS` ausente/`false`; sem credenciais reais; remover contêineres de teste.
- [ ] **Step 5:** `RELATORIO-7.md`, `NOTAS-PLANO-7.md`, `ACEITE-VISUAL.md`; ficha do projeto no acervo; commit.
- [ ] **Step 6 (com o advogado):** **revisão jurídica** de `REGRAS-DE-PRAZO.md`; só depois do "validado" o advogado liga `PAINEL_CALCULADORA_REGRAS_VALIDADAS=true`. Cadastrar os calendários dos tribunais que o escritório usa, **cada dia com a fonte**.

## Self-review

- Cobertura do B: drawer e botão (T4), resultado e frases (T4), os 4 tipos de regra (T2). Bugs do B corrigidos: feriados sem fonte (Cinzas/Corpus Christi), Semana Santa incompleta, 20/11 sem distinção de ano, material cível sem prorrogação, CPP com recesso do CPC, erro ilegível na tela, "Prazo fatal" sem ressalva, calendário único.
- Regras do v2: sempre "A CONFERIR"; premissas e fontes à vista; nunca arredonda a favor de prazo mais longo; dúvida jurídica marcada `[VERIFICAR]` e decidida pelo advogado; nada se apaga; tudo auditado.
- Fora do plano (explícito): termo inicial por juntada/citação/carga, consulta tácita, prazos em horas/meses/anos, sugestão automática de prazo a partir do texto da intimação, notificações (Fase F).
