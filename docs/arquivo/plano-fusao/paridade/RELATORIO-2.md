# Relatório de paridade — Plano 2 (Pessoas e Processos), 03/11/2026

Método: B servido por `tools/refb/servidor.py` (porta 18787, dados fictícios de `pessoas_processos.py`) e v2 por instantâneos do seed fictício (`PARIDADE_OUT`, porta 18790). Mesma janela do Chrome (1920x945), estilo computado de 18 seletores de página e 21 de drawer (24 propriedades) via `javascript_tool`, somente leitura. Acordeões abertos nos dois lados; drawers abertos pelos botões reais (B: `App.open*`; v2: `data-drawer`).

## Telas medidas
Pessoas (página; drawer "Nova pessoa"; drawer "Editar pessoa" PF), Processos (página; "Novo processo"; "Editar processo"; "Mover estado"; "Registrar andamento"; "Atualizar financeiro"; "Próxima providência").

## Diferenças restantes (todas justificadas)
| Seletor / propriedade | B | v2 | Justificativa |
|---|---|---|---|
| `.empty` (n, display, width, height) nas páginas | 1 aviso visível | 4 avisos, 3 com `hidden` | o v2 deixa um aviso oculto por tabela para o filtro ao digitar mostrar "Nenhuma … nesta categoria." sem recarregar; invisível |
| Drawer "Nova pessoa": `.field`, `label`, `input`, `.hint` (25/27/19/2 contra 24/26/18/1) | com "Nome do arquivo da ficha" | omitido | única omissão de campo (NOTAS-PLANO-2, item 1) |
| Drawer "Novo processo": `.field`/`label`/`input` (24/27/8 contra 23/26/7) e 168 caracteres de texto | idem | idem | idem |
| Larguras de `.acc`, tabela etc. na página de Processos (1619 contra 1634 px) | com barra de rolagem | sem | a página do B ficou mais alta; não é regra de CSS (ignorado na comparação) |

Fora isso: **zero diferença** nas propriedades medidas. Correções feitas por causa da medição: `.search` 240px, quebra de linha nas células, `td.row-actions` nowrap, rótulo "É cliente do escritório" sem negrito/margem (compat.css).

## Texto a texto (`python3 tools/refb/textos.py`)
248 textos extraídos de `14-pessoas.js`/`09-processos.js` (rótulos, legendas, h2/h3, subtítulos, dicas, placeholders, botões, toasts, confirmações) e das listas (`OPCOES`, `PAPEIS_PROCESSUAIS`, `SISTEMAS_PROCESSO`, `INSTANCIAS_PROCESSO`) contra templates e Go do v2: **19 ausentes, todos justificados**:
- Nome do arquivo da ficha, sua dica e placeholder, "Só letras minúsculas, números e hífen." (omissão, item 1) — 5
- "Excluir definitivamente" e a mensagem de exclusão (substituídas por segurança, item 2) — 2
- Confirmação de CEP ("Endereço pelo CEP", "Há campos…", "Substituir") e "Falha ao consultar o CEP." (Plano 8) — 4
- "Nenhuma pessoa cadastrada." (ramo inalcançável no B: o acordeão só chama a tabela com lista não vazia) — 1
- "Esta pessoa não está referenciada em nenhum processo…" (no v2 abre a confirmação de exclusão, decisão do plano) — 1
- Toasts ligados à ficha em arquivo ("já existe … nome de arquivo", "campos não existiam na ficha", 2x) — 4
- "…vinculada ao rascunho do atendimento" (Plano 3) — 1
- "Ação registrada, mas houve falha ao recarregar a lista de pessoas/processos." — presentes em `forms.js` numa tabela de mensagens que o extrator (só `showToast(` literal) não alcança — 2 (conferido à mão)

Capturas: `V2-processos-drawer-novo.jpg` (v2, drawer Novo processo). Capturas do B e em 400 px **não feitas** (a janela do Chrome não redimensiona; a comparação é por estilo computado).

## Errata (conferência independente de 04/11/2026)

- A contagem de textos ausentes está errada neste relatório: o script atual (`textos.py`) lista **18** ausências no commit; a lista acima soma 20 e cita itens que o script atual não lista. Valem as 18 do script.
- A captura `V2-processos-drawer-novo.jpg` **não existe** no repositório (só há 3 imagens em `paridade/`). Não há captura de Pessoas/Processos versionada.
