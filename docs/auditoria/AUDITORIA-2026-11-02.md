# Auditoria de segurança do backend — 02/11/2026

**Alvo:** `painel-teodoro-v2` no commit `9efdd39` (7.128 linhas de Go de produção, 5 migrações, `deploy/`).
**Nenhum código foi alterado** nesta auditoria: este documento é o relatório e a proposta de correção.
Método guiado pela skill `golang-security` (modo *audit*, executado em sequência por uma só pessoa-IA, sem subagentes) e
pela regra "pesquisar antes de reportar": cada achado foi seguido até a origem do dado e a severidade foi **ajustada ao
nosso contexto** (aplicação só alcançável pelo Cloudflare Access, HTTP simples atrás do túnel).

## O que foi executado

| Ferramenta / verificação | Versão | Resultado |
|---|---|---|
| `govulncheck` (módulos + biblioteca padrão, com análise de alcance) | v1.1.4 | **16 vulnerabilidades alcançáveis** (ver F1–F3) |
| `gosec` (SAST) | v2.22.4 | 5 avisos; todos triados, nenhum explorável (F12) |
| `staticcheck` | 2025.1.1 | 3 avisos de qualidade, 1 com efeito prático (F6) |
| `go vet`, `gofmt`, limite de 300 linhas | — | limpos |
| Suíte inteira com **detector de corrida** (`-race`, contêiner Go, PostgreSQL real) | go1.26.8 | **sem corrida**; todos os pacotes passam |
| Busca de segredo literal no código de produção | — | nada encontrado |
| Teste dinâmico pontual na pilha dev (POST multipart sem login) | — | confirmou F4 |
| Revisão manual contra o checklist da skill (injeção, cripto, sessão/cookies, cabeçalhos, upload, autorização, logs, segredos, SSRF) | — | achados F4–F11 |

**NÃO foi feito:** Semgrep e CodeQL (não instalados; a skill de análise estática do acervo os usa), teste de invasão
dinâmico completo, *fuzzing*, revisão da configuração real do Cloudflare/Oracle (não existe ainda), auditoria do
`supply-chain-risk-auditor` (consulta APIs externas e o token do `gh`; não rodei sem o seu ok). **[Provável]** uma rodada com
Semgrep/CodeQL achará mais itens de qualidade; os de maior risco estão cobertos pela revisão manual.

## Achados, por severidade

Legenda de confiança: **[Certeza]** reproduzido ou confirmado por ferramenta; **[Provável]** inferência forte.

### ALTA

**F1 — Go 1.26.8 tem 14 vulnerabilidades da biblioteca padrão alcançáveis [Certeza]**
`crypto/x509` (corrigida na 1.26.10), `net/http`, `net/textproto`, `crypto/tls`, `html/template` (1.26.9): GO-2026-6599,
6600, 6603, 6605, 6607–6613, 6617, 6629 (esta última em `x/text`). **Impacto real:** as de HTTP/2 são de **servidor** e o
nosso servidor só fala HTTP/1.1 em texto simples atrás do túnel [Provável: não explorável como está]; as de
`html/template` (contexto de escape) importam porque renderizamos tudo por esse pacote, mas não temos JavaScript
dentro de modelos; `crypto/x509` toca a busca do JWKS do Cloudflare. Severidade alta pelo custo ínfimo de corrigir.
**Correção:** subir para `go1.26.10` (ou o mais novo) em `go.mod` (`go 1.26.10`), em `deploy/Dockerfile`
(`golang:1.26.10-bookworm`) e na imagem de teste; rodar `go test`. Esforço: 15 min.

**F2 — `pgx v5.7.2` com injeção de SQL por confusão de marcador em literal com `$$` (GO-2026-5004) [Certeza]**
Corrigida em `v5.9.2`. **Impacto real:** nossas consultas são todas parametrizadas e nenhuma monta texto SQL com `$$` a
partir de entrada de usuário [Certeza, por leitura]; o `govulncheck` aponta o caminho genérico (`pool.Query`). Mesmo assim é
a biblioteca que fala com o banco do cliente: **atualizar**. Correção: `go get github.com/jackc/pgx/v5@v5.9.2` + testes.

### MÉDIA

**F3 — `golang.org/x/text v0.27.0` (GO-2026-5970 laço infinito, GO-2026-6629 pânico) [Certeza]**
Corrigida em `v0.41.0`. Entra por `pgx`/`goose`. Correção: `go get golang.org/x/text@v0.41.0`.

**F4 — Corpo *multipart* de até 26 MiB é lido e processado ANTES de verificar login e CSRF [Certeza, reproduzido]**
`internal/auth/router.go` chama `ParseMultipartForm` antes de `session()`/`csrfOK()`. Teste na pilha dev: `POST /documentos`
sem login com 27.000.203 bytes foi **recebido por inteiro** e só então respondeu 403. Um cliente sem credencial (em produção,
quem passou pelo Access) pode ocupar ~26 MiB de memória por requisição. **Correção proposta:** resolver sessão (e rota/permissão) primeiro; só
então `ParseMultipartForm`; negar sem ler o corpo quando não houver sessão válida; opcionalmente limitar envios simultâneos.
Teste novo: POST grande sem sessão deve retornar rapidamente sem consumir o corpo.

