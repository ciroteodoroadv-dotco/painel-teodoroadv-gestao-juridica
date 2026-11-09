# Auditoria técnica: Painel Teodoro

**Data:** 08/11/2026 · **Escopo:** árvore de trabalho atual (`main` + alterações não commitadas) · **Modo:** somente leitura. O repositório não foi alterado; `git status` segue idêntico ao inicial.

## Resumo executivo

A base de segurança é sólida: autenticação com 2FA, RBAC com negação por padrão, CSRF, JWT do Cloudflare Access validado, Argon2id, cifra AES-GCM dos documentos e cadeia de auditoria encadeada. Os riscos estão em outra camada:

1. **Alteração não commitada que liga as coletas externas** (DJEN, DataJud, BCB) contra a ADR-0011. O binário atual ainda não tem esse comportamento.
2. **Testes que dão falsa segurança.** Sem PostgreSQL, 179 de 323 execuções são puladas e a saída ainda diz `ok`.
3. **Código morto e não versionado** que duplica implementações ativas, inclusive uma rota sem autenticação.
4. **Regra jurídica divergente** no cálculo de prazos (prazo em dobro), a validar pelo advogado.
5. **Migrações que não aplicam sobre dados existentes**, o que bloqueia qualquer conversão de banco com registros.

## Verificação prática executada

| Verificação | Resultado |
|---|---|
| `go build ./...` (árvore atual e `HEAD` via `git archive`) | OK nos dois |
| `go vet ./...` | Sem achados |
| `go test ./...` sem banco | `ok`, mas **179 de 323 execuções puladas** (`PAINEL_TEST_DB` ausente) |
| `go test -race -count=1 -p 1 ./...` com PostgreSQL 17 descartável | **1 falha real** (CRLF, ver 1.2) e 0 data races. Dois pacotes falharam por bloqueio do Windows e passaram na repetição |
| `staticcheck` v0.8.2 | 0 achados |
| `gosec` v2.30.0 | 18 achados: 8 HIGH e 10 MEDIUM. Os abertos estão em código não versionado (`worker/datajud`, `crypto/vault`, `handler/live`). Os suprimidos são justificados |
| `govulncheck` v1.9.0 | 13 vulnerabilidades alcançáveis na biblioteca padrão, todas corrigidas em 1.27.2 ou 1.27.3, com o toolchain local em 1.27.0. Mais 1 em pacote importado e 1 em módulo (`golang.org/x/crypto` v0.58.0, pacote `openpgp`, não chamado) |
| `scripts/check-lines.sh` | OK (nenhum arquivo de produção acima de 300 linhas) |
| `scripts/check-links.sh` | Não rodou: chama `python3`, ausente neste Windows |
| `scripts/analise-estatica.sh` | Não rodou: grava arquivo dentro do projeto. Rodei as mesmas três ferramentas diretamente |
| `docker compose config` (base e prod, em cópia com segredos de teste) | OK |
| `docker build deploy/Dockerfile` (em cópia) | OK. Recusa subir sem segredos e roda como `nonroot` |
| Sonda de IDs malformados (em cópia) | Confirmou o item 6.1 |

## 1. FALHAS CRÍTICAS E IMPEDITIVAS

