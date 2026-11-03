# Paridade da casca — painel B × v2 (Plano 1B, 03/11/2026)

Método: o mesmo `tools/refb/paridade.js` (estilo computado, 20 propriedades por seletor) rodou no painel B
(servidor de referência, porta 18787, dados fictícios, Agenda com "Aulas" aberta e drawer "Novo compromisso" aberto)
e na página `componentes.html` do v2 (instantâneo do harness, seed fictício, porta 18788; mesma casca real do v2 com o
miolo do B: acordeão, drawer, máscaras). Mesma aba do Chrome, mesmo viewport (1920x945; a janela não aceitou
redimensionar para 1440x900).

Resultado: 35 seletores comparados, 20 propriedades cada. Após os ajustes, 10 diferenças, todas de **conteúdo**, nenhuma de estilo.

| Seletor / propriedade | B | v2 | Decisão |
|---|---|---|---|
| `.title-row`, `.topbar .sub` / width | 382px | 460px | Texto do subtítulo diferente (o do B descreve categorias que o v2 ainda não tem). Largura do item flex depende do texto. Justificado. |
| `#content` / height | 272px | 463px | Conteúdo do instantâneo diferente. |
| `th` / width | 110px | auto | Regra `.agenda-tarefas th:first-child` do B se aplica à tabela da Agenda do B; o v2 não tem essa tabela até o Plano 4. |
| `.badge` / presença | existe | não existe | O fixture de componentes não tem selo; selos existem nas telas reais do v2 (`.badge-warn`, `.badge-danger`). |
| `.sync-status` / height | 100px | 73px | O B lista fontes (JS 16); o v2 mostra o texto honesto de "nenhuma coleta configurada" (Tarefa 4). Regra de segurança. |
| `.drawer h2`, `.field label`, `.field input`, `.drawer-actions` / width | 408px | 423px | O drawer do B tem barra de rolagem vertical (formulário longo) que subtrai 15px; o fragmento fictício do v2 é curto. |
| `.empty` | ausente nos dois | ausente | Não comparado (nenhum estado vazio nas telas usadas). |

Diferenças que existiam e foram **corrigidas** (compat.css): itens do menu (`a` com `box-sizing:content-box`, como o `button` com `all:unset` do B: largura 174px, altura 19,56px), `h2` do drawer em negrito 700 e rótulo `.field label` sem margem superior.

Itens do menu, item ativo, contador, topbar, h1, chip de data, botão primário, `.stats`, `.acc-head` (aberto e fechado), contador do acordeão, tabela/th/td,
campo de busca, `.idx-ticker`, `.idxt-status`, `#drawer`, `.toast`, fonte, cores, raios, sombras: **zero diferenças**.

## Comportamento verificado no navegador (v2, `componentes.html`)
Drawer abre, `h2` vira `#drawerTitle`; ESC e clique fora não fecham; `data-drawer-sobrepor` empilha (pilha=1) e `data-drawer-fechar` desempilha; Cancelar fecha e devolve o foco ao botão de origem;
acordeão do grupo `agenda` abre um por vez; máscara de CPF `12345678909` → `123.456.789-09`; `data-validate="cpf"` recusa `111.111.111-11`; "Outro" revela/oculta o campo com `hidden`; linhas repetíveis adicionam/removem; filtro de tabela sem acento; toast some em 3,2 s com classe `error`.
`maskCPF/CNPJ/CEP/Phone/CNJ/Currency` e `isValidCPF/CNPJ` conferidas com valores conhecidos (todas verdadeiras). Sem erros no console.

## Não verificado
- Capturas nas larguras 1440x900 e 400x800 (a janela não redimensiona; só há capturas do viewport ~1920x945). Anexadas: `B-agenda-drawer-novo-compromisso.jpg`, `V2-agenda-inicial.jpg`, `V2-tarefas.jpg`. Largura de celular não foi vista.
- Telas Processos, Pessoas, Atendimentos, Audiências do B e drawer "Nova tarefa" do B não foram capturadas; paridade de estilo foi medida só na Agenda do B.
- Envio de formulário em drawer (`data-drawer-form`, `X-Painel-Ok`): código escrito, não exercitado (não há rota que o use ainda; `http.server` não aceita POST). Fica para o Plano 2.
- O v2 em execução (porta 18080) não foi aberto (sem credenciais reais, por regra).
