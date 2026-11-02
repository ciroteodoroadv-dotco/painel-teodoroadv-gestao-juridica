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

**Verificado em 01/11/2026:** `docker compose up` sobe app e db saudáveis; `/healthz` e `/readyz` = 200;
app sem root, sistema de arquivos somente leitura, capacidades removidas. O CI ainda não rodou (sem remoto).

## M2 — Identidade, autorização e auditoria

Parte 1 (implementada em 01/11/2026):

- [x] Validação do JWT do Cloudflare Access (`internal/platform/cfaccess`); produção recusa iniciar sem ele.
- [x] Usuários, Argon2id (m=64MiB, t=3, p=2), rate limit por IP, bloqueio progressivo (1/5/30 min, depois admin).
- [x] 2FA por TOTP com anti-reuso do código e 8 códigos de recuperação de uso único.
- [x] Sessão (cookie `__Host-`, ociosidade 30 min, absoluta 10 h, rotação ao completar 2FA), CSRF, cabeçalhos.
- [x] RBAC com negar por padrão: rota sem permissão derruba a inicialização.
- [x] Auditoria encadeada (SHA-256), gatilho contra UPDATE/DELETE/TRUNCATE e `painel audit verify`.
- [x] Migrações goose com `down` (up→down→up exercitado nos testes de integração).

Parte 2 (implementada em 01/11/2026):

- [x] **WebAuthn / chave física** (passkey): cadastro como primeiro fator, login, remoção, contador anti-clonagem.
  Testado de ponta a ponta com um autenticador de software nos testes; **NÃO testado com chave física real**.
- [x] Papel de banco `painel_app` sem poder de dono: na auditoria só INSERT/SELECT (prova no teste `papel_test.go`).
- [x] Lista de ~30 mil senhas vazadas (SecLists, 12+ caracteres) e troca de senha pelo próprio usuário.
- [x] Senha provisória: criada/redefinida por administrador obriga a troca no primeiro acesso.
- [x] Tela de administração de usuários (criar, papel, ativar/desativar, desbloquear, redefinir senha).
- [x] Auditados: revelar documento de pessoa, relatório do titular, exclusões (M3). Leitura de arquivo e exportação entram no M4.

**Aceite:** sem papel → 403; POST sem CSRF → 403; toda escrita identifica o ator; `audit verify` ok.
**Verificado:** testes de integração cobrem os quatro (`internal/auth/flow_test.go`); `audit verify` rodado na pilha real.

## M3 — Núcleo funcional (dados fictícios)

- [x] Pessoas, Processos (com partes) e Tarefas, cada um com handler, serviço e repositório próprios.
- [x] CPF/CNPJ cifrado (AES-256-GCM, subchave própria) + índice cego HMAC; mostrado mascarado; revelar é auditado.
- [x] Número CNJ validado pelo dígito; CPF/CNPJ validados; vencimento sempre informado pelo advogado.
- [x] Concorrência otimista (`version`) e transação única (escrita + auditoria) em toda operação.
- [x] Exclusão por marcação (nada é apagado); pessoa vinculada a processo não pode ser excluída.
- [x] Relatório "dados desta pessoa" (pedido do titular, LGPD art. 18) auditado.
- [x] UX: busca ao vivo (htmx local) que mantém o foco, confirmação em dois cliques, aviso de formulário
  alterado, navegação por teclado, link "pular para o conteúdo".
- [x] Fronteiras: teste de importação e de SQL impedem um domínio de tocar outro (`fronteiras_test.go`).
- [x] `painel seed-demo` (só em modo dev) cria dados fictícios.
- [ ] **Importação reversível** de cadastros existentes: não implementada (não há dado real a migrar ainda).
- [ ] **Expurgo definitivo** após janela de confirmação: não implementado (hoje só marcação).
- [ ] **E2E em navegador automatizado** (Playwright): não feito. Os "E2E" são testes HTTP completos
  (sessão, CSRF, banco real). Uma verificação manual em navegador real foi feita na busca e nas telas principais.

**Aceite:** por domínio, testes unitários, integração e HTTP ponta a ponta; falha simulada no meio da operação
não deixa duplicata nem estado parcial. **Verificado:** `internal/app/*_test.go` (inclui falha simulada trocando
o nome da tabela de auditoria no meio da operação: nenhuma parte fica gravada).