**F5 — Toda negativa (403/CSRF) grava um evento de auditoria: crescimento sem limite e fila no *lock* da cadeia [Provável]**
`Auth.deny` → `audit.RecordPool` com `pg_advisory_xact_lock`. Quem tem acesso pode inflar `audit_events` e serializar a
escrita da cadeia (afeta o login de todos). **Correção:** limitar eventos de negação por origem/janela (ex.: no máximo 1
por minuto por IP+rota, com contador) e/ou fora do caminho crítico.

### BAIXA

**F6 — Erro de banco tratado como "código inválido" e ainda soma tentativa (`internal/auth/mfa.go:72`, staticcheck SA4006) [Certeza]**
Se o consumo do código de recuperação falhar por erro **transitório** do banco, a pessoa recebe "código inválido" e a
tentativa é contada como erro dela (e, repetido, pode bloquear a conta). Correção: separar erro interno (500, sem contar
tentativa) de código errado.

**F7 — Parâmetros do Argon2 lidos do hash armazenado sem teto (`VerifyPassword`, gosec G115) [Provável]**
Um hash adulterado no banco poderia pedir memória gigante. Só quem já escreve no banco consegue isso. Correção: recusar
`m > 256 MiB`, `t > 10`, `p > 16`.

**F8 — Confiança em `Cf-Connecting-Ip` sem validar a origem [Provável]**
Em produção só chega tráfego pelo túnel (JWT do Access exigido), então o cabeçalho é do Cloudflare. No **modo dev** é forjável
(contorna o limite por IP). Correção: só confiar no cabeçalho quando o JWT foi validado; senão, usar o endereço da conexão.

**F9 — Conexão com o banco com `sslmode=disable` por padrão [Certeza]**
Aceitável na rede interna do Docker (sem saída, sem porta). Vira falha se o banco sair do host/rede. Correção: documentar
e exigir `verify-full` quando o banco for externo.

### INFORMATIVO / DESENHO

**F10 — WebAuthn com verificação de usuário apenas "preferida".** Para administrador, exigir `required`.

**F11 — Cadeia de suprimento da entrega.** Imagens base (`golang:…`, `distroless`, `alpine`, `postgres`, `cloudflared`) e
GitHub Actions fixadas por **tag**, não por *digest*/hash. O CI já tem `govulncheck` — **ele reprovaria hoje** por F1–F3.
Faltam `gosec`/`staticcheck` no CI. Correção: fixar por digest/hash; adicionar os dois ao CI.

**F12 — Triagem do `gosec` (nenhum explorável).** G115 em `password.go:53` e `totp.go:42` (conversões de valores limitados;
ver F7); G203 em `agenda/handler.go:257` (`template.HTML` recebe saída já escapada do `html/template`; o mesmo padrão em
`tarefas/handler.go` já leva `#nosec` com a justificativa e por isso o `gosec` o contou como "Nosec: 1");
G304 em `documentos/storage.go:42,75` (caminho sempre validado por `^[0-9a-f]{64}$` + `filepath.Join`; teste cobre `../`).

**F13 — Autorização é "do escritório inteiro", sem escopo por equipe, cliente ou processo [Certeza, é decisão de desenho].**
Hoje qualquer advogado vê todas as pessoas, processos e documentos. Serve a um escritório pequeno; **para "grandes bancas"
é preciso ACL por equipe/carteira/cliente** (minimização da LGPD) antes de usar com mais de uma equipe. É trabalho de
arquitetura, não correção rápida.

## O que está bom (confirmado por teste ou leitura)

Consultas todas parametrizadas e `LIKE` escapado; `html/template` em toda saída e CSP sem `unsafe-inline`/`unsafe-eval`;
htmx com `allowEval:false`; cookies `__Host-`, `Secure`, `HttpOnly`, `SameSite=Strict`; sessão de 256 bits guardada só como
hash, rotacionada ao concluir o 2FA; CSRF por HMAC com comparação em tempo constante; senha Argon2id com comparação em
tempo constante e tempo igualado para e-mail inexistente; segredos só em arquivo/variável (nenhum literal no código);
AES-256-GCM com subchaves por finalidade; `crypto/rand` em todo token; auditoria encadeada com gatilho e papel de banco sem
poder de dono; envio de arquivo com tipo pelo conteúdo, nome aleatório e cifra em disco; logs com redação de campos
pessoais; sem corrida de dados na suíte inteira.

## Ordem de correção sugerida (aguardando o seu ok; nenhuma foi aplicada)

1. **F1 + F2 + F3** juntos (toolchain, `pgx`, `x/text`) — 1 commit, rodar suíte, `govulncheck` deve zerar. ~30 min.
2. **F4** (multipart só depois da sessão) com teste. **F6** e **F7** (pequenos, com teste).
3. **F11** (CI com `gosec` + `staticcheck`; fixar imagens e ações por hash).
4. **F5** e **F8** (limite de eventos de negação; origem do IP).
5. Antes de qualquer uso com mais de uma equipe: **F13** (escopo/ACL) como marco próprio.
