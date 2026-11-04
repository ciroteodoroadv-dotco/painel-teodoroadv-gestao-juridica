# Termos novos do Plano 6 (para o glossário do acervo)

- **DataJud** — base nacional de dados processuais do CNJ, com API pública. Traz as **movimentações** do processo (andamentos), não o teor das
  intimações. Não é em tempo real. O painel só a consulta para processos já cadastrados, pelo número CNJ.
- **Monitor** — a rotina que consulta o DataJud processo a processo. Vem desligada (`PAINEL_DATAJUD_ATIVO=false`).
- **Movimentação / movimento** — um andamento do processo no DataJud (código, nome, data e hora, complementos, grau e órgão julgador).
- **Grau** — a instância (G1 = primeiro grau, G2 = segundo grau…). O DataJud devolve um bloco por grau; o painel lê todos.
- **Linha de base** — na primeira consulta de um processo, os andamentos que já existem são só registrados como ponto de partida: não viram
  alerta nem tarefa. Só o que surgir depois é novidade.
- **Sem registro** — o DataJud não devolveu nada para o número. Não significa "sem movimentação": pode ser segredo de justiça, atraso na
  publicação ou número errado. O painel marca o processo como **parcial**.
- **Tarefa de monitoramento** — tarefa automática ("Monitoramento processual — número"), criada ao cadastrar ou reativar um processo ativo/sobrestado
  e baixada ("Concluída com sucesso") quando o processo vira extinto/arquivado. Nunca há duas abertas para o mesmo processo.
- **Tarefa de análise individual** — uma por intimação ou movimentação nova, em "A confirmar".
- **Tarefa consolidada do dia** — uma por processo e por dia de coleta com novidades; lista as novidades do dia.
- **Prazo interno de confirmação** — prazo fatal de 2 dias corridos (a partir da criação, data de São Paulo) das tarefas de análise, para o advogado
  confirmar a tarefa e não perder o prazo processual. **Não é o prazo processual.** Aparece como "A CONFERIR" e acaba quando a tarefa sai de
  "A confirmar" ou a intimação é decidida.
- **Segredo de justiça (DJEN por processo)** — processos sigilosos não vêm do DataJud; o painel consulta o DJEN pela OAB **filtrando pelo número do
  processo** (nunca a pauta inteira).
- **Origem** — de onde veio a linha da tabela de Intimações: DJEN, DataJud ou "DJEN (sigiloso)".
- **Texto completo** — texto integral da comunicação do DJEN (teto de segurança de 100.000 caracteres, com a marca "[texto cortado]"). Para o
  DataJud, são todos os campos do movimento; a fonte não fornece o teor da intimação.
- **Alias do tribunal** — nome do endpoint do DataJud (ex.: tjsp, trt2, tre-sp), deduzido dos campos J e TR do número CNJ.
- **Chave do DataJud** — chave pública do CNJ, guardada só em segredo (arquivo), nunca no banco, no log ou na tela.