## M4 — Agenda, audiências e documentos

- [x] Agenda e Audiências (audiência, reunião, diligência): horário digitado em Brasília e gravado em UTC;
  processo opcional, responsável, participantes (pessoas); tela "Hoje" mostra os próximos 7 dias.
- [x] Upload: limite de 25 MiB (413 acima), tipo pelo **conteúdo** (PDF, PNG, JPEG, DOCX com estrutura conferida),
  extensão tem de combinar, nome do arquivo reduzido a nome simples, vínculo obrigatório a processo ou pessoa.
- [x] Arquivo cifrado em disco (AES-256-GCM, subchave própria), nome aleatório, volume privado (0600, dono sem privilégios),
  fora de qualquer pasta pública; integridade conferida (SHA-256 + autenticação do GCM) a cada leitura.
- [x] Leitura de documento auditada **antes** da entrega (sem auditoria, sem arquivo); download sempre como anexo,
  `nosniff` e CSP `sandbox`. Envio e exclusão auditados. Envio que falha no banco remove o arquivo.
- [x] Permissões: todos leem; só advogado envia/exclui; assistente escreve na agenda (e em tarefas).

**Limites conhecidos (M4):** sem antivírus (ClamAV é item futuro); PDF pode conter script — por isso é entregue só
como anexo; exclusão apenas oculta (o arquivo continua no volume até um expurgo decidido pelo escritório);
arquivo órfão é possível se o processo cair entre gravar o arquivo e o registro (varredura de órfãos: futuro);
DOCX é conferido só pela estrutura (não é descompactado nem analisado).

**Aceite:** upload com tipo falso é recusado; arquivo inacessível sem permissão. **Verificado:** `internal/app/documentos_test.go`
e `agenda_test.go` (inclui falha simulada no meio e adulteração do arquivo em disco) e, na pilha Docker real
(raiz somente leitura, usuário sem root): envio, arquivo cifrado no volume, download e auditoria.

## M5 — Infraestrutura Oracle

Parte local (feita em 02/11/2026, **testada com dados fictícios**):

- [x] Backup cifrado com age (banco + documentos + manifesto), permissão 0600, SHA-256 do arquivo; cópia externa
  por `rclone` (configurável); aviso quando não há cópia externa. ADR 0007.
- [x] Retenção 7 diários / 4 semanais / 12 mensais (`prune.sh`, testada com 400 arquivos fictícios).
- [x] Restauração isolada (rede sem internet) + `painel restore-check` (cadeia de auditoria e **todos** os documentos
  decifrados e conferidos). Provado: chave errada recusada, backup adulterado recusado, backup com documento ausente reprovado.
- [x] Compose de produção: nenhuma porta publicada, `cloudflared` só de saída, app e túnel endurecidos, logs com rotação.
- [x] Modo produção do app: sem JWT do Cloudflare Access recusa iniciar; `/login`, `/static` e demais rotas → 403
  sem JWT; `/healthz` só interno (403 se vier pelo Cloudflare). Verificado em contêiner real.
- [x] `deploy.sh`: backup antes, exige imagem por digest, espera saúde, **rollback automático** (provado com imagem quebrada).
- [x] `RUNBOOK-ORACLE.md`, `cloud-init.yaml`, `bootstrap.sh`, firewall nftables (sintaxe validada), timers systemd,
  `status.sh`, `release.yml` (imagem por digest, SBOM, trivy) e jobs de CI para os scripts.

Pendente — **depende das contas do M0 e de uma VM real**:

- [ ] Criar VCN privada, NAT, Bastion, VM, bucket e usuário de backup na Oracle.
- [ ] Criar túnel e política do Access no Cloudflare; testar a validação do JWT contra o Cloudflare real.
- [ ] Executar `cloud-init`, `bootstrap.sh`, timers e firewall na VM (nunca rodaram numa VM real).
- [ ] Testar a cópia externa (Object Storage + segunda cópia) e a restauração com a chave offline.
- [ ] Rodar `release.yml` e o CI num repositório remoto (ações e imagens já fixadas por hash/digest; Dependabot configurado).
- [ ] Varredura externa de portas e teste de desligar/religar o túnel.

**Aceite:** restauração fictícia funciona ✔ (local); varredura externa não encontra porta aberta ☐ (precisa da VM); deploy
de teste com rollback demonstrado ✔ (local).

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
