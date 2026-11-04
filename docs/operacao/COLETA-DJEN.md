# Coleta de intimações do DJEN — guia para quem não é de informática

## O que a coleta faz

O DJEN (Diário de Justiça Eletrônico Nacional) é a publicação oficial e pública das comunicações dos tribunais.
O painel pode **consultar o DJEN pelo número da OAB** dos advogados cadastrados e trazer as comunicações para a aba
**Agenda > Intimações**. Para cada comunicação nova, o painel:

1. registra a intimação na lista (status **Em triagem**);
2. abre uma **tarefa de análise** no estado **A confirmar**, com o título "Analisar intimação — tipo — número do processo";
3. **não calcula nenhum prazo.** A tarefa nasce sem data. Quem apura o prazo, e o confirma, é o advogado.

O painel **detecta e sugere; o advogado confirma.** Só o advogado pode **Confirmar** ou **Cancelar** uma intimação (o cancelamento pede o motivo); a cancelada
continua no histórico: nada se apaga.

## O que a coleta NÃO faz (leia isto)

- **O DJEN não é em tempo real.** Uma comunicação pode demorar para aparecer, e a consulta roda de tempos em tempos.
- **Processos em segredo de justiça podem não aparecer.** A coleta lê só o que a fonte pública devolve pela OAB.
- **"Sem resultado" nunca significa "sem prazo".** Se a coleta não trouxe nada, isso não prova que não há intimação nem prazo
  correndo. A conferência nos sistemas dos tribunais e no DJEN continua obrigatória. Por isso a tela e o rodapé não dizem
  "nenhuma intimação nova" como garantia: dizem o estado da última coleta.
- Não envia mensagem a ninguém, não protocola nada e não altera processo nenhum.

## Cadastro da OAB (somente o administrador)

Menu **Coleta DJEN > Nova OAB**. Campos:

- **Nome do advogado**, **Número da OAB** (só dígitos) e **UF**;
- **Base da autorização**: o texto já vem preenchido com "Distribuição dos processos em nome do advogado — responsabilidade do
  titular do escritório". Pode ser ajustado;
- **Data da autorização**: obrigatória (já vem com a data de hoje);
- **Ativa**: só OABs ativas entram na coleta.

A coleta **só consulta OAB ativa com base e data da autorização preenchidas**, e o sistema registra na auditoria **quem
cadastrou** cada OAB. Sem esses dados, o cadastro é recusado e a recusa fica registrada.

## Como ligar (só depois do teste manual)

A coleta vem **DESLIGADA**. Neste estado o painel não faz nenhuma consulta à internet; a página Coleta DJEN mostra
"Coleta desligada neste ambiente." e o rodapé mostra o estado "bloqueado".

Para ligar, quem cuida do servidor define `PAINEL_DJEN_ATIVO=true` e reinicia o painel. **Faça isso somente depois de um teste
manual orientado**: cadastrar as OABs, ligar, acompanhar a primeira rodada e conferir os resultados com o DJEN.

Outras configurações (todas opcionais): `PAINEL_DJEN_INTERVALO` (padrão 3h, mínimo 15 minutos) e `PAINEL_DJEN_ALERTA_HORAS`
(padrão 6). O botão **Executar coleta agora** só aparece com a coleta ligada.

## Rodapé "Estado das coletas" e o alerta

O rodapé de toda página mostra, para cada coleta, a última tentativa, o último sucesso e uma mensagem curta (só contagens e
códigos de erro, nunca o texto de uma intimação). É um histórico local: **não prova que a coleta está agendada nem substitui a
conferência nas fontes oficiais.**

Se a coleta estiver ligada e o último sucesso tiver mais de 6 horas, a Agenda mostra uma faixa amarela:
**"Coleta de intimações sem sucesso desde …"**. O que fazer: abrir o DJEN e conferir as OABs manualmente; depois avisar quem cuida
do servidor. A faixa some sozinha quando uma coleta der certo.

## Limites e coleta educada

Uma consulta por vez, com pausa de 1 segundo entre elas; se o DJEN pedir calma (erro 429), o painel espera 1 minuto e repete a
mesma página; o painel lê todas as páginas e, se o DJEN atingir o teto de 10.000 resultados, refaz a consulta dia a dia. Se um
único dia ainda estourar, a coleta fica **parcial** e é avisado. O painel se identifica como "painel-teodoro" e nunca envia
e-mail de usuário.

## Dados pessoais (LGPD)

O texto de uma intimação pode conter dados pessoais. O painel guarda só um resumo (até 2.000 caracteres) e o link oficial (só
https), não escreve esse texto em logs e limita o acesso por papel: assistente e administrador veem a lista, só o advogado
decide, só o administrador cadastra OAB e roda a coleta.
