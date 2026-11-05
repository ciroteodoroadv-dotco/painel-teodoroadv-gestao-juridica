# Auditoria técnica — 05/11/2026

**Alvo:** commit `c938855`. Método: leitura de código, `go vet`, `staticcheck v0.8.2`, `gosec v2.30.0`, suíte inteira com
`-race` e PostgreSQL real, e testes de prova escritos para os achados (hoje em `internal/app/auditoria_1105_test.go`).
`govulncheck` não rodou nesta sessão (rede sem acesso ao banco de vulnerabilidades).

Gravidade: P0 bloqueante, P1 alta, P2 média, P3 baixa, P4 melhoria. Estado do achado: **C** confirmado (teste ou
leitura inequívoca), **FI** fortemente indicado, **H** hipótese.

## Achados e situação da correção

| # | Achado | Grav. | Estado | Correção |
|---|---|---|---|---|
| 1.1 | PDF da tarefa recusado acima de ~1 MiB com a mensagem "o limite é 25 MiB" (teto de 1 MiB do `httpx` fora de `/documentos`) | P1 | C (teste) | **Fase 1**: limite por rota só no `auth.Router` (`HandleUpload`); `httpx` só aplica teto geral; semáforo de 3 envios simultâneos |
| 1.2 | Prazo "a conferir" virava "confirmado" ao advogado salvar a tarefa ou o prazo; prazo digitado por advogado nascia confirmado, contra o ADR 0012 item 3 | P1 | C (teste) | **Fase 1**: nenhuma escrita confirma; confirmar é só o ato explícito (evento `prazo.confirmado`); mudar a data derruba a confirmação, seja quem for |
| 1.3 | Prazo fatal da tarefa gravado fora da transação da tarefa | P2 | C | **Fase 1**: `PrazoSync` só tem variantes `...Tx`; tarefa e prazo na mesma transação |
| 7.1 | Contratos entre domínios sem a unidade de trabalho (causa-raiz de 1.3) | P2 | C | **Fase 1** (idem) |
| 6.1 | Suíte não cobria 1.1 e 1.2 | P3 | C | **Fase 1**: testes de regressão |
| 2.1 | Senha do DONO do banco montada no contêiner do app | P2 | C | **Fase 2** (feito) |
| 3.4 | Metadados de auditoria descartados em silêncio (`partes`, `abertos_vinculados`, `quantidade`) | P3 | C (teste) | **Fase 2** (feito) |
| 3.5 | Gestão de usuários traduz qualquer erro em 404 | P3 | C | **Fase 2** (feito) |
| 2.5 | Login de conta bloqueada responde mais rápido (oráculo de tempo) | P3 | C | **Fase 2** (feito) |
| 3.1 | JWKS do Access: falha na renovação horária derruba tudo; busca com mutex travado | P2 | C | **Fase 3** (feito) |
| 3.2 | Coletas em segundo plano sem `recover` | P2 | H | **Fase 3** (feito) |
| 1.5 | "Executar coleta agora" síncrono na requisição | P2 | FI | **Fase 3** (feito) |
| 4.1 | Backup monta tudo num tmpfs de 2 GiB | P2 | FI | **Fase 4** (feito) |
| 3.3 | Falha de backup sem alerta | P2 | C | **Fase 4** (feito) |
| 5.1 | Imagem de backup por tag mutável, fora do release | P3 | C | **Fase 4** (feito) |
| 2.2 | Ações de tomada de conta sem reautenticação | P2 | C | **Fase 5** (feito) |
| 2.3 | Separação de funções do administrador é nominal | P3 | FI | **Fase 5** (feito) |
| 2.8 | Limitador por IP único para login e 2FA | P3 | H | **Fase 5** (feito) |
| 1.4 | Listas de pessoas/processos cortadas em 2.000 sem aviso; seletor de vínculo herda o corte | P2 | FI | **Fase 6** (feito) |
| 4.2 | Tela de auditoria recalcula a cadeia inteira a cada acesso | P3 | C | **Fase 6** (feito) |
| 4.4 | Sessões expiradas nunca apagadas | P3 | C | **Fase 6** (feito) |
| 4.5 | Listas de tarefas/agenda/audiências/intimações sem limite | P3 | C | mantido de propósito (ver Fase 6) |
| 1.6 | Regra "A confirmar" checada sobre leitura anterior à transação | P3 | H | **Fase 6** (feito) |
| 4.3 | Envio ocupa ~3× o tamanho em memória, sem limite de simultâneos | P3 | H | **Fase 1** (semáforo de 3 envios) |
| 6.2 | README desatualizado | P3 | C | **Fase 6** (feito) |
| 2.4 | Leitura sem escopo por equipe (F13) | P2 | C | marco próprio (decisão do escritório) |
| 2.6 | Hash da auditoria sem chave nem âncora externa | P4 | C | **Fase 7** (âncora no backup) |
| 2.7 | Segredo TOTP com a chave mestra, sem versão de chave | P4 | C | **Fase 7** (feito) |
| 7.2 | `auth.Auth`/`PageData` acumulam responsabilidades | P4 | C | marco próprio |
| 5.2 | Bloqueios de uso real já listados em `PENDENCIAS.md` | P1 | C (doc) | decisão do advogado/administrador |

