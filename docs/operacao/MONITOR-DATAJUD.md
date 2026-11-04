# Monitor do DataJud — guia para quem não é de informática

## O que o monitor faz

O **DataJud** é a base de dados do CNJ com as **movimentações** (andamentos) dos processos. O monitor consulta, **processo a processo**,
todos os processos cadastrados no painel que tenham **número CNJ válido** — em **qualquer estado** (ativo, sobrestado, extinto ou arquivado).
Para cada movimentação nova, o painel:

1. registra uma linha na aba **Agenda > Intimações**, com a **Origem "DataJud"** (status **Em triagem**);
2. abre uma **tarefa de análise individual** em **A confirmar**;
3. abre (ou completa) a **tarefa consolidada do dia** daquele processo, que lista as novidades do dia;
4. dá às duas tarefas um **prazo interno de confirmação de 2 dias corridos**, com o selo **A CONFERIR**. Esse prazo **não é o prazo
   processual**: é o prazo para o advogado confirmar a tarefa. O painel **não calcula nem presume prazo processual**.

O botão **Ver** de cada linha abre o painel lateral "Intimação" com todos os campos que o DataJud devolveu (grau, órgão julgador, código e
nome do movimento, complementos, data e hora).

## O que o monitor NÃO faz (leia isto)

- **O DataJud não fornece o teor da intimação.** A tela diz, em cada detalhe: "Movimentação processual (DataJud). Esta fonte não fornece o
  teor da intimação." Para saber o que a intimação manda, é preciso abrir o processo no sistema do tribunal ou o DJEN.
- **Não é em tempo real.** O CNJ atualiza a base de tempos em tempos; uma movimentação pode demorar a aparecer.
- **"Sem registro" não é "sem movimentação".** Se o DataJud não devolve nada para o número, o painel marca o processo como **parcial** e
  escreve: "Sem registro no DataJud para este número (pode haver segredo de justiça, atraso na publicação ou número incorreto). Confira nas
  fontes oficiais." Pode ser segredo de justiça, atraso ou número errado.
- **"Sem resultado" nunca significa "sem prazo".** A conferência nas fontes oficiais continua obrigatória.
- **Processo sem número CNJ válido não é consultado.** O estado dele diz "Processo sem número CNJ válido: não é consultado".
- Não protocola, não envia mensagem e não altera processo nenhum.

## Linha de base e novidades

Na **primeira consulta** de um processo, os andamentos que já existem são só registrados como **linha de base** ("Linha de base registrada em
dd/mm/aaaa"): não viram alerta nem tarefa. Só o que surgir **depois** é novidade. Um andamento só é dado como "visto" junto com a linha
da tabela e a tarefa: se algo falhar, nada fica pela metade e a próxima rodada tenta de novo.

## Tarefa consolidada do dia (interpretação registrada)

"Uma tarefa por processo e todos os dias" foi entendida como **uma tarefa consolidada por processo em cada dia em que a coleta trouxer
novidade** — não uma tarefa fixa por processo todos os dias sem novidade, o que geraria milhares de tarefas vazias. O advogado pode corrigir.

## Tarefa de monitoramento (ciclo de vida do processo)

Ao **cadastrar** um processo ativo, ou ao **movê-lo** para ativo/sobrestado, o painel garante a tarefa **"Monitoramento processual — número"**
(estado Pendente, sem prazo). Ao mover para **extinto/arquivado**, a tarefa é baixada para **"Concluída com sucesso"** com uma nota de
encerramento e **continua listada** (nada se apaga). Voltando a ativo, uma **nova** tarefa é criada. Nunca há duas abertas.

## Processos em segredo de justiça

Processos em segredo de justiça podem não aparecer no DataJud. Na página **Coleta DJEN**, a seção **Processos em segredo de justiça** liga um
processo cadastrado (com número válido) a uma OAB ativa, com a base da autorização. A consulta usa o DJEN pela OAB **filtrando pelo número do
processo** — a pauta inteira da OAB nunca é lida por essa via. O resultado aparece na tabela com a **Origem "DJEN (sigiloso)"**.

## Como configurar a chave e ligar (só depois do teste manual)

A chave pública do DataJud (gratuita, fornecida pelo CNJ) é um **segredo**: fica **num arquivo**, nunca no banco, na tela ou em log.
Quem cuida do servidor cria o arquivo de segredo e define `PAINEL_DATAJUD_KEY_FILE` com o caminho dele (ou `PAINEL_DATAJUD_KEY`).
Sem chave, a coleta fica "bloqueada" com a mensagem "Chave do DataJud não configurada.".

O monitor vem **DESLIGADO**. Neste estado o painel não faz nenhuma consulta à internet; a página Coleta DJEN mostra "Monitor desligado neste
ambiente." Para ligar: `PAINEL_DATAJUD_ATIVO=true` e reiniciar. **Faça isso somente depois do teste manual orientado:**

1. configurar a chave no arquivo de segredo;
2. ligar o monitor;
3. acompanhar a **primeira rodada com UM processo de cliente** (cadastre só um, ou aguarde a rodada e olhe a tabela);
4. conferir a linha de base e o formato real da resposta contra a documentação oficial **antes** de ampliar.

Outras variáveis (todas opcionais): `PAINEL_DATAJUD_INTERVALO` (padrão 24h, mínimo 15m), `PAINEL_DATAJUD_PAUSA` (padrão 1s entre processos, mínimo
500ms), `PAINEL_DATAJUD_ALERTA_HORAS` (padrão 36), `PAINEL_DATAJUD_URL` (padrão o endereço oficial; precisa ser https).

## Pendências de conferência (a fazer no teste manual)

A documentação oficial (`datajud-wiki.cnj.jus.br/api-publica/`) devolveu **HTTP 403** ao painel de desenvolvimento e **não foi contornada**.
Ficam para conferir no teste manual: (1) o formato exato da requisição `POST .../api_publica_<alias>/_search` e da resposta (campos
`movimentos`, `complementosTabelados`, `orgaoJulgador`, `dataHora`); (2) o alias do TRE do Distrito Federal (usamos `tre-dft`); (3) o
parâmetro `numeroProcesso` da consulta do DJEN por processo (o swagger do DJEN também devolveu 403).

## Rodapé "Estado das coletas" e alerta

O rodapé mostra três linhas: **DJEN**, **DataJud** e **DJEN por processo sigiloso**. É histórico local: não prova que a coleta está agendada
nem substitui a conferência nas fontes oficiais. Com o monitor ligado e sem sucesso há mais de 36 horas, a Agenda mostra uma faixa amarela.

## Limites e coleta educada

Uma consulta por vez, pausa entre processos, espera e repetição no erro 429, resposta limitada em tamanho, nenhum redirecionamento a outro
endereço, identificação genérica ("painel-teodoro") e nunca o e-mail do usuário. Os logs trazem só contagens e códigos de erro.

## Dados pessoais (LGPD)

Movimentações podem citar nome de parte. Ficam só em tabelas protegidas, com acesso por papel (todos leem a Agenda; só o advogado decide;
só o administrador roda o monitor e cadastra processos em segredo de justiça).
