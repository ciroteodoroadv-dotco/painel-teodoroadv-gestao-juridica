# Relatório de paridade — Plano 7 (Calculadora de prazo e Calendário forense), 04/11/2026

## Método
B servido por `tools/refb/servidor.py` (porta 18787, só leitura) no Chrome; v2 servido por `TestServirParaConferenciaManual` (app inteiro, banco
descartável, demonstração fictícia, `web/static` real por proxy). Mesmo Chrome. Estilo computado de 10 elementos do drawer nos dois (`font-size`, peso,
cor, fundo, borda, raio, paddings, margem, display, gap, largura). Nenhum dado real; sem a porta 18080.

## Resultado
| Item | B | v2 | Justificativa |
|---|---|---|---|
| Botão na sidebar, depois de Audiências | botão | link `data-drawer` na mesma posição | v2 usa links; mesmo visual |
| Drawer: `h2`, subtítulo, rótulos, dicas, radios, "Número de dias", `.drawer-actions`, botões | — | rótulos e ordem **idênticos** (teste `TestCalculadoraDrawerComOsRotulosDoB`); estilo computado idêntico em 9 de 10 elementos | — |
| `input[type=number]` | nativo (178 px, borda 2px) | corrigido para o mesmo (era 423 px estilizado) | `compat.css` estilizava todo input; neutralizado |
| Campo "Calendário" | não existe | novo, depois de "Número de dias", com dica "Extra do v2" | extra do plano |
| Resultado: "Prazo fatal: dd/mm/aaaa" | `.form-block` com "Prazo fatal" | "Vencimento calculado: dd/mm/aaaa" + selo A CONFERIR + premissas + aviso | regra do v2 (nunca "Prazo fatal" sem ressalva) |
| Frases "faltam/vence hoje/venceu há" | sim | iguais | — |
| Erro | "Número de dias inválido." / "Falha ao calcular o prazo." (perdia o motivo) | mesmas frases + a mensagem do motivo do servidor | correção do bug do B |
| Faixa vermelha "Regras de cálculo ainda não validadas…" | não existe | no topo do resultado enquanto `PAINEL_CALCULADORA_REGRAS_VALIDADAS` não for `true` | regra do plano |
| Botão "Criar prazo com este cálculo" | não existe | `btn btn-ghost` no resultado | plano |
| Página Calendário forense | não existe | nova, grupo "Sistema", vocabulário visual do B | plano |

## Textos
`python3 tools/refb/textos7.py`: 26 textos de `12-prazo.js` e do botão da sidebar; 1 ausência, **justificada**: "Prazo fatal: " (substituído por
"Vencimento calculado:" + A CONFERIR).

## JS executado no Chrome
- Clique no botão da sidebar: abre o drawer; `htmx.process` ativou o `hx-post`; "Calcular" trocou **só** `#resultadoPrazo` (a página não navegou):
  15 dias úteis, Diário, evento 06/11/2026 → "Vencimento calculado: 01/12/2026", A CONFERIR, "faltam 18 dia(s) útil(eis)" (hoje 04/11/2026).
- Dias = 0: resposta 400 exibida no bloco vermelho com "Número de dias inválido." (o `htmx:beforeSwap` de `calculadora.js` funcionou).
- Regra material penal: novo resultado trocado no mesmo lugar.
- Calendário "TRIBUNAL DE TESTE" (Cinzas 10/02/2027 cadastrada): ciência 09/02/2027, 1 dia útil → 11/02/2027, com a portaria fictícia nas premissas.
- "Criar prazo com este cálculo": abriu `/prazos/novo` com data 2027-02-11, fonte "calculadora" e fundamento preenchidos; nada foi criado.
- Calendário forense: "Novo dia" (drawer, salvar, toast, linha nova), "Inativar" (motivo, toast, linha com selo "Inativo").
Abas criadas por mim foram fechadas.

## Não verificado
Capturas em 400 px e 1440 px da calculadora e do calendário (só 1568); estilo computado do bloco de resultado e da página Calendário (B não tem);
validação jurídica das regras; fontes legais não lidas (ver `NOTAS-PLANO-7.md` item 2); comportamento com dados reais; leitura por leitor de tela; navegadores
além do Chrome; o app dev (`painel-app-1`) **não foi reconstruído** nesta rodada.