## Decisão registrada (Fase 1)

Todo prazo nasce "A CONFERIR", inclusive o digitado pelo advogado (na tela de Prazos ou na tarefa). O advogado confirma
pelo botão "Confirmar prazo", que grava quem e quando e deixa `prazo.confirmado` na auditoria. Editar mantendo a data
mantém a situação; mudar a data volta a "A CONFERIR" e apaga quem/quando. Isso é o que o ADR 0012, item 3, já dizia.

## Fase 2 (feita)

- **2.1**: serviço `migrate` no Compose (execução única, `restart: "no"`), o ÚNICO com `db_password`; o `app` depende
  dele (`service_completed_successfully`) e não recebe mais a senha do dono. Sem ela, o servidor só abre o papel
  restrito (`openApp`); com ela (dev ou instalação antiga) migra como antes e, fora do dev, avisa no log.
  `deploy.sh` roda `migrate` antes de subir a nova imagem (falhou = deploy cancelado, versão atual segue no ar).
  Comandos de administração (`user create`, `unlock`, `audit verify`, `seed-demo`) passam a usar o serviço `migrate`
  (RUNBOOK, RUNBOOK-ORACLE, COMECAR-USO-REAL e `comecar-uso-real.sh` atualizados).
  **Não validado com Docker nesta sessão** (sem daemon): rodar `docker compose -f docker-compose.yml config -q`, a
  variante prod do CI e `teste-producao-local.sh` antes do deploy.
- **3.4**: `partes`, `abertos_vinculados` e `quantidade` na lista permitida; teste que varre o código e falha se uma
  chave usada em auditoria estiver fora da lista (verificado: sem a correção, acusa 4 usos).
- **3.5**: 404 só para usuário inexistente ou id inválido; demais falhas = 500 com log (teste vermelho→verde).
- **2.5**: conta bloqueada gasta o mesmo tempo de Argon2 que uma verificação real.

## Fase 3 (feita)

- **3.1**: `cfaccess.key` renova o JWKS fora do mutex, no máximo uma tentativa por minuto, sem bloquear as demais
  requisições; se a renovação falhar, a chave conhecida vale por até 24 h. Teste vermelho no código antigo
  ("não consegui obter as chaves do Access") e verde no novo.
- **3.2**: `coletas.Protegida` converte pânico em erro nas rodadas do DJEN, DataJud e BCB (o valor do pânico não vai
  para o log, só o tipo de erro de execução).
- **1.5**: "Executar coleta agora" (DJEN e DataJud) roda desacoplado da requisição (`coletas.EmSegundoPlano`): se
  terminar em 20 s responde como antes; senão responde "em andamento" e a rodada segue até o fim.

## Fase 4 (feita)

- **4.1**: o dump continua no tmpfs (texto do banco nunca em disco em claro); o `docs.tar` (documentos já cifrados
  pelo app) vai para `BACKUP_DIR/.trabalho.*` em disco, apagado ao fim; sobras com mais de 12 h são limpas. Formato
  do arquivo (`db.dump`, `docs.tar`, `MANIFEST`) igual: `restore.sh` não mudou. **Prova local** com `pg_dump`/`age`
  reais: backup → restauração em banco novo → 32 tabelas e documentos com SHA-256 idênticos.
- **3.3**: `OnFailure=painel-alerta@%n.service` no backup e no teste de restauração; `oracle/alerta.sh` grava no diário
  com prioridade crítica, em `/var/lib/painel/alerta.ultimo` (mostrado por `status.sh`) e, se houver
  `secrets/alerta_webhook` (https), envia por webhook.
