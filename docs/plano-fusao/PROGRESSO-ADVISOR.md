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
  - trivy e SBOM no job `docker` do CI (05/11/2026), sobre a imagem `painel:ci` já construída ali.
    SHAs consultados com `git ls-remote` (o `gh` deste ambiente só faz busca) e conferidos em clone dos repositórios:
    cada tag aponta para commit do histórico do branch principal. `aquasecurity/trivy-action` v0.36.0 → `ed142fd0673e97e23eac54620cfb913e5ce36c25`
    (22/04/2026); `anchore/sbom-action` v0.24.3 → `66cbf4bc1f1c0d2edc94016e65bc221b6bb0ad6c` (02/10/2026);
    `golangci/golangci-lint-action` v9.3.0 → `ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a` (consultado, não usado: o lint segue por `go install`).
    YAML validado com PyYAML. `release.yml` usava o SHA do objeto da tag do trivy; corrigido para o commit `ed142fd…`.
  - Prova real local de trivy e SBOM (05/11/2026), na imagem `painel:ci` construída com `deploy/Dockerfile`
    (cópia temporária com o certificado do proxy só no estágio de build; imagem final igual):
    - trivy v0.70.0 (padrão da ação; compilado com `go install`): 0 vulnerabilidades CRÍTICA/ALTA com correção, status 0.
      Teste negativo: `alpine:3.10` → 1 CRÍTICA, status 1.
    - syft v1.54.0 (padrão da ação): SPDX 2.3, 30 pacotes; os 22 módulos Go são idênticos aos de `go version -m` do binário.

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

- PR #13 (05/11/2026): a pasta de skills em .claude (115 arquivos de skills de terceiros) removida do repositório — as skills são
  ferramenta do advisor, não código do painel (decisão do advogado). Nunca entraram no binário/imagem (Dockerfile copia só
  cmd, internal, migrations, web). Regra gravada no `/painel`. A exceção do `.gitleaks.toml` fica: o arquivo com a chave de
  exemplo da AWS continua no histórico do PR. gitleaks local nos 24 commits: "no leaks found".

## Não verificado
- `scripts/analise-estatica.sh`: status 2. O contêiner `golang:1.26.10-bookworm` não confia no certificado do proxy
  de rede deste ambiente (x509 ao baixar o gosec); o relatório `ANALISE-ESTATICA.md` não foi regravado.
  Equivalente parcial no host acima.
- govulncheck v1.9.0: não rodou; `vuln.go.dev` bloqueado pela política de rede (403).
- gitleaks: não instalado neste ambiente; não rodou. O CI tem o passo "Segredos no Git".
- Testes locais usam Postgres 16; o CI usa 17.

## Ponto de parada
**Pronto para revisão** (05/11/2026): etapas 0 e M1-CI completas.
- Etapa 0: 22 arquivos de `referencia/painel-b/` conferidos de novo por `sha256sum -c` contra `HASHES-B.txt`
  (22 OK, nenhum a mais nem a menos); `check-lines` OK; `check-links` 0 quebradas.
- M1-CI: CI verde com os jobs novos (run 40, `c3b34ac`, log conferido); PENDENCIAS.md "Fundação (M1)": 5 de 5 resolvidos.
- PR #13: o primeiro CI em evento `pull_request` falhou só no gitleaks (403 ao listar os commits do PR: o token tinha
  apenas `contents: read`). Corrigido com `pull-requests: read` no job `go`. Antes do push, gitleaks 8.24.3 local nos
  23 commits do PR (`origin/main..HEAD`): "no leaks found", status 0.
- Ressalva: o run 41 (`a65f904`, só documentação) estava em andamento na conferência; o PR só deve ser aberto com ele verde.
- O branch inclui os 7 commits de `feat/advisor-command`, que ainda não estão em `main`.
Próximo passo: autorização do advogado para push e abertura do PR para revisão.

## Último commit verificado
`c3b34ac` (CI run 40, 05/11/2026): conclusão `success` nos 4 jobs. Log conferido:
- job `docker`: trivy → `painel:ci (debian 12.15)` 0 e `painel` (gobinary) 0, igual à prova local;
  SBOM com syft v1.54.0 → artefato `painel_ci.spdx.json` publicado.
- job `go`: lint "0 issues"; `internal/app` 176 s (integração rodou); 300 linhas OK; referências 0 quebradas;
  gosec alta=0 média=0, staticcheck=0, govulncheck rc=0; gitleaks "no leaks found".
Anteriores: run 39 (`25dca67`) e run 36 (`df1f679`) verdes com log conferido; run 35 (`fe1e205`) falhou no lint.