**1.1 Coletas externas forçadas a LIGADAS no código, sem o controle da ADR-0011**
- **Gravidade:** P1 · **Estado:** confirmado por leitura de código; comportamento não executado com rede.
- **Localização:** `cmd/painel/main.go:88–117` (alteração não commitada, 07/11 22:21).
- **Evidência:** `DJEN{Ativo: true}`, `DataJud{Ativo: true}`, `Indices{Ativo: true}`. As variáveis `PAINEL_*_ATIVO` deixam de ser lidas. O `Iniciar` recebe a **primeira espera**, não o intervalo: `svcs.Monitor.Iniciar(ctx, log, 5*time.Second)`. O log diz `"monitor do DataJud LIGADO", "intervalo", "5s"`, o que está errado: o intervalo real é `cfg.Intervalo` (24 h).
- **Explicação técnica:** a primeira rodada das três coletas começa entre 2 s e 3 min após cada subida, e cada reinício dispara novas consultas às APIs externas.
- **Analogia jurídica:** é como um escritório que liga o protocolo eletrônico automaticamente no primeiro minuto do expediente, sem o advogado ter validado o acesso. O sistema diz "ativo" no painel, mas o procedimento de validação nunca aconteceu.
- **Comportamento esperado:** coleta desligada por padrão; só liga por `PAINEL_*_ATIVO=true` após o teste manual orientado (ADR-0011, item 3).
- **Comportamento atual:** liga sempre. Com OABs ou processos cadastrados e chave configurada, sai dado do escritório (número OAB, CNJ) para fora sem o teste manual.
- **Causa-raiz:** alteração de depuração ("para testes") gravada na árvore principal. Nenhum teste cobre `cmd/painel`.
- **Impacto:** privacidade (LGPD) e contrato com o cliente; consultas repetidas às APIs a cada reinício; log que mente sobre o estado.
- **Verificação:** `git diff cmd/painel/main.go` e leitura de `Iniciar` em `internal/monitor/agendador.go:15`.
- **Correção recomendada:** restaurar o trecho do `HEAD` (`Ativo` lido de `cfg`, intervalos de `cfg`). Se o intervalo de 5 s for teste, usar uma variável de ambiente explícita e nunca em produção.
- **Relações:** 9.2, 9.3. O binário `painel.exe` (07/11 00:23) é anterior à alteração e ainda não contém o comportamento.

**1.2 Pacote de testes com falha em ambiente Windows (CRLF)**
- **Gravidade:** P3 (ambiente) · **Estado:** confirmado e reproduzido.
- **Localização:** `internal/app/processos_test.go:124–125`; template `internal/pessoas/templates/lista.html`.
- **Evidência:** o HTML renderizado tem `</div>\r\n    </div>`; o teste procura `</div>\n    </div>`. Com `fim = -1` nos quatro grupos, as quatro asserções falham. Convertendo só o template da cópia para LF, o teste passa.
- **Explicação técnica:** o `core.autocrlf=true` deste PC grava CRLF na cópia de trabalho; o repositório guarda LF. O teste depende de fim de linha.
- **Analogia jurídica:** é como conferir um despacho pela vírgula exata: a decisão está certa, mas a comparação literal falha pela formatação.
- **Comportamento esperado:** teste independente de fim de linha.
- **Comportamento atual:** falha em Windows; a CI Linux não é afetada (não verifiquei o log da CI).
- **Causa-raiz:** teste com comparação literal de HTML; ausência de `.gitattributes`.
- **Impacto:** o desenvolvedor não consegue usar `go test` como sinal confiável neste PC.
- **Correção recomendada:** comparar com normalização de `\r\n`, ou criar `.gitattributes` com `* text=auto eol=lf` para templates e código.
- **Relações:** 9.1. Também faz `go.mod` aparecer como modificado sem mudança de conteúdo.

**1.3 Código de produção fora do controle de versão**
- **Gravidade:** P2 · **Estado:** confirmado (`git ls-files --others`).
- **Localização:** 17 arquivos Go em `internal/{calculator,crypto,handler,middleware,router,sse,worker}`; `Dockerfile`, `docker-compose.yml`, `iniciar.ps1` na raiz; `painel.exe` (29 MB). Nenhuma regra de `.gitignore` cobre `*.exe`, nem os arquivos da raiz.
- **Causa-raiz:** geração em momentos diferentes sem integração ao repositório e sem regra de ignore.
- **Impacto:** `git add .` leva binário de 29 MB, segredos de desenvolvimento (2.4) e código morto ao histórico. O `HEAD` compila, mas não reflete o que está na pasta.
- **Correção recomendada:** decidir o que é versionado; ignorar `*.exe`, `docker-compose.yml`/`Dockerfile`/`iniciar.ps1` da raiz (ou movê-los para `deploy/`) e apagar os pacotes mortos (3.1).

