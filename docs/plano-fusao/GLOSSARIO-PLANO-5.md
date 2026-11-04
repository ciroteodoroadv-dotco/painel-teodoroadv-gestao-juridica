# Termos novos do Plano 5 (para o glossário do acervo)

- **DJEN** — Diário de Justiça Eletrônico Nacional: publicação oficial e pública das comunicações dos tribunais. Não é em tempo real.
- **OAB monitorada** — número de OAB (com UF) cadastrado em Coleta DJEN para o painel consultar o DJEN.
- **Base da autorização / data da autorização** — o texto e a data que justificam consultar o DJEN por aquela OAB (padrão: distribuição dos
  processos em nome do advogado, sob responsabilidade do titular do escritório). Sem os dois, a coleta recusa a OAB.
- **Coleta** — uma rodada em que o painel consulta o DJEN e registra o que for novo. Vem desligada por padrão.
- **Janela** — intervalo de datas consultado. Recua 7 dias a partir do último sucesso para pegar publicações atrasadas; só avança
  quando a coleta inteira dá certo.
- **Hash** — impressão digital de uma comunicação, usada para não registrar a mesma intimação duas vezes.
- **Triagem / Confirmada / Descartada** — status da intimação: aguardando decisão do advogado / confirmada por ele / descartada, mas
  mantida no histórico.
- **Tarefa de análise** — tarefa aberta automaticamente (estado "A confirmar", sem prazo) para o advogado analisar a intimação.
- **Estado das coletas** — rodapé com a última tentativa, o último sucesso e a mensagem de cada coleta. É histórico local.
- **Faixa de alerta (heartbeat)** — aviso na Agenda quando a coleta ligada fica mais de 6 horas sem sucesso.
- **Teto de 10.000 resultados** — limite da consulta do DJEN; acima dele o painel divide a janela por dia.
- **Segredo de justiça** — processos que a fonte pública pode não devolver; a conferência manual continua necessária.
