# Relatório de paridade — Plano 5 (Intimações / DJEN), 04/11/2026

## Método
Igual ao do Plano 4: B servido por `tools/refb/servidor.py` (porta 18787) com as fixtures de `tools/refb/intimacoes.py` (a mesma massa do seed do v2:
1 OAB, 3 intimações — triagem, confirmada, descartada —, 3 tarefas de análise, 2 coletas: 1 sucesso e 1 falha de exemplo) e o v2 por instantâneos
(`PARIDADE_OUT`, servidor estático com o cabeçalho de fragmento). Mesmo Chrome; estilo computado (30 propriedades + largura/altura) de 23 seletores
somente-leitura via `javascript_tool`; resultado do B levado ao v2 em `window.name`. Nenhuma requisição a serviço externo; nenhum dado real.

## Tabela de paridade (medida)
Seletores: `.acc-item.open`, `.acc-body`, `table`, `th`, `th:last-child`, `td`, `td.tabular`, `.badge-warn/-ok/-neutral`, `td strong`, `td .hint`,
`td.row-actions`, `.btn-text`, `.btn-danger-text`, `.acc-head.open` e seu `.count`, `.sync-status`, `strong`, `p`, `ul`, `li`, `li.sync-error`.

Resultado final: **zero diferença de estilo**. Antes do ajuste havia 4: `td.row-actions` 41,1 x 41,8 px e `.btn-text`/`.btn-danger-text` `display:block` (B, itens
flex) x `inline` (v2, botões dentro de `<form>`); corrigido com `.row-actions > .form-inline{display:contents}` em `compat.css`.

| Item | B | v2 | Justificativa |
|---|---|---|---|
| Linhas da tabela (texto) | 3 linhas | idênticas, na mesma ordem | - |
| Rodapé (texto) | "Estado das coletas … DJEN: última coleta concluída. Tentativa: 03/11/2026, 12:00:00 (Brasília) …" | idêntico (inclui a vírgula do `toLocaleString`) | corrigido após a medição |
| Coluna "Tarefa de análise" | título + chave da pasta | idêntico (`a-confirmar`) | decisão "B prevalece" (NOTAS-PLANO-5 item 7) |
| Tarefas (Agenda): linhas TAR. 7 a 9 | 8 linhas | 8 linhas, texto igual | a linha "Conferir intimação (exemplo)" difere por trazer o selo A CONFERIR do v2 (Plano 3/4) |
| Origem da tarefa na página Tarefas | hash da intimação | "DJEN" | legibilidade; o hash é interno |
| Botões Confirmar/Descartar | `<button>` soltos | `<form data-drawer-form>` + `<button>` | CSRF e feedback pelo servidor |
| Página Coleta DJEN, drawers da OAB | não existe | nova | vocabulário visual do B (`topbar`, `.field`, `.row2`, tabela, `.badge`, drawer) |

## Textos
`python3 tools/refb/textos5.py`: 129 textos de `11-agenda.js` e `16-sync-status.js`; ausências: 8, **todas do Plano 4** (já justificadas em
NOTAS-PLANO-4/`RELATORIO-4.md`). Zero ausência de Intimações ou do rodapé.

## JS novo executado no Chrome
`App.atualizarSync` (`data-refresh`): trocado o rodapé por "ADULTERADO" e recarregado do servidor de instantâneos, voltou idêntico ao original; com caminho
inexistente mostrou "Estado das coletas indisponível — Não foi possível ler o histórico local…"; restaurado. O recarregamento a cada 60 s e ao voltar à aba
usa as mesmas funções (não esperei 60 s).

## Não verificado
Clique real em Confirmar/Descartar no Chrome (servidor coberto por testes HTTP); capturas 1440/400; página Coleta DJEN e seus drawers no Chrome;
conferência do advogado na tela real; o swagger do DJEN (HTTP 403 ao WebFetch) e qualquer consulta real ao DJEN.