## 2. SEGURANÇA E PRIVACIDADE

**2.1 Rota de busca e canal de eventos sem autenticação, em pacote não ligado**
- **Gravidade:** P2 (latente) · **Estado:** confirmado (código); não alcançável hoje.
- **Localização:** `internal/router/router.go:23–24` (`/events` e `/api/processos/busca`); `internal/handler/live/search.go:54–66`.
- **Evidência:** `mux.Handle("/events", cfg.Hub)` e `mux.HandleFunc("/api/processos/busca", ...)` sem passar por `auth.Router`. O `search.go` monta HTML com `fmt.Fprintf` e `query` sem escape. O `gosec` marca como G705 aberto.
- **Explicação técnica:** o `Router` do projeto é o único caminho de registro com permissão obrigatória; este pacote o contorna.
- **Analogia jurídica:** é uma porta lateral do cartório sem recepção, que dá acesso aos autos pelo balcão de busca.
- **Comportamento esperado:** toda rota passa por sessão, permissão e CSRF.
- **Comportamento atual:** qualquer chamada anônima a `/api/processos/busca` pode consultar processos. Se o pacote for ligado, ele também abre XSS refletido.
- **Causa-raiz:** código gerado fora da arquitetura de rotas.
- **Impacto:** nenhum hoje; P0 se alguém ligar `router.NewRouter`.
- **Correção recomendada:** remover (3.1) ou reescrever sobre `auth.Router`.

**2.2 Isolamento de dados entre advogados e equipes inexistente**
- **Gravidade:** P2 · **Estado:** confirmado; decisão de produto pendente (auditoria de 05/11, item 2.4).
- **Localização:** repositórios de `processos`, `pessoas`, `documentos` (ex.: `documentos/handler.go:158`, `Baixar`).
- **Evidência:** qualquer `advogado` ou `assistente` lê todo processo, pessoa e documento. A permissão é por papel, não por carteira.
- **Analogia jurídica:** todos os advogados do escritório têm a chave do arquivo geral, mesmo sem serem os responsáveis pelo caso.
- **Impacto:** sigilo entre equipes e cliente depende de disciplina, não de controle técnico.
- **Correção recomendada:** definir com o escritório quem vê o quê; depois aplicar filtro por responsável ou equipe nas consultas.

**2.3 Constante com formato de credencial em código morto**
- **Gravidade:** P3 · **Estado:** confirmado que existe; natureza do valor é hipótese (provavelmente chave pública de exemplo do DataJud).
- **Localização:** `internal/worker/datajud/client.go:15`; `gosec` G101 HIGH aberto.
- **Correção recomendada:** remover com o pacote. Se o valor for real, rotacionar.

**2.4 Segredos de desenvolvimento em arquivos não ignorados**
- **Gravidade:** P3 · **Estado:** confirmado.
- **Localização:** `docker-compose.yml` e `iniciar.ps1` na raiz (senhas `devpassword`, chaves base64 previsíveis de 32 bytes, `PAINEL_DATAJUD_KEY`).
- **Impacto:** risco de commit; combinados com 1.1, o `iniciar.ps1` também ligaria a coleta do DataJud.
- **Correção recomendada:** ignorar os arquivos e gerar chaves por script.

