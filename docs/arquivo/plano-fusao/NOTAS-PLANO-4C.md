# Notas de decisão — Plano 4C (04/11/2026)

Determinação do advogado: tarefa é ata e não se apaga; audiência, compromisso, diligência e intimação só se cancelam; processo duplicado
exige marcador e justificativa. Decisões conservadoras onde o plano foi ambíguo:

1. **Tarefa:** removidos rotas, drawer, serviço e `PermTarefaDelete` (nenhum papel apaga). A auditoria do "Mover para…" passou a `tarefa.estado_alterado`
   com `de`/`para` (as chaves `de` e `para` entraram na lista permitida de metadados da auditoria; `processo.estado_movido` já tentava gravá-las e elas
   eram descartadas em silêncio). `Service.Reauth` de Tarefas saiu (só a Audiência ainda reautentica, no cancelamento). A coluna `excluido_em` continua no
   banco porque as listas filtram por ela, mas nada a preenche para tarefas.
2. **Cancelar compromisso / diligência / intimação:** drawer com motivo obrigatório, sem senha (o plano não pede; a audiência mantém motivo + senha).
   O botão "Excluir" do compromisso (que no B só mostrava "Ainda não implementado…") virou "Cancelar" (`btn-danger-text`). Só advogado
   (`PermCompromissoCancel`, `PermDiligenciaCancel`; intimação segue `PermIntimacaoDecide`). Assistente edita, não cancela.
3. **Contadores:** o compromisso/diligência cancelado continua listado, com selo "Cancelado"/"Cancelada" no lugar das ações, e continua nas contagens do menu
   e das pílulas, como a audiência cancelada (que o v2 já listava e contava). "Não bloqueia horário": compromisso e diligência nunca bloquearam outro
   item (só audiência bloqueia) e a audiência cancelada já não bloqueia. Cancelado não pode ser editado ("Compromisso cancelado não pode ser editado.").
4. **Intimação:** `descartada` virou `cancelada` (migração 0012 converte os registros existentes, com motivo "Descartada antes da migração 0012…"; nada é
   perdido). "Confirmar" continua um botão direto; "Cancelar" abre o drawer. `Decidir` só aceita `confirmada`. No B a ação era "Descartar"; a diferença é
   determinação do advogado.
5. **Diligência:** o `status` já aceitava "Cancelada"; ganhou também o booleano `cancelado` e as colunas de quem/quando/por quê, como as demais.
6. **Processo duplicado:** o índice único do número saiu (migração 0012); a regra mora no serviço. Duplicata = outro processo não excluído com o mesmo
   número CNJ (comparação pelo número normalizado, ou seja, ignora a pontuação digitada). `duplicado_de` aponta para o mais antigo. Justificativa: mínimo
   de 10 caracteres úteis (também no banco). Ao editar:
   - número inalterado e o processo já é o original: não vira duplicata (nem "de si mesmo" nem do seu próprio duplicado);
   - duplicata com número inalterado: mantém marcador e justificativa;
   - número mudado para um que já existe: exige justificativa;
   - número corrigido para um livre: o marcador some (a justificativa anterior permanece na auditoria `processo.duplicado_justificado`).
   Número vazio nunca é duplicata.
7. **Consulta antes de salvar:** é `POST /processos/duplicado` (e não `GET …?numero=`, como o plano sugeria) para o número do processo não ir na URL
   (regra de privacidade; o número identifica o caso). O bloco volta como fragmento HTML e o JS (`data-verifica-duplicado`) o põe abaixo da linha do número;
   a justificativa já digitada é preservada. O servidor valida sempre, mesmo sem o JS.
8. **Dev:** a migração 0012 só converte e acrescenta colunas; a reversão recusa se houver cancelamentos ou duplicatas (nada se apaga).
9. **Pendências da conferência (Task 4):** mensagens de recarga do B para agenda, audiências e diligências em `forms.js` (com teste que lê o arquivo);
   teste que enumera todas as rotas (nenhuma de exclusão nos registros de ata, nenhuma DELETE, nenhuma escrita pública fora de login/logout);
   `RepCPFMask` renomeado para `RepCPFFormatado` (a função formata, não mascara); comentário do RBAC atualizado.
10. **TDD:** vermelho observado em cada tarefa (rota/coluna ausentes, mensagens, comportamento antigo). Na Task 1, a migração de teste 0011 (`descartada`)
    e a 0008 (número único) foram atualizadas porque as regras mudaram.
11. **Paridade:** ver `paridade/RELATORIO-4C.md`.
