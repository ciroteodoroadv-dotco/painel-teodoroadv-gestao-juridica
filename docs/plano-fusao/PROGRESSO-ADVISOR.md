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
  - Passo `scripts/check-links.sh` no job `go` (05/11/2026): o CI não verificava referências da documentação.
    Teste negativo local: referência inexistente → status 1; sem ela → status 0.

- Lint (05/11/2026): CI run 35 (`fe1e205`) falhou no job `go`, passo golangci-lint, com 6 ocorrências.
  Causa de o lint falhar em lotes: o golangci-lint corta a saída por padrão (`max-same-issues`),
  então o CI não mostrava todas. Rodando localmente com `--max-issues-per-linter=0 --max-same-issues=0`
  apareceram mais 5. As 11 foram corrigidas sem mudar comportamento:
  - `internal/app/calculadora_tela_test.go` (linha 113): `_ = resp.Body.Close()`
  - `internal/app/documentos_test.go` (linha 62), `internal/app/harness_test.go` (linha 135), `internal/app/helpers_test.go` (linha 60),
    `internal/app/tarefas_test.go` (linha 54), `internal/cep/cliente.go` (linha 126): `defer func() { _ = resp.Body.Close() }()`
  - `internal/intimacoes/djen/cliente.go` (linha 61), `internal/monitor/datajud/cliente.go` (linha 66): `!(a && b)` → `(!a || !b)`
  - `internal/platform/config/config.go` (linha 105): `!(dev && http)` → `(!dev || u.Scheme != "http")`
  - `internal/prazos/calculo/corridos.go` (linha 108): dois `!(a && b)` → `(!a || !b)`
  - `internal/processos/duplicado.go` (linha 60): `!(a && b && c)` → `!a || !b || just != atualJust`
- Prova local (go1.26.10, golangci-lint v2.5.0 compilado com go1.26.10, Postgres 16):
  gofmt limpo; `go vet ./...` limpo; golangci-lint sem limite de saída: 0 issues;
  `go test -count=1 -p 1 ./...` com `PAINEL_TEST_DB`: tudo ok (`internal/app`: 108 pass, 2 skip manuais, 0 fail).

- Prova local completa (05/11/2026, `3a0b525` + correção de referências neste arquivo; GOTOOLCHAIN=go1.26.10,
  Postgres 16 local com `PAINEL_TEST_DB`):
  - `test -z "$(gofmt -l .)"`: status 0, sem saída.
  - `go vet ./...`: status 0, sem saída.
  - `go test -race -count=1 -p 1 ./...`: status 0, todos os pacotes `ok` (`internal/app` 204 s).
  - `scripts/check-lines.sh`: status 0 ("nenhum arquivo de produção acima de 300 linhas").
  - `scripts/check-links.sh`: status 0 (47 arquivos, 21 exceções, 0 quebradas). Na primeira execução falhou com 11
    referências quebradas criadas por este arquivo (`caminho:linha` em crase); corrigido para `caminho` (linha N).
  - gosec v2.30.0 e staticcheck v0.8.2 rodados no host, nas versões fixadas do script: staticcheck sem achados;
    gosec 12 achados, todos suprimidos com justificativa, 0 abertos.

## Não verificado
- `scripts/analise-estatica.sh`: status 2. O contêiner `golang:1.26.10-bookworm` não confia no certificado do proxy
  de rede deste ambiente (x509 ao baixar o gosec); o relatório `ANALISE-ESTATICA.md` não foi regravado.
  Equivalente parcial no host acima.
- govulncheck v1.9.0: não rodou; `vuln.go.dev` bloqueado pela política de rede (403).
- gitleaks: não instalado neste ambiente; não rodou. O CI tem o passo "Segredos no Git".
- Testes locais usam Postgres 16; o CI usa 17.
- SBOM e trivy: pendentes. As actions precisam de SHA fixado, e não consegui consultar.

## Ponto de parada
Prova local completa feita; esta atualização do PROGRESSO está sem commit e sem push, por instrução do advogado.
Próximo passo: decidir o push desta atualização.

## Último commit verificado
Anterior: `fe1e205` (CI run 35) falhou no lint do job `go` (1 de 4 jobs com falha).
`df1f679` (CI run 36) e `3a0b525` (CI run 37): CI verde, conclusão `success`, conferido em 05/11/2026 03:14 UTC.
