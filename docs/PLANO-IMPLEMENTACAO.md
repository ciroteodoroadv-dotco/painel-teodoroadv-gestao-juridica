# Plano de implementação — Painel Teodoro v2

Cada marco só começa quando o anterior cumpriu o aceite. Todo recurso nasce com critério de aceite e teste.

## M0 — Inventário e decisões humanas (o advogado preenche)

- [ ] Preencher `docs/lgpd/` (ROPA, retenção, incidente, titular).
- [ ] Definir responsáveis, perfis e quem é administrador.
- [ ] Criar repositório remoto privado; guardar o ZIP original só para consulta.
- [ ] Conta Oracle (região São Paulo ou Vinhedo); conta Cloudflare; arquivar DPA do Cloudflare.
- [ ] Não migrar dado real antes de cópia validada e plano de retorno.

**Aceite:** inventário aprovado pelo advogado; `gitleaks` limpo.

## M1 — Fundação

- [x] Esqueleto Go, `internal/platform`, config que falha sem segredo.
- [x] Docker Compose (app, PostgreSQL), contêineres sem root. (`cloudflared` entra no M5.)
- [x] `/healthz`, `/readyz`, logs JSON sem conteúdo de cliente.
- [x] CI básico (gofmt, vet, testes, check-lines, govulncheck, gitleaks, build Docker) e `scripts/check-lines.sh`.
- [ ] CI restante: staticcheck, golangci-lint, SBOM, trivy, migração up→down→up (com M2/M3).
- [x] Runbook em linguagem simples: `docs/operacao/RUNBOOK.md`.

**Aceite:** `docker compose up` em máquina nova pelo runbook; CI verde.

**Verificado em 01/11/2026:** `docker compose up` sobe app e db saudáveis; `/healthz` e `/readyz` = 200; app sem root, sistema de arquivos somente leitura, caps removidas.
