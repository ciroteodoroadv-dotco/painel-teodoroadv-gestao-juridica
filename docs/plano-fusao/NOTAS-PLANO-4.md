# Notas de decisão — Plano 4 (03/11/2026)

Decisões conservadoras onde o plano foi ambíguo ou o B contradiz a segurança do v2.

1. **Participantes e página de detalhe de evento** removidos junto com `agenda_eventos`/`agenda_participantes` (migração 0010 só aplica com as tabelas vazias; o B não os tem).
2. **Conflito de horário (lido em `agenda/http.go`, `audiencias/http.go`, `diligencias/http.go` do B):** mesmo dia e mesma hora exata. Crítico = só audiência (o "Prazo Fatal" saiu por decisão de 29/10). Compromisso e diligência são **recusados** diante de audiência; audiência **nunca** é bloqueada (só aviso na criação, não na edição). Edição de diligência não checa conflito (como o B). Diferença: audiência **cancelada** não bloqueia (o B a contava); mensagem de bloqueio é a do B, exibida como toast de erro (HTTP 400 em vez de 409).
3. **Aviso de tarefa no mesmo dia** ("Atenção: há uma tarefa com prazo no mesmo dia (\"…\").") usa o prazo de entrega das tarefas em aberto e o título.
4. **Tipo "Diligência" na edição de compromisso:** o select mostra as 4 opções como no B, mas o servidor recusa ("Diligência tem ficha própria — use Novo compromisso."), pois o B gravaria um item incoerente.
5. **Processo vinculado em audiência/diligência:** texto com máscara CNJ; número existente grava também `processo_id`. Na diligência criada pela Agenda, o select de processo grava o vínculo e o texto passa a ser o rótulo do processo (o B gravava o id interno como texto).
6. **Cancelar audiência:** permissão `PermAudienciaCancel` só do advogado (assistente e administrador: 403); reautentica a senha do usuário; ficha preservada (motivo, quem, quando); audiência cancelada não é editável. Mensagens de validação específicas ("Informe o motivo do cancelamento.", "Senha incorreta.") em vez do genérico do B.
7. **Edição de audiência** não tem "Observações" (o B também não); o campo `descricao` só existe na criação.
8. **Permissões novas:** audiências e diligências: leitura para todos; escrita para advogado e assistente. Administrador só lê.
9. **"Excluir" de compromisso:** botão e toast "Ainda não implementado nesta fase do painel." (erro), sem exclusão; só aparece a quem escreve.
10. **Home:** `GET /` redireciona (302) para `/agenda`; a faixa `.notice` e a lista de prazos ficam no topo da Agenda (`Auth.HomeExtras`).
11. **Busca da Agenda e de Audiências:** no navegador, ao digitar (`data-filtro`), sem acento; o contador do botão soma as tabelas do painel. `TarefaLookup` do plano ficou sem o parâmetro `q` por isso.
12. **Cabeçalho `X-Painel-Ok`:** cabeçalho HTTP não carrega UTF-8 (o navegador leria "Ã­"); o texto agora vai em %XX e o JS decodifica (`App.textoDoCabecalho`). Corrige "Tarefa excluída." do Plano 3.
13. **Tarefas na Agenda:** a lista de clientes ganhou o código PES (`VinculoLookup.Clientes`); `AbertasPorSubtitulo` foi substituída por `AgendaPorSubtitulo` (comparação sem acento, por prazo ou data base). Estados atrasados seguem `computeTarefaStatus`.
14. **Seed:** cancela uma audiência com um `Reauth` que aceita qualquer senha, usado só em `SeedDemo` (dev).
15. **TDD:** testes escritos antes do código nas Tasks 1–5 (vermelho por tabela inexistente / não compilava). Exceção: `tarefas.Agenda*` teve o código escrito antes do teste, que passou de primeira. Os commits da Task 2–5 foram juntados num só (o estado intermediário não compilava).
16. **Verificação visual no Chrome e paridade de estilo computado B × v2 não foram feitas nesta rodada** (ver RELATORIO-4.md).
