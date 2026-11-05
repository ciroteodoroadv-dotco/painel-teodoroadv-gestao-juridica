# Notas de decisão — Plano 6 (04/11/2026)

Decisões conservadoras onde o plano foi ambíguo ou a emenda do advogado (E1, E2, E3) exigiu mudar o Plano 5.

1. **Interpretação de "uma por processo e todos os dias" (E3)**: *uma tarefa consolidada por processo em cada dia em que a coleta trouxer
   novidade*, e não uma tarefa fixa por processo todos os dias sem novidade (milhares de tarefas vazias). O advogado pode corrigir.
2. **Consolidada também para o DJEN**: a emenda diz "para cada nova intimação/movimentação" e que o prazo interno vale também para o Plano 5.
   A consolidada foi aplicada às duas fontes **quando o número do processo está cadastrado** (a consolidada é por processo). Intimação do DJEN
   sem processo cadastrado tem só a tarefa individual.
3. **Prazo interno (E3)**: `tarefas.prazo_fatal` = data de hoje em São Paulo + 2 dias corridos, `tarefas.prazo_interno=true`, espelhado em
   `prazos` como **a_conferir** pelo mesmo caminho do prazo fatal da tarefa (`PrazoSync`), agora também em versão que roda **na transação do
   chamador** (`SincronizarNaTx`), porque a tarefa ainda não está commitada. Encerrado (prazo cancelado e `prazo_fatal` limpo) quando a
   tarefa sai de "A confirmar" (Mover ou Update) ou quando a intimação é Confirmada/Cancelada. A tarefa **individual** é encerrada pela decisão
   da intimação; a **consolidada** só ao sair de "A confirmar". Falha ao encerrar devolve erro explicativo (a decisão/tarefa já foi gravada).
4. **Autor das tarefas/prazos automáticos**: `created_by` = dono do registro (DJEN: quem cadastrou a OAB, como no Plano 5; DataJud e
   DJEN sigiloso: quem cadastrou o **processo**). A auditoria usa esse usuário nos prazos e `sistema`/o administrador nas tarefas.
5. **E1**: o monitor consulta **todos** os processos não excluídos com número CNJ válido, em qualquer estado. A **tarefa de monitoramento**
   continua só para ativo/sobrestado. **Processos já existentes** (cadastrados antes da migração) não ganham tarefa de monitoramento sozinhos:
   só ao serem cadastrados/movidos. Backfill fica para decisão do advogado (nenhum dado real no ambiente de desenvolvimento).
6. **E2**: nova coluna `intimacoes.texto` (teto 100.000 caracteres + "[texto cortado]"); `resumo` segue em 2.000 (listas). A migração copia
   `resumo` para `texto` (nada se perde). O texto do DJEN é normalizado (tags HTML removidas, espaços juntados) — as quebras de parágrafo
   do original se perdem. Para o DataJud, o "texto" é a lista de todos os campos do movimento; os dados estruturados também ficam em
   `processo_movimentos` (grau, órgão julgador, código, nome, complementos em JSON, data e hora).
7. **Aviso de fonte** ("Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação.") aparece no drawer "Intimação" das
   linhas de origem DataJud, e também na descrição da tarefa de análise.
8. **Drawer "Intimação" e botão "Ver"**: o B não tem; usa o mesmo mecanismo de drawer (`data-drawer`, sem JS novo). Qualquer papel que lê a
   Agenda abre. O botão "Ver" ficou na coluna de ações (a coluna "—" some). CSS novo: `.texto-integral` e `.aviso-fonte` em `compat.css`.
9. **Documentação oficial do DataJud**: `datajud-wiki.cnj.jus.br/api-publica/endpoints/` devolveu **HTTP 403** ao WebFetch (só documentação). Não
   houve tentativa de contornar. **Pendências de conferência no teste manual** (também em `MONITOR-DATAJUD.md`): formato de requisição/resposta
   (usado o do coletor do B: `POST /api_publica_<alias>/_search`, `match numeroProcesso`, `Authorization: APIKey`), alias do TRE-DF (`tre-dft`,
   **[Chute]** por analogia com `tjdft`), nome do parâmetro `numeroProcesso` no DJEN (**[Provável]**).
10. **Tabela de tribunais**: TJs 27 (07 → tjdft), TRFs 1–6, TRTs 1–24 + TST, TREs 27 + TSE, TJMs MG/RS/SP, STM e STJ = 91 aliases; STF e CNJ sem
    endpoint (processo `bloqueado` com explicação). O teste compara com uma lista **escrita à mão** e confere que a tabela não aceita nada além dela.
11. **Primeira leitura**: se o DataJud devolver zero hits na primeira consulta, nenhuma base é registrada (fica "parcial"); a base nasce na
    primeira resposta com pelo menos um grau. Processo em segredo no DataJud (`nivelSigilo>0` e sem movimentos) fica "parcial" com orientação para
    cadastrá-lo em "Processos em segredo de justiça".
12. **Falha**: cada movimento novo é um item atômico (linha + tarefas + movimento visto). Falha em um item interrompe aquele processo (status
    `falha`, `ultimo_sucesso` não avança); itens anteriores da mesma rodada ficam gravados e a próxima rodada continua dali.
13. **Sigilosos**: janela fixa de 30 dias por consulta (a deduplicação pelo hash evita repetir); só rodam com o monitor ligado **e** o DJEN ligado
    (senão o módulo fica `bloqueado`). Desativar não apaga (`ativo=false`). Uma comunicação que já entrou pela coleta por OAB (mesmo hash) não
    é duplicada.
14. **Intervalo/chave**: `PAINEL_DATAJUD_KEY` ou `_KEY_FILE`; ausente não impede a subida (a coleta fica bloqueada). O tipo `config.Segredo` não aparece
    em `%v`/`%+v`/`%#v`. A pausa entre processos é do cliente (padrão 1 s).
15. **Rodapé**: já tinha os rótulos dos três módulos desde o Plano 5; o Plano 6 passa a gravar `monitor` e `monitor-segredos`.
16. **TDD**: vermelho observado na migração 0013 (coluna ausente), no cliente DataJud (pacote inexistente), no E3 de tarefas (método ausente),
    no monitor e na tela (símbolos ausentes). **Exceções honestas** (teste e código escritos no mesmo passo, sem execução vermelha isolada):
    `internal/monitor/cnj`, `config/datajud.go`, `djen.NormalizarTexto`, `tarefas/monitoramento.go`.
17. **Testes antigos ajustados por mudança de regra**: `migracoes_0012_test` (a reversão agora desce duas migrações), testes de Plano 5 que afirmavam "sem prazo"
    (agora: só prazo interno), `TestSeedDemo`, o "—" da coluna de ações (agora o botão Ver).
18. **Paridade**: ver `paridade/RELATORIO-6.md`.
