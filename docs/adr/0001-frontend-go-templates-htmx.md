# ADR 0001 — Frontend com Go templates + htmx

**Status:** aceita (11/2026)

**Contexto:** o sistema precisa ser simples, acessível e mantido por pessoa leiga, com pouca superfície de ataque.

**Decisão:** HTML renderizado no servidor com `html/template`; htmx versão fixada, servido do próprio app.

**Consequências:** um só binário, sem Node nem build JS; CSRF e sessão simples; CSP estrita viável.
Interatividade rica (arrastar, gráficos) exigirá ilhas de JS pontuais.

**Alternativa rejeitada:** SPA React/Svelte — dobra a pilha, o CI e a superfície de dependências.
