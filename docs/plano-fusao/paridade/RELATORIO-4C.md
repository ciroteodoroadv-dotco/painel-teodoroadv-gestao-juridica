# Relatório de paridade — Plano 4C (cancelar em vez de excluir; processo duplicado), 04/11/2026

## Método
Igual ao do Plano 5: B por `tools/refb/servidor.py` (18787) e v2 por instantâneos (`PARIDADE_OUT`, agora com `agenda/{id}/cancelar`, `diligencias/{id}/cancelar`,
`intimacoes/{id}/cancelar` e `duplicado-bloco.html`). Estilo computado (21 propriedades) via `javascript_tool`, somente leitura. Referência no B: o drawer real
"Cancelar audiência" (`App.cancelarAudiencia`). O B não tem o bloco "PROCESSO DUPLICADO"; o mesmo markup foi montado dentro do drawer do B (que usa o mesmo
`painel.css`) para medir os componentes `.form-block`, `.badge-danger`, `.field`, `textarea`.

## Resultado (13 seletores)
`h2`, `.drawer-sub`, `.field`, `label`, `textarea`, `.drawer-actions`, `.btn-ghost`, `.btn-primary` do drawer de cancelamento e `.form-block`, `.badge-danger`, `p`, `label`,
`textarea` do bloco: **zero diferença de estilo além de 3, todas justificadas**:

| Item | B | v2 | Justificativa |
|---|---|---|---|
| `.drawer-sub` altura | 36,3 px (2 linhas) | 18,1 px (1 linha) | o texto do v2 é mais curto ("O registro é preservado com o motivo do cancelamento."), sem a parte da senha |
| `.btn-primary` largura | 118,5 px | 143,8 px | rótulo "Cancelar compromisso" (mais longo que "Cancelar audiência") |
| `textarea` dentro de `.form-block`: cor | cinza (herda do bloco) | escuro | o v2 fixa a cor do texto digitado em `.field textarea` (legibilidade); fora do bloco é idêntico ao B |

Diferenças de conteúdo por determinação do advogado: "Excluir"/"Ainda não implementado…" do compromisso e "Descartar" da intimação viraram "Cancelar" (drawer com
motivo); status "Descartada" virou "Cancelada"; tarefa sem botão "Excluir".

## Não verificado
Clique real nos botões Cancelar e envio dos drawers no Chrome (testes HTTP cobrem o servidor); a verificação ao sair do campo (`data-verifica-duplicado`) foi
coberta por testes do endpoint e inspeção do JS, **não executada no Chrome**; capturas 1440/400; conferência do advogado na tela real.
