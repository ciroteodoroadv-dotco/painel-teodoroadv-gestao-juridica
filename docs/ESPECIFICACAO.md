# Especificação técnica — Painel Teodoro v2

Base: `PLANO-REESCRITA-PAINEL-TEODORO-LINUX-LGPD.md` (auditoria de 31/10/2026). Nesta etapa não há código,
dados reais, segredos, WhatsApp, e-mail, DataJud nem DJEN.

> LGPD não é selo obtido por código. O sistema fornece controles e evidências técnicas; a conformidade
> também depende de inventário, bases legais, retenção, contratos, resposta a incidentes e revisão jurídica.

## 1. Decisões fixadas

| Tema | Decisão |
|---|---|
| Frontend | HTML no servidor (Go `html/template`) + htmx servido localmente, sem CDN |
| Acesso | Cloudflare Access + Cloudflare Tunnel; nenhuma porta de entrada aberta na VM |
| 2FA | Obrigatório no app: chave física (WebAuthn) ou autenticador (TOTP) |
| Banco | PostgreSQL, única fonte transacional |
| Backend | Go modular, um só binário |

**LGPD [Provável]:** o Cloudflare termina o TLS na borda e vê o conteúdo em trânsito. Passa a ser operador
(LGPD art. 5º, VII) e há transferência internacional (art. 33). Exige DPA arquivado e registro no ROPA.
Alternativa sem terceiro (WireGuard/Headscale) está no ADR 0002 como rejeitada, para revisão futura.

## 2. Arquitetura

Navegador → Cloudflare Access (política por e-mail autorizado) → Tunnel → `cloudflared` na VM →
app Go (porta só na rede interna do Docker) → PostgreSQL (sem porta publicada) + volume privado de documentos.

- O app valida o JWT `Cf-Access-Jwt-Assertion` em toda requisição e recusa o que não veio pelo Access.
- VM Oracle em São Paulo ou Vinhedo (dados em repouso no Brasil).
- SSH só por chave, via OCI Bastion ou Cloudflare Access for Infrastructure; porta 22 fechada à internet.

## 3. Pilha

Go estável (versão fixada na M1, lançada há mais de 2 semanas) · `net/http` com roteamento por padrão ·
`html/template` · htmx fixado e local · `pgx` + `sqlc` · `goose` (migrações com `down`) ·
`x/crypto/argon2` · `pquerna/otp` · `go-webauthn/webauthn` · `log/slog` (JSON) · Docker Compose.
Sem Caddy: o túnel faz o TLS externo.

## 4. Estrutura do repositório

```
cmd/painel/            main: só monta dependências
internal/platform/     config, db, http, logs, health, csrf, cfaccess
internal/auth/         login, sessão, 2FA, RBAC
internal/audit/        eventos append-only
internal/pessoas|processos|tarefas|agenda|documentos/
                       handler.go, service.go, repo.go, queries.sql, *_test.go, templates/
internal/jobs/         fila com lock e idempotência (M6)
migrations/            SQL versionado
web/static/            css, htmx.min.js, ícones
deploy/                compose, runbook, scripts de backup e restauração
docs/                  especificação, plano, ADRs, lgpd/
scripts/check-lines.sh limite de 300 linhas
```

Fronteira: um domínio só consulta a própria tabela. Acesso cruzado passa por interface do `service`
(ex.: `tarefas` recebe `ProcessoLookup`). Verificado no CI por teste de importação.

## 5. Identidade, sessão e RBAC

- Um usuário por pessoa; o e-mail do usuário deve coincidir com o do JWT do Access.
- Senha Argon2id (m=64 MiB, t=3, p=2; parâmetros gravados no hash). Mínimo 12 caracteres e checagem
  contra lista de senhas vazadas offline.
- 2FA obrigatório a todos os papéis; códigos de recuperação de uso único (guardados com hash).
- Rate limit por IP e por conta; bloqueio progressivo 1 min, 5 min, 30 min, depois só administrador libera.
- Sessão na tabela `sessions` (id de 256 bits, só o hash no banco); cookie `__Host-`, `Secure`, `HttpOnly`,
  `SameSite=Strict`; ociosidade 30 min, duração máxima 10 h; logout e troca de senha revogam todas.
- CSRF: token por sessão em todo método não-GET (htmx envia por header).
- Papéis: **administrador** (usuários, integrações, auditoria); **advogado** (leitura e escrita nos domínios);
  **assistente** (leitura; escrita só em tarefas e agenda).
- Matriz módulo × ação em código, negar por padrão. Rota sem permissão declarada faz o teste falhar.
- Cabeçalhos: CSP estrita sem `unsafe-inline`, HSTS, `X-Content-Type-Options`, `Referrer-Policy`,
  `frame-ancestors 'none'`.

## 6. Auditoria

