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
| 2.1 | Senha do DONO do banco montada no contêiner do app | P2 | C | Fase 2 |
| 3.4 | Metadados de auditoria descartados em silêncio (`partes`, `abertos_vinculados`, `quantidade`) | P3 | C (teste) | Fase 2 |
| 3.5 | Gestão de usuários traduz qualquer erro em 404 | P3 | C | Fase 2 |
| 2.5 | Login de conta bloqueada responde mais rápido (oráculo de tempo) | P3 | C | Fase 2 |
| 3.1 | JWKS do Access: falha na renovação horária derruba tudo; busca com mutex travado | P2 | C | Fase 3 |
| 3.2 | Coletas em segundo plano sem `recover` | P2 | H | Fase 3 |
| 1.5 | "Executar coleta agora" síncrono na requisição | P2 | FI | Fase 3 |
| 4.1 | Backup monta tudo num tmpfs de 2 GiB | P2 | FI | Fase 4 |
| 3.3 | Falha de backup sem alerta | P2 | C | Fase 4 |
| 5.1 | Imagem de backup por tag mutável, fora do release | P3 | C | Fase 4 |
| 2.2 | Ações de tomada de conta sem reautenticação | P2 | C | Fase 5 |
| 2.3 | Separação de funções do administrador é nominal | P3 | FI | Fase 5 |
| 2.8 | Limitador por IP único para login e 2FA | P3 | H | Fase 5 |
| 1.4 | Listas de pessoas/processos cortadas em 2.000 sem aviso; seletor de vínculo herda o corte | P2 | FI | Fase 6 |
| 4.2 | Tela de auditoria recalcula a cadeia inteira a cada acesso | P3 | C | Fase 6 |
| 4.4 | Sessões expiradas nunca apagadas | P3 | C | Fase 6 |
| 4.5 | Listas de tarefas/agenda/audiências/intimações sem limite | P3 | C | Fase 6 |
| 1.6 | Regra "A confirmar" checada sobre leitura anterior à transação | P3 | H | Fase 6 |
| 4.3 | Envio ocupa ~3× o tamanho em memória, sem limite de simultâneos | P3 | H | Fase 1 (semáforo) |
| 6.2 | README desatualizado | P3 | C | Fase 6 |
| 2.4 | Leitura sem escopo por equipe (F13) | P2 | C | marco próprio (decisão do escritório) |
| 2.6 | Hash da auditoria sem chave nem âncora externa | P4 | C | marco próprio |
| 2.7 | Segredo TOTP com a chave mestra, sem versão de chave | P4 | C | marco próprio |
| 7.2 | `auth.Auth`/`PageData` acumulam responsabilidades | P4 | C | marco próprio |
| 5.2 | Bloqueios de uso real já listados em `PENDENCIAS.md` | P1 | C (doc) | decisão do advogado/administrador |

## Decisão registrada (Fase 1)

Todo prazo nasce "A CONFERIR", inclusive o digitado pelo advogado (na tela de Prazos ou na tarefa). O advogado confirma
pelo botão "Confirmar prazo", que grava quem e quando e deixa `prazo.confirmado` na auditoria. Editar mantendo a data
mantém a situação; mudar a data volta a "A CONFERIR" e apaga quem/quando. Isso é o que o ADR 0012, item 3, já dizia.
