# Calculadora de prazo — guia para quem usa

## O que é

Uma ferramenta de **apoio**. Você informa a data do evento, como o prazo conta e quantos dias, e ela mostra o
vencimento **com todas as premissas**. Fica no menu lateral, logo depois de **Audiências**. Não está ligada a ficha
nenhuma: só calcula e mostra.

## O que ela calcula

- **CPC, dias úteis**: não conta sábado, domingo, feriado nem o período de 20/12 a 20/01.
- **CPP, dias corridos**, e **prazo material cível** e **penal**.
- Feriados nacionais por lei, com a norma ao lado de cada data excluída. A **Sexta-Feira Santa** não é excluída por
  padrão (a lei a trata como feriado religioso municipal): o resultado mostra o vencimento mais curto e, em
  "alternativas", o vencimento se o tribunal a adotar como dia sem expediente. Se o calendário do tribunal a trouxer
  com fonte, ela é excluída.
- Em prazo corrido (CPP e material cível), **sábado não é prorrogado**; no material cível **domingo também não**
  (só feriado prorroga). Cada leitura mais longa aparece em "alternativas", com `[VERIFICAR]`.
- Feriados e suspensões **do tribunal**, se estiverem cadastrados em **Calendário forense** (cada data com a fonte).

## O que ela NÃO calcula (e avisa em todo resultado)

Termo inicial por juntada de AR ou mandado, citação, carga dos autos ou audiência; consulta tácita de 10 dias;
indisponibilidade do sistema do tribunal; suspensão por portaria que não esteja cadastrada; ponto facultativo e feriado
municipal não cadastrados; **prazo em dobro** (o painel não o aplica: se o caso exigir, o advogado ajusta à mão);
prazos em horas, meses ou anos.

## Por que o resultado é sempre "A CONFERIR"

Perder prazo é falta grave. O sistema nunca é a única fonte: quem confere o prazo é o advogado, no processo e no
calendário do tribunal. Por isso o resultado:

1. traz o selo **A CONFERIR**, nunca "prazo fatal";
2. mostra as premissas e cada dia excluído com a fonte;
3. na dúvida entre duas datas, mostra a **mais curta** e a outra como alternativa;
4. marca **[VERIFICAR]** o que depende de decisão do advogado;
5. enquanto o advogado não validar `docs/juridico/REGRAS-DE-PRAZO.md`, mostra a faixa vermelha "Regras de cálculo ainda
   não validadas pelo advogado — não use para prazo real".

## Criar um prazo a partir do cálculo

O botão **Criar prazo com este cálculo** só abre o formulário de Prazos já preenchido (data, fundamento e fonte
"Calculadora de prazo"). **Nada é criado até você salvar**, e o prazo nasce **A CONFERIR** mesmo quando quem salva é
advogado: ele só passa a valer quando um advogado clicar em **Confirmar** na lista de Prazos.

## Calendário forense

Menu "Calendário forense" (grupo Sistema). Todos os papéis leem; **advogado e administrador** cadastram.
- **Sem fonte, não grava.** Cite a norma ou a portaria.
- Dia errado não se apaga: **Inativar** pede o motivo; a linha continua listada com o selo "Inativo".
- Ponto facultativo só conta como dia não útil se um **advogado** marcar, com a fonte do tribunal.
- Os calendários dos tribunais começam **vazios**. Enquanto não houver um, o resultado diz "Sem calendário verificado
  deste tribunal: considerados só os feriados nacionais".

## Ligar a validação (decisão do advogado)

Depois de revisar `docs/juridico/REGRAS-DE-PRAZO.md` e assinar a coluna "Validado por", o advogado define no ambiente
`PAINEL_CALCULADORA_REGRAS_VALIDADAS=true` e reinicia o painel. **Ausente ou `false`, o aviso vermelho aparece.**
