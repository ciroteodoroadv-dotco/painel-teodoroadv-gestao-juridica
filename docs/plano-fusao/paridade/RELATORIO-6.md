# Relatório de paridade — Plano 6 (Monitor DataJud), 04/11/2026

## Método
B servido por `tools/refb/servidor.py` (porta 18787) com as fixtures de `tools/refb/intimacoes.py` (mesma massa do seed do v2: as 3 intimações do DJEN + 2
movimentações do DataJud do processo PRO. 0000001, tarefas TAR. 10 a 12 e os 3 módulos do rodapé) e o v2 por instantâneos (`PARIDADE_OUT`, servidor
estático). Mesmo Chrome; leitura do texto das linhas da tabela de Intimações e do rodapé nos dois, comparados por `window.name`. Nenhuma requisição a
serviço externo; nenhum dado real. Correção durante a medição: o B liga tarefa e intimação por `origemTipo == origem` (a fixture usava "processo" e o B
mostrava "Tarefa não localizada"); a massa de referência foi ajustada ("DataJud"), e a contagem "sem registro / sem número" do rodapé também.

## Resultado
| Item | B | v2 | Justificativa |
|---|---|---|---|
| Intimações: 5 linhas (origem, processo, data, status, tarefa de análise) | 2 linhas DataJud + 3 DJEN | idênticas, mesma ordem | única diferença: "Descartada" (B) x "Cancelada" (v2) — Plano 4C, determinação do advogado |
| Rodapé "Estado das coletas" | 3 linhas (DJEN, DataJud, DJEN por processo sigiloso) | texto idêntico (777 caracteres), `sync-error` nas 2 fora de sucesso | - |
| Botão "Ver" e drawer "Intimação" | não existem | novos (E2) | pedido da emenda; usam o mecanismo de drawer existente |
| Coluna de ações | Confirmar / Descartar | Ver / Confirmar / Cancelar | idem |
| Página Coleta DJEN: seções do monitor e dos sigilosos | não existe | nova | vocabulário visual do B; texto lido no Chrome |
| Prazo interno nas tarefas de análise (selo A CONFERIR; aparece na lista de prazos da Agenda) | sem prazo | prazo fatal interno de 2 dias, "a conferir" | E3 (B na forma, v2 na regra) |
| Origem da tarefa na página Tarefas | origem + hash | "PRO. 0000001 · DataJud" | legibilidade |

## Textos
`python3 tools/refb/textos6.py`: 129 textos de `11-agenda.js` e `16-sync-status.js`; 6 ausências, **todas já justificadas** (5 do Plano 4: audiências/diligências; "Descartar"
virou "Cancelar" no Plano 4C). Zero ausência do rodapé.

## JS executado no Chrome
Clique em "Ver" (linha DataJud): `drawer.js` buscou `/intimacoes/{id}/detalhe`, abriu o painel de 480 px com os 8 campos, o aviso da fonte e o texto
completo; "Fechar" (`data-drawer-fechar`) fechou (largura 0). Aba criada por mim, fechada.

## Não verificado
Estilo computado seletor a seletor (só texto e estrutura nesta rodada; `.texto-integral`/`.aviso-fonte` são CSS novo, vistos em captura 1568 px, **sem** captura a 400 px);
clique real em Confirmar/Cancelar no Chrome; botão "Executar monitor agora" e drawer do sigiloso no Chrome (testes HTTP cobrem); qualquer consulta real ao DataJud/DJEN;
documentação oficial do DataJud (HTTP 403, não contornado).
