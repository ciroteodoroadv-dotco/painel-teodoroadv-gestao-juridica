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

**Pendente de verificação:** `docker compose up` e o build da imagem não foram executados (Docker Desktop
desligado); só a sintaxe do Compose foi validada. A primeira execução do CI é a primeira prova do build.

## M2 — Identidade, autorização e auditoria

- [ ] Validação do JWT do Cloudflare Access.
- [ ] Usuários, Argon2id, rate limit, bloqueio progressivo.
- [ ] 2FA (TOTP e WebAuthn) e códigos de recuperação.
- [ ] Sessão, CSRF, cabeçalhos de segurança.
- [ ] RBAC com negar por padrão.
- [ ] Auditoria encadeada e `painel audit verify`.

**Aceite:** sem papel → 403; POST sem CSRF → 403; toda escrita identifica o ator; `audit verify` ok.

## M3 — Núcleo funcional (dados fictícios)

- [ ] Pessoas, Processos e Tarefas, um por vez, com importação reversível.
- [ ] Cifra de campos sensíveis e índice cego.
- [ ] Concorrência otimista; transações nas operações multi-entidade.
- [ ] UX: busca sem perder foco, aviso de formulário alterado, teclado, mensagens claras.

**Aceite:** por domínio, testes unitários, integração e E2E; falha simulada no meio da operação não deixa
duplicata nem estado parcial.

## M4 — Agenda, audiências e documentos

- [ ] Agenda e Audiências.
- [ ] Upload com limite, tipo por bytes mágicos, nome aleatório, volume privado.
- [ ] Leitura de documento auditada.

**Aceite:** upload com tipo falso é recusado; arquivo inacessível sem permissão.

## M5 — Infraestrutura Oracle

- [ ] VM, Tunnel, política do Access, firewall (sem porta de entrada), SSH por chave via Bastion.
- [ ] Backup diário cifrado, retenção 7/4/12, cópia fora da Oracle.
- [ ] Restauração mensal em ambiente isolado.
- [ ] Pipeline dev → homologação → produção por digest, com rollback.

**Aceite:** restauração fictícia funciona; varredura externa não encontra porta aberta; deploy de teste
com rollback demonstrado.

## M6 — Jobs e integrações cadastradas

- [ ] Fila com `FOR UPDATE SKIP LOCKED`, idempotência, timeout, retentativa limitada, status visível.
- [ ] Tabela `integrations` com tudo desligado e aceite auditado ao ligar.

**Aceite:** job interrompido não duplica; integração só liga com aceite registrado.

**Primeiro marco realista = M0 a M5.** WhatsApp, e-mail, DataJud e DJEN ficam para depois, cada um com
revisão própria.

## Matriz de rastreabilidade (plano-base → onde está)

| Item do plano-base | Onde |
|---|---|
| Decisão 1 — sem acesso público | Espec. §2; ADR 0002; M5 |
| Decisão 2 — usuário individual, papéis | Espec. §5; M2 |
| Decisão 3 — PostgreSQL fonte transacional | Espec. §1, §8; ADR 0004 |
| Decisão 4 — 300 linhas | Espec. §12; ADR 0006; M1 |
| Decisão 5 — automação não decide prazo | Espec. §8, §13; M6 |
| Decisão 6 — integrações desligadas | Espec. §7, §13; M6 |
| Decisão 7 — sem modo irrestrito | Regra de trabalho; README |
| Fase 0 | M0 |
| Fase 1 | M1 e M5 |
| Fase 2 | M2; ADR 0003, 0005 |
| Fase 3 | M3 e M4 |
| Fase 4 | M6 |
| Fase 5 | Espec. §12; M1, M5 |
| Regras permanentes | README; `.gitignore`; Espec. §7 |
