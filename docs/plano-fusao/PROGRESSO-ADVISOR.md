# Progresso do advisor (/painel)

Atualizado em 04/11/2026. Branch: `feat/fase-1-saneamento` (base: `origin/feat/advisor-command`).

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

## Não verificado
- Testes Go e lint não rodaram neste ambiente: o toolchain pede go1.26.10 e os módulos não baixam (sem acesso ao proxy).
  Evidência de testes = CI após push.
- SBOM e trivy: pendentes. As actions precisam de SHA fixado, e não consegui consultar.
- `scripts/analise-estatica.sh`: não executado (mesma causa dos testes Go).

## Ponto de parada
Commits locais prontos. Nada foi enviado ao remoto. Próximo passo: autorização do advogado para push, depois observar o CI do branch.

## Último commit verificado
Nenhum commit verificado por CI ainda.
