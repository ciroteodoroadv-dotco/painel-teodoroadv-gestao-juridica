# ADR 0006 — Arquivos de produção com até 300 linhas

**Status:** aceita (11/2026)

**Decisão:** `.go` (exceto `_test.go`), `.html`, `.css` e `.sql` de produção não passam de 300 linhas.
`scripts/check-lines.sh` roda no CI e falha o build. Arquivo maior é dividido por responsabilidade.

**Por quê:** revisão de diff viável por uma pessoa e por IA sem leitura parcial; força coesão.

**Exceções:** nenhuma para código de produção. Arquivos gerados (`sqlc`) ficam em diretório marcado e
fora da checagem, sem edição manual.

**Nota de 04/11/2026 (Plano 0):** `referencia/painel-b/` é cópia verificada (SHA-256 contra `docs/plano-fusao/HASHES-B.txt`) do painel B, usada só como referência visual. Não é código de produção, não é embutida no binário e fica fora da checagem de linhas (`scripts/check-lines.sh`).
