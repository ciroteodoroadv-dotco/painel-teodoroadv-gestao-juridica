# Relatório do Plano 8 — fechamento (04/11/2026)

## Onde parou
Concluídas e commitadas, na ordem de prioridade: **Tarefas 1, 2, 4, 3 e 6**. **Tarefa 5 (ponta a ponta por clique) não executada** (`paridade/E2E-8.md`).

## Tarefas 1 e 2 — índices do BCB e CEP
Cliente isolado do SGS (`internal/indices/bcb`), migração 0015, serviço, agendador (desligado por padrão), fragmento do rodapé, página do administrador, rota `POST /pessoas/cep` com cliente ViaCEP isolado, `cep.js`.
Nenhum teste usa rede (transporte falso; o teste falha se chamado). Decisões e verificação no Chrome: `NOTAS-PLANO-8.md`.

## Tarefa 3 — análise estática (`ANALISE-ESTATICA.md`)
Primeira passada: gosec 3 altas + 3 médias (6 abertas); staticcheck 21 achados; govulncheck 0 alcançáveis (1 no módulo `golang.org/x/crypto` v0.56.0, GO-2026-5932, corrigida só em pseudo-versão; código não chama).
Corrigido: G710 (redirecionamento da exclusão de pessoa passou a escapar o id), G120 (limite explícito de corpo no multipart), falsos positivos justificados com `#nosec` (G704 no CEP e no healthcheck, G705 no download),
e os 21 do staticcheck (erros de frase ao usuário viraram tipo próprio, `nil` Context nos testes, coordenadas ECDSA pelo formato não depreciado, comparações `x != x` dos testes, código morto). Resultado final:
**gosec 0 alta, 0 média, 0 baixa abertas (12 suprimidos e justificados); staticcheck 0; govulncheck sem vulnerabilidade alcançável.** O CI passou a rodar `scripts/analise-estatica.sh` (mesmas versões).
Não se atualizou `x/crypto`: a única correção é uma versão não marcada; reexecutar o script quando sair a versão (`PENDENCIAS.md`).

## Tarefa 6 — documentos
ADRs 0009 a 0012, `docs/operacao/INDICES-E-CEP.md`, adendos no `RUNBOOK.md`, `COMECAR-USO-REAL.md` e `ESPECIFICACAO.md`, `ACEITE-FINAL.md`, `PENDENCIAS.md`, `GLOSSARIO-PLANO-8.md`, `ACEITE-VISUAL.md`.

## Backup e restauração com o esquema final (Tarefa 4)

**Como foi feito.** Imagem do app construída do código atual (`painel-app:p8`, esquema na migração 0015), pilha **isolada** (`docker compose -p painelp8`, porta de loopback 18081,
volumes e rede próprios, segredos gerados só para ela, `PAINEL_ENV=dev`), usuário de teste e `seed-demo` (agora também com **1 documento fictício cifrado**, `SeedDemoDocumentos`).
Rodou `deploy/teste-backup-local.sh` (copiado para fora do repositório, com `IMAGE_APP_TESTE=painel-app:p8`). Nada da pilha do advogado (`painel-app-1`/`painel-db-1`) foi tocado.

**Resultado: todos os passos passaram.**

| Prova | Resultado |
|---|---|
| Backup criado, permissão 0600, arquivo age, nenhum texto em claro (banco e documentos cifrados) | passou |
| Aviso de falta de cópia externa; 3 backups guardados | passou |
| Restauração em banco novo (sem internet) + conferência (`restore-check`) | passou: cadeia de auditoria íntegra (74 eventos), 1 documento decifrado e conferido |
| **Contagem de 19 tabelas e hash do documento cifrado, original × restaurado** (novo, `REFERENCIA` no `restore-test.sh`) | **idênticos** (users 1, pessoas 4, processos 3, processo_partes 6, atendimentos 3, tarefas 12, agenda_compromissos 3, audiencias 2, diligencias 2, documentos 1, prazos 9, intimacoes 5, oab_monitoradas 1, processo_movimentos 4, monitor_processos 3, calendarios 3, calendario_dias 35, indices_valores 0 (integração desligada), coletas 4) |
| Chave errada | recusada |
| Backup adulterado | recusado |
| Documento ausente no volume | apontado; conferência reprovou |
| Segredos (`db_password`, `db_app_password`, `session_key`, `field_key`) e variáveis `PAINEL_*` no conteúdo decifrado do backup e no log do backup | **nenhum** (novo passo 6) |
| `deploy/secrets/` ignorado pelo Git | confirmado (`git check-ignore -v secrets/field_key` → `.gitignore:6`) |

**Correções feitas (com teste):** `restore-check` conferia `agenda_eventos` (tabela que saiu no Plano 4); a lista agora tem uma tabela por domínio do esquema final (`tabelasConferidas`).
`SeedDemoDocumentos` tem teste próprio (`TestSeedDemoDocumentos`: arquivo cifrado em disco, verificação íntegra, recusa duplicar).

**Não verificado:** cópia externa (rclone) e restauração em outro computador; agendamento do backup no Windows (`agendar-backup-windows.ps1`).
