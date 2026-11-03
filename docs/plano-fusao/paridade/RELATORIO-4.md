# Relatório de paridade — Plano 4 (Agenda, Audiências, Diligências), 03/11/2026

## O que foi feito
- Leitura integral de `11-agenda.js`, `05-dominio.js`, `07-app-core.js`, `06-helpers-vinculos.js` e das regras do servidor do B (`agenda/agenda.go`, `agenda/http.go`, `audiencias/http.go`, `diligencias/http.go`, `ficha/audiencia.go`, `ficha/diligencia.go`). `08-tarefas.js`, `03`, `15`, `02` e `painel.css` foram lidos só nos trechos usados (mapeadores, `todayISO`/`fmtDate`, toast, classes da Agenda).
- Textos: `python3 tools/refb/textos4.py` — 121 textos de `11-agenda.js` conferidos; **10 ausentes, todos justificados**: intimações (Plano 5: "Descartar", "Falha ao atualizar a intimação."); ficha em arquivo ("estes campos não existiam na ficha:" x2); recarregar listas (mensagens na tabela de `forms.js`, 3); "Falha ao cancelar a audiência." (o v2 usa "Não foi possível cancelar a audiência." e mensagens específicas); "Tipo do ato *" (composto por `{{if .Novo}}` no template).
- Fixtures do B com a mesma massa do seed do v2: `tools/refb/agenda.py`.
- Testes de textos/ordem dos drawers, contadores, 7 pílulas, conflito, cancelamento, RBAC e CSP: `internal/app/{agenda,agenda_pagina,audiencias,diligencias}_test.go`.

## Método da paridade e do JS (Chrome, 03/11/2026)
B servido por `tools/refb/servidor.py` (porta 18787, fixtures `agenda.py`, idênticas ao seed) e v2 por instantâneos do seed fictício (`PARIDADE_OUT`, servidos por `python3 -m http.server`; sem login, sem dado real). Mesmo Chrome e janela; estilo computado (19 a 24 propriedades) via `javascript_tool`, somente leitura; resultado do B levado ao v2 em `window.name` (mesmo site) e comparado propriedade a propriedade. Os instantâneos do v2 agora incluem `audiencias.html` e os fragmentos `agenda/novo`, `agenda/{id}/editar`, `audiencias/nova`, `audiencias/{id}/editar`, `audiencias/{id}/cancelar`, `diligencias/{id}/editar`.

## Tabela de paridade (medida)
Telas: Agenda (7 categorias abertas uma a uma: pílula aberta/fechada, contador, busca, painel, tabelas, th/td, selos, hints, botões, `.empty`, `.form-block`), aba Audiências (2 tipos abertos ao mesmo tempo), drawers Novo compromisso (também com tipo Diligência), Editar compromisso, Nova audiência, Editar audiência, Cancelar audiência, Editar diligência (18 seletores x 23 propriedades cada).

Resultado: **zero diferença de estilo não justificada**. Diferenças restantes:
| Item | B | v2 | Justificativa |
|---|---|---|---|
| Intimações: conteúdo | tabela com 2 linhas das fixtures | `.form-block` "Intimações indisponíveis." e contador 0 | Plano 5 ainda não existe; o v2 nunca mostra dado inventado |
| `.acc-body`/tabela de Tarefas: largura 672 x 657 px e altura +19 px | sem barra de rolagem | com barra de rolagem | a lista de prazos do v2 no topo alonga a página (regra do v2); a tabela quebra uma linha a mais |
| `.empty` | 1 por painel | 1 oculto por tabela (`hidden`) | aviso do filtro ao digitar |
| Drawers: +1 `input` (hidden `version`) | - | sim | concorrência otimista (regra do v2) |
| Drawers: nomes `atendimentoVinculado`/`criadoPor` x `atendimento`/`criado_por` | - | - | só o atributo name; mesma visibilidade |
| Drawers Diligência: `input[disabled]` | só o select é desabilitado | também o campo de busca do vínculo oculto | efeito idêntico (campo oculto) |
| Ordem das pessoas nos selects | ordem de cadastro | ordem alfabética | ordem do v2 desde o Plano 2 |
| Número do processo de uma audiência | igual | igual | fixture corrigida para o dígito verificador do v2 |

Textos dos drawers: mesmos itens, na mesma quantidade, nas 8 variantes (diferem só na ordem dos nomes de pessoa acima).

## JS novo executado no Chrome (instantâneos, sem login)
- Acordeão da Agenda abre um painel por vez; clicar no aberto fecha; Audiências abre vários: OK.
- Busca ao digitar (`data-filtro`): "AULA", "reuniao" (sem acento) e "xyzzy" ajustam os contadores das 7 pílulas (somando as duas tabelas do painel) e mostram os avisos de vazio; limpar restaura: OK.
- `data-mostra-quando`/`data-oculta-quando`: tipo Diligência mostra "Responsável pela diligência" e oculta/desabilita Atendimento vinculado e Criado por; voltar a Reunião desfaz: OK.
- `data-filtra-opcoes`: "ana" esconde tudo menos "Sem vínculo"; "pes. 0000004" deixa 1 pessoa; limpar restaura as 5 opções: OK.
- `data-toast-erro` ("Excluir"): toast de erro "Ainda não implementado nesta fase do painel.": OK.
- Decodificação de `X-Painel-Ok`: `App.textoDoCabecalho` devolve "Tarefa excluída." e "Audiência cancelada; a ficha foi preservada."; o envio do drawer "Cancelar audiência" com `fetch` simulado (resposta com o cabeçalho em %XX) fechou o drawer e mostrou o toast com acentos: OK.

## Não verificado
Capturas 1440/400; envio real de formulário por clique contra o servidor (coberto por testes HTTP; no Chrome só com resposta simulada); aba Audiências em 400 px; conferência do advogado na tela real.