**2.5 Toolchain com vulnerabilidades alcançáveis e versões divergentes**
- **Gravidade:** P2 · **Estado:** confirmado (`govulncheck`).
- **Localização:** toolchain local `go1.27.0`; `go.mod` declara `go 1.26.10` (CI usa esse arquivo); `deploy/Dockerfile` usa `golang:1.27.3`.
- **Evidência:** 13 vulnerabilidades na biblioteca padrão alcançáveis, corrigidas em 1.27.2 ou 1.27.3 (GO-2026-6599 a GO-2026-6617 e similares).
- **Causa-raiz:** versão não alinhada entre `go.mod`, CI, imagem e máquina de desenvolvimento.
- **Impacto:** a imagem de produção (1.27.3) provavelmente está corrigida, mas a CI testa com 1.26.10, que não foi verificado nesta auditoria.
- **Correção recomendada:** alinhar `go.mod`, CI e imagem na mesma versão corrigida e reexecutar `govulncheck`. Atualizar `golang.org/x/crypto` (achado não chamado, P4).

**2.6 Processos sigilosos armazenados com texto integral**
- **Gravidade:** P3 · **Estado:** confirmado (código); avaliação LGPD pendente.
- **Localização:** `internal/intimacoes/sigiloso.go`, `internal/monitor/sigilosos.go:169–170`.
- **Evidência:** o campo `Texto` (comunicação integral) é gravado na tabela de intimações.
- **Correção recomendada:** registrar a base legal no ROPA e avaliar se o resumo basta.

**2.7 Cache do CEP sem teto e sem expurgo**
- **Gravidade:** P4 · **Estado:** confirmado.
- **Localização:** `internal/cep/handler.go:79, 93`: `h.cache[cep] = ...` sem remoção de expirados.
- **Impacto:** crescimento de memória limitado pelo número de usuários autenticados e pelo rate limit por usuário.

**2.8 Limitador de taxa sem teto de chaves**
- **Gravidade:** P4 · **Estado:** confirmado.
- **Localização:** `internal/platform/ratelimit/ratelimit.go:33`. A limpeza só roda acima de 10.000 chaves e não há limite duro.
- **Impacto:** baixo; o acesso passa antes pelo Cloudflare Access.

**Controles verificados (sem achado):** sessão com expiração absoluta e ociosa (`auth/session.go`); CSRF por HMAC ligado à sessão (`platform/csrf`); RBAC com negação padrão e registro de rota obrigatório (`auth/router.go`, `auth/rbac.go`); JWT do Access com RS256 fixo, `iss`, `aud`, `exp`, `nbf` e e-mail (`cfaccess/cfaccess.go`); tempo constante no login (`auth/login.go`); upload validado por magic bytes, nome aleatório, cifra AES-GCM e download como `attachment` com `nosniff` e CSP `sandbox` (`documentos/`); cadeia de auditoria com trava consultiva e lista permitida de metadados (`audit/audit.go`); SQL parametrizado (checado nos concatenamentos, que só juntam fragmentos constantes); CSP sem `inline` (`TestTemplatesSemInline` passa).

## 3. ARQUITETURA E QUALIDADE DO CÓDIGO

**3.1 Sete pacotes sem importador duplicam implementações ativas**
- **Gravidade:** P3 · **Estado:** confirmado (grafo de importação).
- **Localização e duplicação:**
  - `worker/{bcb,datajud,djen}` duplica `indices/bcb`, `monitor/datajud`, `intimacoes/djen`, usando `database/sql` enquanto o projeto usa pgx.
  - `calculator/prazos` duplica `prazos/calculo`.
  - `crypto/vault` duplica `platform/crypt`, com outra cifra e caminhos sem validação.
  - `middleware` duplica `platform/httpx`.
  - `router`, `sse`, `handler/live` são o subsistema de busca/eventos já citado.
- **Defeito dentro do código morto:** `search.go` lê `id` como `int64` e a tabela usa `uuid` (`migrations/0004`, linha 30); o `Scan` falha e `if err == nil` descarta a linha em silêncio. A busca ficaria sempre vazia, se fosse ligada.
- **Analogia jurídica:** é como manter três cópias de uma mesma procuração em gavetas diferentes: uma delas pode ser a usada pelo cartório, e a outra ser assinada por engano.
- **Causa-raiz:** geração assistida sem remoção do que foi substituído.
- **Impacto:** ambiguidade de qual cifra e qual cliente são os oficiais; o gate `analise-estatica.sh` reprovaria a árvore com os 3 HIGH abertos (9.3).
- **Correção recomendada:** remover os pacotes (lista acima) após confirmar que nada os usa.