Tabela `audit_events`: id, momento, ator, ação, tipo e id do alvo, resultado, `request_id`, metadados em lista
permitida (nunca senha, token, CPF ou texto integral), `prev_hash` e `hash` (SHA-256 encadeado).

- O papel de banco do app tem só `INSERT` e `SELECT`; gatilho bloqueia `UPDATE` e `DELETE`.
- `painel audit verify` recalcula a cadeia.
- Auditados: login, falha de login, mudança de papel, leitura de documento, exportação, exclusão.

## 7. LGPD por desenho

- Classificação por coluna (comum, pessoal, sensível) documentada no esquema.
- CPF e campos sensíveis cifrados na aplicação (AES-256-GCM) + índice cego HMAC para busca.
- Logs só com ids e `request_id`; nunca conteúdo de cliente.
- Exclusão em dois tempos: marcar → janela de confirmação → expurgo auditado, com backup anterior.
- Relatório "dados desta pessoa" (atendimento a titular) e prazo de retenção por registro.
- Tabela `integrations` com tudo `enabled=false`; ligar exige administrador, texto de aceite e evento de
  auditoria. Nenhum conector é implementado nesta fase.

## 8. Domínios

- **Pessoas:** tipo PF/PJ, nome, documento cifrado, contatos, papel (cliente, parte contrária, testemunha).
- **Processos:** número CNJ validado por dígito, órgão, partes (ligação com pessoas), situação, responsável.
- **Tarefas:** título, processo opcional, responsável, vencimento informado pelo advogado (nada calculado),
  status, origem (`manual` ou `automacao_a_conferir`).
- **Agenda/Audiências (M4):** data, local ou link, processo, participantes.

Toda tabela tem `created_at/by`, `updated_at/by` e `version` (concorrência otimista: edição simultânea
não sobrescreve em silêncio). Operação multi-entidade roda em transação PostgreSQL.

## 9. Interface

Menu lateral: Hoje · Pessoas · Processos · Tarefas · Agenda · Administração. Tela "Hoje" com tarefas
vencendo e audiências. Busca htmx que não perde o foco. Aviso ao sair de formulário alterado. Navegação
completa por teclado. WCAG 2.2 AA. Tema claro e escuro. Erros em português simples. Funciona no celular.
Visual limpo, no estilo de sistemas jurídicos web, sem copiar marca de terceiros.

## 10. Documentos (M4)

Corpo de requisição até 1 MiB; upload até 25 MiB. Tipo real por bytes mágicos (PDF, PNG, JPEG, DOCX).
Nome aleatório. Volume fora de `web/`. Entrega só por handler autorizado e auditado.

## 11. Operação

- Config por variáveis de ambiente ou segredos do Docker. O app **não inicia** sem segredo de sessão,
  chave de cifra, senha do banco ou audiência do Access.
- `/healthz` (vivo) e `/readyz` (banco ok), só na rede interna.
- Backup diário: `pg_dump` + documentos, cifrado com `age`, para OCI Object Storage privado (região BR)
  e cópia fora da Oracle. Retenção 7 diários, 4 semanais, 12 mensais.
- Restauração mensal em contêiner isolado; o script confere contagens e `audit verify`. No dev, dados fictícios.
- Contêineres sem root, imagem distroless, sistema de arquivos somente leitura.

## 12. CI/CD

GitHub Actions em repositório privado: `gofmt`, `go vet`, `staticcheck`, `golangci-lint`, `govulncheck`,
`gitleaks`, `scripts/check-lines.sh` (todo arquivo de produção até 300 linhas: `.go` exceto `_test`,
`.html`, `.css`, `.sql`), testes unitários, integração com PostgreSQL, migração up→down→up, E2E, SBOM
(`syft`) e varredura de imagem (`trivy`).

Imagem identificada por digest; promoção dev → homologação → produção pelo mesmo digest. Deploy por script:
backup → migrar → healthcheck → rollback automático se falhar. Nenhuma edição manual em produção.

## 13. Fora de escopo (marcos posteriores)

Migração de dados reais, WhatsApp (exige DPIA), e-mail, DataJud, DJEN. Prazo processual nunca é calculado
automaticamente; dados externos entram como "pendente de conferência pelo advogado".

## 14. Adendo — Fusão com o painel B (novembro de 2026)

O frontend do painel B prevaleceu (forma) e o v2 manteve a regra de segurança. O que mudou em relação às seções acima (menu, telas, telas novas de Atendimentos, Audiências, Diligências, Intimações,
Monitor do DataJud, Calculadora de prazo, Calendário forense, Índices e CEP) está em `docs/plano-fusao/` (`ESPECIFICACAO-FUSAO.md`, `00-SEQUENCIA.md`, `NOTAS-PLANO-*.md`, `PENDENCIAS.md`)
e nos ADRs 0009 a 0012. Em especial, o item "Fora de escopo" da seção 13 foi superado: DJEN, DataJud, índices e CEP existem, **desligados por padrão** (ADR 0011).
