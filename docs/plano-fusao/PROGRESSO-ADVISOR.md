# Progresso do advisor (/painel)

Atualizado em 05/11/2026. Branch: `feat/fase-1-saneamento` (base: `origin/feat/advisor-command`).

## Feito
- Etapa 0 (saneamento):
  - `referencia/painel-b/`: 22 arquivos do painel B, conferidos por SHA-256 contra `HASHES-B.txt`.
  - Exceção em `scripts/check-lines.sh` para essa pasta, com nota datada no ADR 0006.
  - README: estado atual, fonte que governa, regra de prazo interno (ADR 0012).
  - Cabeçalhos de histórico em `docs/ESPECIFICACAO.md` e `docs/PLANO-IMPLEMENTACAO.md`.
  - 23 arquivos de histórico movidos para `docs/arquivo/plano-fusao/`, com índice em `docs/arquivo/INDICE.md`.
  - `scripts/check-links.sh` com `scripts/check-links.excecoes` (21 exceções, cada uma com motivo).
- Etapa M1-CI:
  - Job `migracoes` com o ciclo up→down→up (`internal/platform/migrate/ciclo_test.go`).
  - Lint com golangci-lint v2.5.0 no job `go`.

- Lint (05/11/2026): CI run 35 (`fe1e205`) falhou no job `go`, passo golangci-lint, com 6 ocorrências.
  Causa de o lint falhar em lotes: o golangci-lint corta a saída por padrão (`max-same-issues`),
  então o CI não mostrava todas. Rodando localmente com `--max-issues-per-linter=0 --max-same-issues=0`
  apareceram mais 5. As 11 foram corrigidas sem mudar comportamento:
  - `internal/app/calculadora_tela_test.go:113`: `_ = resp.Body.Close()`
  - `internal/app/documentos_test.go:62`, `internal/app/harness_test.go:135`, `internal/app/helpers_test.go:60`,
    `internal/app/tarefas_test.go:54`, `internal/cep/cliente.go:126`: `defer func() { _ = resp.Body.Close() }()`
  - `internal/intimacoes/djen/cliente.go:61`, `internal/monitor/datajud/cliente.go:66`: `!(a && b)` → `(!a || !b)`
  - `internal/platform/config/config.go:105`: `!(dev && http)` → `(!dev || u.Scheme != "http")`
  - `internal/prazos/calculo/corridos.go:108`: dois `!(a && b)` → `(!a || !b)`
  - `internal/processos/duplicado.go:60`: `!(a && b && c)` → `!a || !b || just != atualJust`
- Prova local (go1.26.10, golangci-lint v2.5.0 compilado com go1.26.10, Postgres 16):
  gofmt limpo; `go vet ./...` limpo; golangci-lint sem limite de saída: 0 issues;
  `go test -count=1 -p 1 ./...` com `PAINEL_TEST_DB`: tudo ok (`internal/app`: 108 pass, 2 skip manuais, 0 fail).

## Não verificado
- `go test -race` não rodou localmente; CI usa Postgres 17, local foi 16. Evidência final = CI após push.
- Lint local exige `GOTOOLCHAIN=go1.26.10`: o Go do container é 1.24.7 e o binário pré-instalado
  do golangci-lint foi compilado com go1.25, que recusa o módulo.
- SBOM e trivy: pendentes. As actions precisam de SHA fixado, e não consegui consultar.
- `scripts/analise-estatica.sh`: não executado (mesma causa dos testes Go).

## Ponto de parada
Correções de lint feitas localmente, sem commit e sem push. Próximo passo: autorização do advogado para commit e push,
depois observar o CI do branch.

## Último commit verificado
`fe1e205` (CI run 35): falhou no lint do job `go` (1 de 4 jobs com falha).