**3.2 Dívida de composição já documentada**
- **Gravidade:** P4 · **Estado:** confirmado (auditoria de 05/11, item 7.2).
- `auth.Auth` e `PageData` acumulam responsabilidades; a auditoria aceitou adiar.

## 4. REGRAS DE NEGÓCIO E JURÍDICO

**4.1 Afirmação de que prazo em dobro não se aplica à advocacia privada**
- **Gravidade:** P1 (a validar pelo advogado) · **Estado:** fortemente indicado. O texto do CPC art. 229 não foi consultado nesta sessão.
- **Localização:** `docs/juridico/REGRAS-DE-PRAZO.md` (emenda de 04/11); `internal/prazos/calculo/regras.go`.
- **Evidência:** o documento diz que o painel "não aplica prazo em dobro (advocacia privada)". Pelo texto do art. 229 que conheço, o dobro vale para **litisconsortes com procuradores de escritórios distintos**, inclusive na advocacia privada.
- **Explicação técnica:** a exclusão do dobro está fundamentada em premissa que contraria a norma citada, e o aviso só aparece no resultado.
- **Analogia jurídica:** é como uma tabela de prazos que ignora a regra de dobro para litisconsortes e, por isso, mostra um vencimento mais cedo do que o real. Nesse caso o erro é a favor da segurança, mas a premissa está errada.
- **Comportamento atual:** o resultado lista "prazo em dobro não considerado", sem calcular a alternativa.
- **Impacto:** cálculo errado em caso com litisconsórcio; o advogado pode confiar na explicação e não revisar.
- **Correção recomendada:** o advogado confere o art. 229 e decide. Se o dobro entrar, calcular como alternativa visível, como já se faz com outras leituras.

**4.2 Calculadora disponível mesmo sem validação das regras**
- **Gravidade:** P2 · **Estado:** confirmado.
- **Localização:** `internal/app/app.go:101` e `internal/prazos/handler.go:45`; `Validada` só altera o banner (`calculadora.go:119`).
- **Evidência:** a rota `POST /prazos/calcular` é registrada sempre que existe o serviço de calendário. A flag `PAINEL_CALCULADORA_REGRAS_VALIDADAS` só troca o aviso vermelho.
- **Explicação técnica:** a validação jurídica é declarada como pré-condição, mas o controle técnico é só cosmético.
- **Analogia jurídica:** é como liberar a minuta de petição no sistema antes da revisão do sócio, com uma faixa de "não use" no topo.
- **Impacto:** o resultado sempre sai "A CONFERIR", o que mitiga, mas não impede o uso.
- **Correção recomendada:** bloquear a criação de prazo a partir do cálculo enquanto a flag estiver desligada, ou documentar a decisão como risco aceito.
- **Relações:** 4.1, 4.3.

**4.3 Suspensão do processo penal (CPP 798-A) não aplicada**
- **Gravidade:** P2 · **Estado:** confirmado (o próprio documento admite a divergência).
- **Localização:** `docs/juridico/REGRAS-DE-PRAZO.md`, linha 9 e divergência 4; `calculo/contagem.go` (`duvidasCorridos`).
- **Evidência:** a Lei 14.365/2022 suspende o prazo de 20/12 a 20/01 no processo penal, com exceções. O sistema não aplica a suspensão e mostra o vencimento mais curto, que é o lado seguro.
- **Impacto:** o vencimento mostrado pode estar **antes** do real; é conservador, mas errado.
- **Correção recomendada:** decidir se implementa com as exceções dos incisos I a III, ou mantém o aviso.