- **5.1**: `release.yml` varre e publica `painel-backup` por digest; `deploy.sh` aceita `PAINEL_BACKUP_IMAGE` (digest
  obrigatório), grava em `.estado-deploy.backup`, e o timer de backup e o teste de restauração usam esse digest; o
  PostgreSQL do teste de restauração usa o mesmo digest do Compose.
  **Não executado aqui:** systemd, GitHub Actions e Docker (sem daemon nesta sessão).

## Fase 5 (feita)

- **2.2**: exigem a SUA senha atual (`senha_atual`; conta tentativa e trava como o login): criar usuário, mudar papel,
  redefinir senha ou 2º fator de outra pessoa, regerar códigos de recuperação, remover chave e cadastrar chave
  ADICIONAL (o primeiro fator, logo após a senha, não pede de novo). Formulários e `webauthn.js` atualizados.
  Desativar/reativar e desbloquear não pedem (não dão acesso a ninguém). Testes de recusa (sem senha e senha errada).
  Decisão: só a senha, não o 2º fator — o administrador pode ter apenas chave física, e exigir TOTP o travaria.
- **2.3**: faixa na Agenda dos ADVOGADOS com as alterações de usuário (criação, papel, senha, 2º fator) das últimas
  72 h. Aprovação por um advogado antes da alteração NÃO foi implementada (é decisão do escritório).
- **2.8**: limite de login e 2FA por IP+conta (10/min) somado a um teto por IP (60/min); teste: uma conta no limite não
  trava a colega no mesmo IP.

## Fase 6 (feita)

- **1.4 — agravado e corrigido**: além do corte silencioso da tela, `ResumosVinculo` (rótulo do vínculo das tarefas)
  e o adaptador da agenda resolviam ids pela lista CORTADA: tarefa de processo fora dos 2.000 mais novos ficava sem
  o rótulo e o seletor de vínculo não oferecia o processo (provado no código antigo). Agora: `PorIDs` e
  `ListarTodos`/`ListarTodas` (sem corte) para resolver ids e montar seletores; a tela avisa "Lista limitada" com o
  total real e o Enter no campo de busca consulta o banco inteiro (processos: número, código, título; pessoas: nome ou
  CPF/CNPJ pelo índice cego). Teste com 2.002 processos.
- **4.4**: `ExpurgarSessoes` a cada hora (absoluto e ociosidade); teste.
- **4.2**: a tela de auditoria confere de forma incremental a partir do último elo conferido, e desde o começo a cada
  reinício e a cada 24 h; defeito zera o ponto. A conferência completa continua em `painel audit verify` e no teste
  mensal de restauração. Teste de `VerifyFrom`.
- **1.6**: `travarEstado` (`SELECT … FOR UPDATE`) na transação de `Update` e `Mover`; estado mudado no meio = 409.
- **6.2**: README com o estado real.
- **4.5 — não alterado, de propósito**: pôr limite nessas listas sem paginação repetiria o defeito 1.4 (dado
  escondido). A agenda já filtra por data; as demais crescem com o uso. Paginação real fica para quando houver volume.

## Fase 7 (parcial, de propósito)

- **2.7**: segredo TOTP novo em formato versionado (0x01), com subchave `auth.totp` e o id do usuário como dado
  associado (`crypt.SealAD/OpenAD`); o formato antigo continua sendo lido e é substituído quando o 2º fator é
  redefinido. Teste: abre o novo, abre o legado, recusa a cópia para outro usuário.
- **2.6**: o MANIFEST de todo backup grava `auditoria_ultimo_elo=SEQ:HASH` (âncora fora do servidor);
  `painel audit ancora SEQ:HASH` confere que a cadeia ainda contém aquele elo e está íntegra; `restore.sh` mostra a
  âncora. Prova local: backup → restauração com a âncora impressa. HMAC com chave fora do banco NÃO foi feito
  (exigiria um segredo novo e convivência de dois formatos de cadeia; a âncora cobre o risco principal).
- **Não feito — precisa de decisão do escritório**: 2.4 (escopo de leitura por equipe/carteira: quem forma cada
  equipe, quem vê o quê), aprovação prévia de alterações de usuário por advogado (2.3) e 5.2 (pendências do
  `PENDENCIAS.md`).
- **Não feito — sem benefício hoje**: 7.2 (dividir `auth.Auth`/`PageData`): refatoração sem defeito associado.
