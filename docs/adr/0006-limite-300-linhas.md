# ADR 0006 — Arquivos de produção com até 300 linhas

**Status:** aceita (11/2026)

**Decisão:** `.go` (exceto `_test.go`), `.html`, `.css` e `.sql` de produção não passam de 300 linhas.
`scripts/check-lines.sh` roda no CI e falha o build. Arquivo maior é dividido por responsabilidade.

**Por quê:** revisão de diff viável por uma pessoa e por IA sem leitura parcial; força coesão.

**Exceções:** nenhuma para código de produção. Arquivos gerados (`sqlc`) ficam em diretório marcado e
fora da checagem, sem edição manual.