**4.4 Premissas do CPC 224 §1º e da Lei 11.419 art. 5º não implementadas**
- **Gravidade:** P3 · **Estado:** confirmado (documento).
- **Evidência:** expediente encerrado antes do horário e indisponibilidade do sistema (CPC 224 §1º); consulta em até 10 dias (Lei 11.419, art. 5º). Constam em "não considera".
- **Correção recomendada:** manter o aviso e revisar após validação.

**4.5 README contradiz o próprio sistema**
- **Gravidade:** P3 · **Estado:** confirmado.
- **Localização:** `README.md`, seção "Regras permanentes".
- **Evidência:** diz "Automação não calcula prazo processual", mas o Plano 7 calcula prazos do CPC. A última frase está duplicada.
- **Correção recomendada:** reescrever a regra conforme a decisão 4.2.

## 5. BANCO DE DADOS

**5.1 Migrações 0008 a 0014 não aplicam sobre dados existentes**
- **Gravidade:** P1 (operacional) · **Estado:** confirmado (`RAISE EXCEPTION` no SQL).
- **Localização:** `migrations/0008_pessoas_processos_do_b.sql:16`, `0009:14`, `0010:15`, `0011:76`, `0012:53`, `0013:76`, `0014:43/61/64`.
- **Evidência:** cada uma recusa aplicar se as tabelas tiverem linhas, e recusa o `Down` se houver dados. O próprio `PENDENCIAS.md` classifica 0008 e 0009 como risco **alto**.
- **Analogia jurídica:** é como um sistema de processo que só aceita migrar de sistema se o acervo estiver vazio, sem um roteiro de conversão.
- **Impacto:** nenhum banco com dados reais consegue subir para esta versão sem trabalho manual não documentado.
- **Correção recomendada:** escrever o roteiro de conversão de dados antes de qualquer uso real.
- **Pontos positivos:** o `Down` é protegido por dados, e a exclusão lógica é consistente (`excluido_em`).

**5.2 Chaves estrangeiras sem índice (hipótese)**
- **Gravidade:** P3 · **Estado:** hipótese a validar. Heurística sobre 22 colunas `uuid REFERENCES`; 14 aparecem sem índice (ex.: `tarefa_pai_id`, `evento_id`, `oab_id`, `duplicado_de`).
- **Verificação:** `EXPLAIN` nas consultas de remoção em cascata com volume realista.
- **Correção recomendada:** índice para as colunas confirmadas.

## 6. BACKEND E APIs

**6.1 IDs malformados viram 500 em vez de 404**
- **Gravidade:** P3 · **Estado:** confirmado (sonda em cópia de teste).
- **Localização:** `GET /processos/{id}`, `GET /pessoas/{id}`, `GET /prazos/{id}/editar`, `GET /documentos/{id}/baixar`.
- **Evidência:** `abc` como ID responde `500 "Erro interno."`. `GET /tarefas/{id}/editar` responde 404 para o mesmo caso.
- **Explicação técnica:** o id chega ao PostgreSQL sem validação, e o erro de conversão de `uuid` vira 500.
- **Impacto:** semântica HTTP inconsistente; erros de entrada poluem o log como falhas do sistema.
- **Correção recomendada:** validar o formato do id com `webx.UUID` antes da consulta, em todas as rotas.

**6.2 Coletas externas com clientes bem protegidos (sem achado)**
- DJEN, DataJud, BCB e CEP exigem `https`, têm timeout, limite de leitura, `CheckRedirect` e `User-Agent` genérico. Os erros não carregam URL nem chave (`datajud/cliente.go:163`).

## 7. FRONTEND

**7.1 htmx vendorizado sem versão registrada**
- **Gravidade:** P4 · **Estado:** confirmado.
- **Localização:** `web/static/htmx.min.js` (minificado, sem cabeçalho de versão); `base.html` carrega sem hash de integridade (SRI).
- **Correção recomendada:** registrar a versão e o hash no repositório.

