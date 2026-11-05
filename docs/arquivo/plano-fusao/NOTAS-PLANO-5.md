# Notas de decisão — Plano 5 (03 e 04/11/2026)

Decisões conservadoras onde o plano foi ambíguo ou o B contradiz a segurança do v2.

1. **"Consentimento" virou "Base da autorização" + "Data da autorização"** (determinação expressa do advogado titular, 03/11/2026, enviada
   durante a execução): os processos das OABs dos parceiros são de clientes dele; a base da autorização é a distribuição dos processos em
   nome desses advogados e ele assume a responsabilidade pelas consultas. Colunas `autorizacao_base` (texto, padrão "Distribuição dos
   processos em nome do advogado — responsabilidade do titular do escritório") e `autorizacao_em` (data, padrão hoje). A coleta continua
   exigindo OAB ativa com os dois campos, a data não pode ser futura, e a auditoria registra quem cadastrou. Nenhum texto do sistema
   condiciona a coleta a consentimento individual do advogado parceiro.
2. **Swagger do DJEN (`djen.yml`) devolveu HTTP 403** ao WebFetch (documentação pública, só leitura). Não houve tentativa de contornar. Os
   parâmetros (`numeroOab`, `ufOab`, `pagina`, `itensPorPagina`, `dataDisponibilizacaoInicio/Fim`), o formato da resposta (`count`,
   `items`) e o cabeçalho `x-ratelimit-remaining` vieram do código do coletor do B (`intimacoes.go`), lido para contraste. **[Provável]**
   que coincidam com o swagger; conferir contra `djen.yml` no teste manual orientado antes de ligar a coleta.
3. **Paginação:** página repetida ou vazia ANTES de atingir `count` é **erro** (coleta não confirmada), não "fim". O plano diz "encerra";
   encerrar como sucesso poderia dar a coleta incompleta por completa. Sem `count`, página curta encerra. 429 repete a mesma página (até 3
   tentativas, espera de 60 s; relógio injetável) e `x-ratelimit-remaining: 0` também espera.
4. **Estado e janela:** o módulo no rodapé chama-se `intimacoes` (rótulo "DJEN", como no B). A janela por OAB mora em
   `oab_monitoradas.ultimo_sucesso` (não polui o rodapé); `coletas` guarda o agregado. Só `sucesso` avança último sucesso e janela.
   Status agregado: `sucesso` só se todas as OABs deram certo; parte falhou = `parcial`; nenhuma deu certo = `falha`; desligada ou sem OAB
   ativa = `bloqueado`.
5. **Autor das tarefas automáticas:** `tarefas.created_by` exige usuário; usa-se quem cadastrou a OAB. A auditoria registra o usuário que
   mandou coletar, ou `sistema` quando é a rodada agendada.
6. **Vínculo da tarefa:** com o número CNJ de processo existente, a tarefa é `processo` + `intimacao_id` e a Origem mostra "PRO. xxxxxxx · DJEN";
   sem processo, `origem_tipo='intimacao'` e a Origem mostra "DJEN" (o B mostraria o hash). Tarefa de intimação sem processo não aceita PDF
   ("Vincule a tarefa a um processo ou atendimento para anexar documentos.") e a "Tarefa vinculada" (filha) escolhe o próprio vínculo.
7. **Estado na coluna "Tarefa de análise":** o B mostra a chave da pasta ("a-confirmar"). O v2 repete a chave, por determinação "B prevalece".
   Trocar pelo rótulo ("A confirmar") é uma linha em `tarefas.PorIntimacoes`, se o advogado preferir.
8. **Lista honesta:** sem nenhuma intimação e sem coleta concluída, a categoria mostra "Intimações indisponíveis." com o estado real e
   "Confira o DJEN manualmente." (texto do B + aviso); lista vazia só depois de uma coleta com sucesso.
9. **Rodapé:** miolo renderizado no servidor (`GET /sync/estado`, qualquer usuário logado) e recarregado por `data-refresh` a cada 60 s e ao
   voltar à aba; falha de leitura mostra "Estado das coletas indisponível". Horário no formato do `toLocaleString('pt-BR')` do Chrome
   (com vírgula: `03/11/2026, 12:00:00 (Brasília)`), medido no B.
10. **Tabela de Intimações na Agenda** (e não em `intimacoes/templates/tabela.html`, que o plano previa): a Agenda monta a página. Confirmar/Descartar
    são formulários (`data-drawer-form`) com `display:contents` na linha de ações para os botões ficarem como itens flex, como no B.
11. **Permissões novas:** `PermIntimacaoDecide` (só advogado), `PermColetaManage` (só administrador). Leitura da lista: quem lê a Agenda.
12. **WhatsApp (fila do B)** não reproduzido (F13/ADR). **Segredo de justiça por credencial própria** e DataJud: fora do plano (Fases D e Plano 6).
13. **"Executar coleta agora"** devolve erro quando desligada/falha ("A coleta falhou. …"), toast de sucesso nunca diz "nenhuma intimação nova".
14. **Nenhuma exclusão de intimação** neste plano: o status existente é mantido; o plano 4C trata cancelamento.
15. **TDD:** testes escritos antes do código; o vermelho foi observado (compilação ou SQL) em migração, coletas, tarefas, config/RBAC e telas.
    Exceção honesta: o cliente `djen` e `coletas.Linha` foram implementados antes da primeira execução dos testes já escritos (vermelho = pacote inexistente,
    não executado isoladamente).
16. **Suíte:** na primeira execução em banco novo `internal/app` falhou uma vez e passou nas seguintes sem mudança de código (provável efeito do banco
    recém-criado); não reproduzido.
17. Tarefas já existentes: a seed ganhou 3 tarefas de análise (TAR. 0000007 a 0000009) e o B de referência (`tools/refb/intimacoes.py`) também.
