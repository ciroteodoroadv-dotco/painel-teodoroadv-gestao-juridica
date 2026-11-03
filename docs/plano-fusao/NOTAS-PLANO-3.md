# Notas de decisão — Plano 3 (03/11/2026)

Decisões conservadoras onde o plano foi ambíguo ou o B contradiz a segurança do v2.

1. **Fluxo do recibo não reproduzido:** o B abre "Novo processo" com dados extraídos do recibo depois do upload do PDF. O v2 só guarda o PDF (cifrado, tipo conferido pelo conteúdo) e mostra "Documento guardado.".
2. **Excluir tarefa:** rótulo "Sua senha *", texto e botão "Excluir" adaptados (reautenticação do usuário logado, exclusão lógica, auditoria `tarefa.excluida` / `tarefa.exclusao_negada`). Senha errada conta na trava de bloqueio do login.
3. **Dica do PDF:** "O PDF será guardado, cifrado, junto ao caso." (o B diz que converte para .md). Toast "Documento guardado.".
4. **`confirmar` no `PrazoSync.Sincronizar`:** a assinatura do plano não tinha como saber se o ator é advogado. Acrescentei o parâmetro `confirmar bool` (o handler passa `PermPrazoConfirm`) e o método `Confirmar` (botão "Confirmar prazo").
5. **Prazo de tarefa só se edita pela tarefa:** a tela de Prazos recusa editar um prazo que pertence a uma tarefa. Cumprir/cancelar/excluir no domínio Prazos continuam possíveis; a coluna `prazo_fatal` da tarefa não acompanha (a situação mostrada vem sempre do domínio Prazos).
6. **Efeitos no prazo:** concluída com sucesso -> prazo confirmado vira cumprido (a conferir é cancelado, pois nunca valeu); concluída sem sucesso/cancelada/data apagada -> cancelado; excluída -> some das listas; reabrir ressincroniza (só advogado confirma). Se o Prazos falhar depois de a tarefa gravar, o usuário é avisado para salvar de novo (as duas gravações não compartilham transação por causa da fronteira de domínios).
7. **"Mover para..." de "A confirmar":** só quem tem `PermPrazoConfirm`; vale também no Status do drawer de edição.
8. **Permissões novas:** `PermAtendimentoRead/Write` (advogado e assistente escrevem; todos leem) e `PermTarefaDelete` (advogado e administrador). O botão Excluir só aparece para quem escreve **e** pode excluir (o administrador não escreve tarefas, então não o vê).
9. **Assistente e "Novo cliente":** o botão que abre "Nova pessoa" some para quem não tem `PermPessoasWrite` (assistente).
10. **Tela Agenda/"Hoje":** o bloco "minhas tarefas" saiu (o responsável da tarefa agora é Pessoa, não usuário). A Agenda volta a listar tarefas no Plano 4 (`AbertasPorSubtitulo` já existe).
11. **Itens a mais que o B (únicas adições):** linha `Tarefa vinculada a: TAR. …` na lista (decisão do plano); aviso "A CONFERIR — um advogado precisa confirmar esta data." no campo Prazo fatal do drawer de edição; selo `A CONFERIR` e botão `Confirmar prazo` ao lado do selo do prazo fatal (regra de ouro).
12. **Datalist de Subtítulo no drawer de edição:** o B referencia a lista mas só a define no "Nova tarefa"; no v2 a lista existe nos dois (efeito pretendido).
13. **Ordem das listas:** Atendimentos e Tarefas por código (o B não define ordem; a do v2 e a do servidor de referência coincidem).
14. **Busca de Tarefas** por GET `?q=` ao enviar, sem acento nem maiúsculas (o B ignora só a caixa); o JS recarrega o conteúdo mantendo os painéis abertos e a URL.
15. **Acordeão após recarregar:** `recarregarConteudo` agora preserva os painéis abertos (como `state.accOpen` do B).
16. **Migração 0009** só aplica com `tarefas` vazia; Down só com `tarefas`/`atendimentos` vazias e sem documento ligado só a atendimento/tarefa.
17. **TDD:** os testes vieram antes do código em todas as tarefas e foram vistos falhar (migração: tabela inexistente; atendimentos e tarefas: não compilava). A matriz de papéis (RBAC) teve teste e código no mesmo passo (sem vermelho separado). Os commits das Tarefas 1 e 2 não são verdes isoladamente (a 0009 refaz `tarefas` antes do domínio novo); o commit da Tarefa 3/4 fecha tudo.
18. **Administrador** continua sem escrever atendimentos/tarefas (só lê).