**7.2 Não avaliado: acessibilidade e responsividade**
- Não há navegador nesta auditoria. Nenhuma afirmação sobre a interface.

**Verificado sem achado:** todo `fetch` de escrita envia `X-CSRF-Token` (`app.js`, `forms.js`, `upload.js`, `cep.js`, `webauthn.js`); não há `onclick` nem `<script>` inline nos templates.

## 8. INFRAESTRUTURA E DEVOPS

**8.1 Infraestrutura de produção nunca executada numa VM real**
- **Gravidade:** P2 (operacional) · **Estado:** confirmado pela própria documentação (`PLANO-IMPLEMENTACAO.md`, item 118).
- **Verificado nesta auditoria:** `docker compose config` (base e prod), build da imagem, recusa de subida sem segredos, usuário `nonroot`.
- **Não verificado:** cloud-init, firewall nftables, timers systemd, túnel Cloudflare, restauração de backup em VM.
- **Correção recomendada:** rodar o runbook numa VM de teste antes de qualquer dado real.

**8.2 Dockerfile e compose da raiz divergem da imagem oficial**
- **Gravidade:** P3 · **Estado:** confirmado.
- **Evidência:** raiz usa `golang:alpine` (tag móvel), `alpine:latest`, roda como root, sem digest; compose da raiz usa `postgres:16`, enquanto CI e produção usam `17`.
- **Correção recomendada:** remover ou apontar para `deploy/`.

**8.3 Operador com sudo sem senha e no grupo `docker`**
- **Gravidade:** P3 · **Estado:** confirmado (`cloud-init.yaml`).
- **Evidência:** `painel-admin` tem `NOPASSWD:ALL` e pertence ao grupo `docker`, que equivale a root.
- **Impacto:** aceitável para um operador único com chave SSH, mas sem separação de funções.

**8.4 Restauração executa linhas do manifesto com `source`**
- **Gravidade:** P4 · **Estado:** confirmado (`deploy/backup/restore.sh`).
- **Mitigação:** o manifesto vem dentro do arquivo cifrado com `age`.
- **Correção recomendada:** ler as chaves do manifesto com `grep`/`cut`, sem `source`.

**Verificado sem achado:** `deploy.sh` (imagem por digest, backup antes, migração separada, rollback documentado com seus limites); `bootstrap.sh` (valida a sub-rede do bastion); segmentação de rede (banco em rede `internal: true`, nenhuma porta pública em produção, dev só em `127.0.0.1`).

## 9. TESTES E PROCESSO

**9.1 "ok" sem PostgreSQL esconde 179 execuções puladas**
- **Gravidade:** P2 · **Estado:** confirmado.
- **Evidência:** `go test ./...` sem `PAINEL_TEST_DB` diz `ok` com 179 `SKIP`. Autorização, migrações, auditoria e fluxos de negócio estão no pacote `internal/app`, que só roda com banco.
- **Correção recomendada:** falhar a suíte local quando o banco estiver ausente, ou exigir o banco no script de teste.

**9.2 Wiring de produção sem nenhum teste**
- **Gravidade:** P2 · **Estado:** confirmado (`go test` lista `cmd/painel` como `[no test files]`).
- **Impacto:** a alteração de 1.1 passou sem nenhuma verificação automática.
- **Correção recomendada:** teste que monta o `serve()` com configuração de coleta desligada e confirma zero chamadas de rede.

**9.3 Gate de análise estática reprovaria a árvore atual**
- **Gravidade:** P3 · **Estado:** confirmado com `gosec` (o script em si não foi executado porque grava no projeto).
- **Evidência:** 3 HIGH abertos e 2 MEDIUM abertos, todos em código não versionado (3.1).
- **Correção recomendada:** resolver 3.1 antes de commitar.

