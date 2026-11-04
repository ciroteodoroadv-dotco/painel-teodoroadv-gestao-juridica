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

## Plano 2 — Pessoas e Processos (03/11/2026)
| Item | Estado |
|---|---|
| Estilo computado B x v2 (páginas e 7 drawers) | feito: zero diferença além das justificadas (`paridade/RELATORIO-2.md`) |
| Comparação texto a texto | feito: 19 ausências, todas justificadas |
| Envio de formulário em drawer (`data-drawer-form`) | exercitado por testes HTTP; **não exercitado por clique no Chrome** |
| Capturas 1440/400 e do B | **não feito** |
| Conferência do advogado na tela real | pendente; aprovar a omissão de "Nome do arquivo da ficha" |

## Plano 3 — Atendimentos e Tarefas (03/11/2026)
| Item | Estado |
|---|---|
| Estilo computado B x v2 (2 páginas e 5 drawers) | feito: zero diferença além das justificadas (`paridade/RELATORIO-3.md`) |
| Comparação texto a texto | feito: 19 ausências, todas justificadas |
| Envio de formulários, upload de PDF, exclusão com senha | exercitados por testes HTTP; **não por clique no Chrome** |
| "Novo cliente" dentro do atendimento (volta ao rascunho) | código pronto (`atendimento.js`); **não exercitado no Chrome** |
| Capturas 1440/400 e do B | **não feito** |
| Conferência do advogado na tela real | pendente |

## Plano 4 — Agenda, Audiências, Diligências (03/11/2026)
| Item | Estado |
|---|---|
| Estilo computado B x v2 (Agenda com 7 categorias, aba Audiências, 7 variantes de drawer) | feito: zero diferença além das justificadas (`paridade/RELATORIO-4.md`) |
| Comparação texto a texto | feito: 10 ausências, todas justificadas |
| JS novo no Chrome (acordeão, busca, mostra/oculta, filtro de opções, toast, decodificação do cabeçalho) | executado nos instantâneos |
| Envio real de formulário por clique, capturas 1440/400 | **não feito** (testes HTTP cobrem o servidor) |
| Conferência do advogado na tela real | pendente |

## Plano 5 — Intimações (DJEN) (03-04/11/2026)
| Item | Estado |
|---|---|
| Estilo computado B x v2 (categoria Intimações da Agenda: 23 seletores x 30 propriedades + rodapé Estado das coletas) | feito: zero diferença além da justificada (`paridade/RELATORIO-5.md`) |
| Comparação texto a texto (11-agenda.js e 16-sync-status.js) | feito: zero ausência do Plano 5 (restam as 8 do Plano 4) |
| JS novo no Chrome (`data-refresh` do rodapé: recarrega, mostra "indisponível" na falha) | executado nos instantâneos |
| Página Coleta DJEN e drawers da OAB | sem equivalente no B (só vocabulário visual); testes HTTP; **não comparados no Chrome** |
| Clique real em Confirmar/Descartar, capturas 1440/400 | **não feito** (testes HTTP cobrem o servidor) |
| Conferência do advogado na tela real e teste manual orientado antes de ligar a coleta | pendente |

## Plano 4C — Cancelar em vez de excluir; processo duplicado (04/11/2026)
| Item | Estado |
|---|---|
| Estilo computado B x v2 dos drawers "Cancelar …" e do bloco "PROCESSO DUPLICADO" (13 seletores) | feito: 3 diferenças, todas justificadas (`paridade/RELATORIO-4C.md`) |
| Cancelar compromisso/diligência/intimação, sem Excluir; tarefa sem exclusão | testes HTTP e enumeração de rotas |
| Verificação ao sair do campo do número (JS), clique real nos botões, capturas 1440/400 | **não feito no Chrome** |
| Conferência do advogado na tela real | pendente |

## Plano 6 — Monitor DataJud, emenda E1–E3 (04/11/2026)
| Item | Estado |
|---|---|
| Intimações com origens DataJud/DJEN (linhas e ordem) x B | feito no Chrome: idênticas (`paridade/RELATORIO-6.md`) |
| Rodapé com 3 linhas x B | feito: texto idêntico |
| Botão "Ver" + drawer "Intimação" (JS executado) | feito: abre, mostra os campos e fecha |
| Página Coleta DJEN (monitor e sigilosos), drawer do sigiloso | testes HTTP; texto lido no Chrome; **sem comparação de estilo** |
| Capturas 1440/400 | **não feito** (só 1568) |
| Teste manual orientado (chave, ligar com UM processo, conferir linha de base e formato real) | **pendente** — `docs/operacao/MONITOR-DATAJUD.md` |
| Conferência do advogado na tela real | pendente |
