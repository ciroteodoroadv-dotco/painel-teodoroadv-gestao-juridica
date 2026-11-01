# ADR 0005 — Auditoria append-only com hash encadeado

**Status:** aceita (11/2026)

**Decisão:** `audit_events` só aceita `INSERT`/`SELECT` para o papel do app; gatilho bloqueia `UPDATE` e
`DELETE`. Cada evento guarda `prev_hash` e `hash` (SHA-256). `painel audit verify` recalcula a cadeia.

**Limite honesto [Provável]:** quem tem acesso de superusuário ao banco ainda pode reescrever a cadeia
inteira. Mitigação: exportar o último hash periodicamente para o backup cifrado fora da Oracle.

**Conteúdo:** ator, ação, alvo, resultado, `request_id`; nunca senha, token, CPF ou texto integral.
