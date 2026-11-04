# ADR 0012 — Prazo: o sistema detecta, sugere e o advogado confirma

**Status:** aceita em 04/11/2026 (Planos 5, 6 e 7).

**Contexto:** erro de prazo processual custa o direito do cliente e a responsabilidade do advogado. Fonte externa (DJEN, DataJud) não é tempo real e pode estar incompleta;
a contagem depende de regra legal e de calendário do tribunal.

**Decisão:**

1. **O sistema detecta** (nova intimação ou movimentação) e **cria a tarefa de análise** em "A confirmar", com o texto integral da comunicação quando a fonte o dá.
2. **Prazo interno de 2 dias corridos**: a tarefa nasce com prazo fatal interno (hoje em São Paulo + 2), marcado como interno, para a análise não ficar esquecida.
3. **O prazo registrado nasce "A CONFERIR"**, em todas as origens (digitado, vindo de tarefa, da intimação ou da Calculadora). Só o **advogado** confirma, cumpre ou cancela.
4. **A Calculadora de prazo sempre devolve "A CONFERIR"**: regra CPC/CPP/material, calendário do tribunal cadastrado com fonte, resultado mais curto quando há dúvida, lista do que **não** foi considerado
   (inclusive prazo em dobro, que o painel não aplica por determinação do advogado). Enquanto `PAINEL_CALCULADORA_REGRAS_VALIDADAS` não for ligada, avisa que as regras não foram validadas.
5. O sistema **nunca** calcula, confirma ou protocola por conta própria; o painel de prazos é apoio, a fonte oficial manda.

**Consequências:** o selo "A CONFERIR" convive com o prazo na mesma tela; confirmar prazo exige permissão `prazos:confirmar` (só advogado); regras legais não lidas ficam marcadas `[VERIFICAR]`
em `docs/juridico/REGRAS-DE-PRAZO.md` até o advogado validar.
