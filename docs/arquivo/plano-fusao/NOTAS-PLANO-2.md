# Notas de decisão — Plano 2 (03/11/2026)

Decisões conservadoras onde o plano foi ambíguo ou o B contradiz a segurança do v2.

1. **Omitido (aprovação do advogado pendente):** campo "Nome do arquivo da ficha" (e a dica "Identificador técnico…") em Pessoa e em Processo — existe no B só porque a ficha é um `.md`. Única omissão de campo; a comparação de textos a lista.
2. **Texto de exclusão** (única divergência de texto por segurança): "Excluir esta ficha? Ela deixa de aparecer nas listas e a exclusão fica registrada na auditoria." O botão de confirmar diz "Excluir" (o B diz "Excluir definitivamente", falso no v2).
3. **Categoria de pessoa, regra literal do B:** quem só é "Responsável pelo cadastro" de um processo cai em "Parte adversa" (qualquer vínculo que não seja cliente/terceiro). Mantido por fidelidade; o advogado pode pedir um grupo próprio.
4. **Nomes de auditoria** mantidos em português (`pessoa.criada/alterada/excluida`, `processo.criado/alterado/excluido`), como no histórico existente, e não `create/update/delete` do texto do plano. Novos: `pessoa.documentos_listados`, `processo.estado_movido`, `processo.andamento_registrado`, `processo.financeiro_atualizado`, `processo.proxima_providencia`.
5. **Documento na lista de pessoas:** inteiro só para quem tem `PermPessoasWrite` (a mesma do `Reveal`); a listagem completa é auditada uma vez por abertura (`pessoa.documentos_listados`, só a quantidade). Os seletores de pessoa em Processos seguem a mesma regra.
6. **Erros de validação** viram resposta de texto simples (HTTP 400) e o JS os mostra como toast de erro, mantendo o drawer e o que foi digitado — como o `showToast(..., true)` do B. Mensagens de falha genérica ("Falha ao criar a pessoa." etc.) são as do B.
7. **Estado do processo:** `suspenso→sobrestado`, `encerrado→extinto` no esquema; as opções do drawer "Mover estado" são os valores em minúsculas do B (`ativo`, `sobrestado`…), literais. O motivo da mudança vai para a linha do tempo (andamento) do processo, não para a auditoria (pode conter dado pessoal). Mover para o mesmo estado é recusado.
8. **Financeiro:** o mesmo item (sem diferenciar maiúsculas) atualiza em vez de duplicar (índice único `processo_id, lower(item)`).
9. **Migração 0008 recusa aplicar** se `pessoas`/`processos` tiverem linhas (e o Down também): o v2 começa limpo e converter dado real em silêncio é perigoso. Colunas extras além do plano: `processo_partes.ordem` (para "primeira parte cliente") e o índice único do financeiro. `uf` é `char(2)` anulável.
10. **Valor da causa:** a máscara do B interpreta os dígitos como centavos; o servidor faz o mesmo (limite de 15 dígitos).
11. **Exclusão de processo** com tarefa/prazo em aberto exige marcar "Excluir mesmo assim" (via interfaces `AbertosDoProcesso` de tarefas e prazos).
12. **Páginas `/pessoas/{id}` e `/processos/{id}`** (ficha de leitura) foram mantidas: outros domínios (documentos, agenda, tarefas, prazos) apontam para elas; o relatório do titular (LGPD) continua em `/pessoas/{id}/relatorio`. Não existem no B.
13. **CEP:** botão e dica idênticos; a consulta responde "Consulta de CEP indisponível neste painel." (Plano 8). Os drawers de confirmação de substituição de endereço do B ficam para esse plano.
14. **Atendimento de origem:** `<select disabled>` com a dica do plano, até o Plano 3.
15. **Mover estado → tarefa de monitoramento:** o texto do B permanece; o efeito só vale no Plano 6.
16. **TDD:** a Tarefa 1 teve vermelho real (migração e contador ausentes). Nas Tarefas 2 e 3 a leitura obrigatória do B e o esqueleto vieram antes dos testes; os testes foram escritos em seguida e a capacidade de falhar foi provada por mutação (categoria e pendências alteradas → testes vermelhos → restauradas). Os commits das Tarefas 1 e 2 não são verdes isoladamente (esquema novo antes dos domínios; pessoas depende de `processos.PapeisPorPessoa`); o commit da Tarefa 3 fecha tudo.
17. **compat.css:** classes novas substituem os `style=""` do B (`.uc`, `.span2`, `.cep-linha`, `.btn-bloco`, `.row-parte`, `.outro-campo`, `.hint-*`) e regras para igualar o B: `.acc th/td` quebram linha, `.toolbar .search` 240px, `.row-parte label.radio-opt` sem negrito.
18. **Fixtures do B:** `tools/refb/pessoas_processos.py` substitui as pessoas/processos antigas (tipoPessoa agora `Física`/`Jurídica`, partes por wikilink) e é igual ao seed do v2. `__pycache__` saiu do Git.
