# Relatório de paridade — Plano 3 (Atendimentos e Tarefas), 03/11/2026

Método: B servido por `tools/refb/servidor.py` (porta 18787, dados de `atendimentos_tarefas.py`, iguais ao seed do v2) e v2 por instantâneos do seed fictício (`PARIDADE_OUT`, porta 18790). Mesmo Chrome, estilo computado (21 propriedades + largura/altura) via `javascript_tool`, somente leitura. B: `App.setTab`, `App.openTarefaForm`, `openTarefaEditForm`, `deleteTarefa`, `openAtendimentoForm`, `openAtendimentoEditForm`. v2: o fragmento do `data-drawer` aberto por `App.openDrawer`.

## Telas medidas
Tarefas (página com 2 painéis abertos, 22 seletores), Atendimentos (página, 11 seletores), drawers "Nova tarefa", "Editar tarefa", "Excluir tarefa", "Novo atendimento", "Editar atendimento" (31 seletores cada).

## Correções feitas pela medição (compat.css)
`.row-actions select` (o B usa o select cru do navegador), campos de senha/arquivo/radio sem a regra genérica, `label.sr-only`, `white-space` da tabela de atendimentos (mantendo `pre-wrap` do objeto e `nowrap` do código).

## Diferenças restantes (todas justificadas)
| Seletor / propriedade | B | v2 | Justificativa |
|---|---|---|---|
| `.tarefa-info .hint` (contagem) | 2 | 3 | linha "Tarefa vinculada a: TAR. …" da tarefa-filha (decisão do plano) |
| `.tarefa-info > .badge-danger` (contagem) | 3 | 2 | o prazo fatal "A CONFERIR" fica dentro de `.tarefa-fatal` junto com o selo (regra de ouro) |
| Larguras de `th`/`td` (~3 px) | — | — | barra de rolagem da página |
| Drawer Excluir: largura do botão primário | 174 px ("Excluir definitivamente") | 75 px ("Excluir") | texto adaptado (reautenticação, exclusão lógica) |

Fora isso: **zero diferença** nas propriedades medidas, nas 5 telas de drawer e nas 2 páginas.

## Texto a texto (`python3 tools/refb/textos3.py`)
137 textos de `10-atendimentos.js`, `08-tarefas.js`, `06-helpers-vinculos.js` e das listas de domínio; **19 ausentes, todos justificados**:
- "Editar atendimento", "Salvar atendimento": compostos por `{{if .Novo}}` no template (presentes, cobertos por teste) — 2
- "Excluir definitivamente", sua frase e "Senha de administrador *" (reautenticação do usuário) — 3
- "…do caso e convertido para .md.", "Documento guardado e convertido:" (conversão em .md não existe) — 2
- "Falha ao criar a tarefa:" (2x; o v2 mostra a mensagem específica ou "Falha ao criar a tarefa.") — 2
- "Já existe uma tarefa com esse nome de arquivo…", "…campos não existiam na ficha:" (não há ficha em arquivo) — 2
- "Papel processual", "É cliente do escritório" (são do repetidor de Processo, Plano 2) — 2
- "Campos liberados para edição.", "Informe a senha de ADM." (função `liberarCamposAdm`, sem uso nas telas) — 2
- Opções `Triagem`, `Em andamento`, `Concluído`, `Convertido em processo` (tipo/status não têm campo na tela; ficam só no esquema) — 4

## Não verificado
Envio real dos formulários por clique no Chrome (o instantâneo é estático; o envio é coberto por testes HTTP), o fluxo "Novo cliente" dentro do atendimento no navegador (só lido e coberto por testes do HTML), upload de PDF por clique, capturas em 1440/400 px, conferência do advogado na tela real.