**9.4 Falha local por bloqueio do Windows (transitória)**
- **Gravidade:** P4 (ambiente) · **Estado:** confirmado. `codigos.test.exe` e `tarefas.test.exe` foram bloqueados pelo "Controle de Aplicativo" na primeira execução e passaram na repetição.
- **Correção recomendada:** nenhuma no código. Se recorrer, avaliar a política de execução da máquina.

## 10. PERFORMANCE

**Sem achados comprovados por medição.** Não houve benchmark nesta auditoria. Observações sem evidência de carga:
- Listas de tarefas, agenda, audiências e intimações sem paginação (decisão registrada na auditoria de 05/11, item 4.5).
- A coleta do DataJud processa processos em sequência, com pausa de 1 s por consulta. Com muitos processos, a rodada pode levar horas. É desenho intencional, mas não medido.

## COBERTURA E LIMITAÇÕES DA AUDITORIA

**Inventário:** 519 arquivos (240 mil linhas, incluindo um arquivo de 29.932 linhas de senhas comuns e um binário de 29 MB). Código de produção: 191 arquivos Go (21.877 linhas). Testes: 90 arquivos (12.329 linhas). 15 migrações SQL. 49 templates. 9 arquivos JS próprios. 132 rotas registradas.

**Examinado em profundidade:** `cmd/painel/main.go`; `internal/auth` (roteador, RBAC, sessão, login, senha, MFA em parte, CSRF); `internal/platform` (cfaccess, config, crypt, csrf, httpx, ratelimit); `internal/documentos` (upload, armazenamento, detecção, download); `internal/audit`; `internal/prazos/calculo` (regras, contagem, corridos, calendário, Páscoa); clientes HTTP externos; migrações (guardas e `Down`); `deploy/` (compose, `deploy.sh`, `bootstrap.sh`, `cloud-init`, firewall, backup e restauração); CI e release; templates e JS (sinks de XSS e CSRF).

**Lido por amostra ou pelo grep, sem leitura integral:** `agenda`, `atendimentos`, `audiencias`, `diligencias`, `tarefas` (serviço), `processos` (serviço e repositório), `pessoas` (serviço e repositório), `intimacoes` (coleta e serviço), `monitor` (coleta), `indices`, `cep` (handler), `calendario` (serviço), `auth/wa.go`, `auth/totp.go`, `auth/admin.go`, `auth/account.go`, `auth/reauth.go`, `cli.go`.

**Não examinado:** `referencia/painel-b` (protótipo JS, não servido); `tools/refb` (Python, ferramenta de paridade); `docs/` além das ADRs, do REGRAS-DE-PRAZO e do PENDENCIAS; senhas comuns (lista de dados, sem lógica).

**Executado:** build, vet, testes (com e sem PostgreSQL, com `-race`), staticcheck, gosec, govulncheck, check-lines, docker compose config, docker build, sonda de API em cópia.

**Não executado:** golangci-lint (roda na CI), gitleaks, trivy, check-links (python3 ausente), `analise-estatica.sh` (grava no projeto), navegador (acessibilidade e responsividade), carga e benchmark, cloud-init e firewall numa VM.

**Limitações de acesso:** sem acesso à CI remota nem à Oracle, Cloudflare ou ao servidor de produção.

**Grau de confiança:**
- Alta nos achados confirmados por leitura de código e nos reproduzidos (1.2, 6.1, 9.1, 9.2, 5.1).
- Média em 1.1, que é comprovada pela leitura do código, mas não executada com rede.
- Baixa em 5.2 (heurística de índices).
- Dependente de validação jurídica em 4.1 (texto do art. 229 não consultado nesta sessão).

Não afirmo que o sistema está seguro ou que tudo foi verificado. Afirmo que os pontos listados foram examinados e o que ficou fora deste escopo.

**Estado do ambiente:** os contêineres de teste foram removidos. Os segredos de teste gerados ficaram no refugo do dia, em `D:\OneDrive\Desktop\00-REFUGOS\2026-11-08\`. O projeto não foi alterado.
