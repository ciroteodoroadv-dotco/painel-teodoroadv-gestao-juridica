# Aceite visual — Plano 1 (03/11/2026)

| Item | Estado |
|---|---|
| CSS e logo do B copiados com hash idêntico (HASHES-B.txt) | feito |
| Tela de login com o tema B, sem erro de CSP | conferido no navegador |
| Casca logada (sidebar, rodapé), drawer sem ESC/clique fora, toast | **pendente**: exige login do advogado (não uso credenciais reais) |
| Comparação tela a tela com o painel B | pendente: por leitura do código do B (sem extrair dados) + conferência do advogado |

## Plano 1B (03/11/2026)

| Item | Estado |
|---|---|
| Esqueleto do B (`#app`, sidebar, topbar, stats, content, syncStatus, idxTicker, drawer, toast) em toda tela logada | feito, coberto por `internal/app/shell_test.go` |
| Paridade de estilo computado com o B (35 seletores x 20 propriedades) | feito: só restam diferenças de conteúdo (ver `paridade/RELATORIO-1B.md`) |
| Drawer (pilha, sem tecla, sem clique fora), acordeão, máscaras, linhas repetíveis, toast | conferido no Chrome em `componentes.html` |
| Capturas 1440x900 e 400x800; Processos/Pessoas/Atendimentos/Audiências do B | **não feito** (janela não redimensiona) |
| Envio de formulário em drawer (`data-drawer-form`) | código pronto, não exercitado (Plano 2) |
| Conferência do advogado na tela real (login dele) | pendente |
