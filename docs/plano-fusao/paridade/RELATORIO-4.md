# Relatório de paridade — Plano 4 (Agenda, Audiências, Diligências), 03/11/2026

## O que foi feito
- Leitura integral de `11-agenda.js`, `05-dominio.js`, `07-app-core.js`, `06-helpers-vinculos.js` e das regras do servidor do B (`agenda/agenda.go`, `agenda/http.go`, `audiencias/http.go`, `diligencias/http.go`, `ficha/audiencia.go`, `ficha/diligencia.go`). `08-tarefas.js`, `03`, `15`, `02` e `painel.css` foram lidos só nos trechos usados (mapeadores, `todayISO`/`fmtDate`, toast, classes da Agenda).
- Textos: `python3 tools/refb/textos4.py` — 121 textos de `11-agenda.js` conferidos; **10 ausentes, todos justificados**: intimações (Plano 5: "Descartar", "Falha ao atualizar a intimação."); ficha em arquivo ("estes campos não existiam na ficha:" x2); recarregar listas (mensagens na tabela de `forms.js`, 3); "Falha ao cancelar a audiência." (o v2 usa "Não foi possível cancelar a audiência." e mensagens específicas); "Tipo do ato *" (composto por `{{if .Novo}}` no template).
- Fixtures do B com a mesma massa do seed do v2: `tools/refb/agenda.py`.
- Testes de textos/ordem dos drawers, contadores, 7 pílulas, conflito, cancelamento, RBAC e CSP: `internal/app/{agenda,agenda_pagina,audiencias,diligencias}_test.go`.

## Tabela de paridade (seletores/propriedades)
**Não medida nesta rodada.** Não foi feita a comparação de estilo computado B × v2 no Chrome (nem capturas). A estrutura usa as mesmas classes do B (`.acc-heads`, `.acc-head`, `.acc-item`, `.agenda-table-scroll`, `.agenda-tarefas`, `.badge-*`, `.row-actions`, `.empty`, `.form-block`, `.field`, `.row2`, `.hint`, `.drawer-actions`, `#buscaAgenda`, `.agenda-date-chip`), mas diferenças de CSS são desconhecidas. Diferenças já conhecidas por construção: um `.empty[hidden]` por tabela (filtro ao digitar); `style=` do B trocado por `.hint-tabela`/`.hint-bloco`; botões de escrita só para quem pode escrever.

## Não verificado
Qualquer coisa no navegador: JS novo (`data-oculta-quando`, `data-filtra-opcoes`, `data-toast-erro`, contagem somada, decodificação do toast) só foi lido, não executado (sem Node; sem Chrome nesta rodada); capturas 1440/400; conferência do advogado.
