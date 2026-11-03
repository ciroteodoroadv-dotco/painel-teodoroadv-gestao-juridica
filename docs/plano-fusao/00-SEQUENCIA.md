# Fusão — sequência dos planos (revisada em 03/11/2026)

**Determinação do advogado (03/11/2026):** o frontend do painel B prevalece, sem exceção. Cada tela é escrita **depois de ler por inteiro** o módulo correspondente do B (`web/index.html`, `web/css/painel.css`, `web/js/*.js`). Onde o B conflita com a segurança do v2: **o B vale na forma, o v2 vale na regra** (ex.: o selo "A CONFERIR" aparece no mesmo campo onde o B mostra o prazo fatal).

## O que a leitura do B mudou

1. A tela do B **não** é "Hoje + menu superior": é sidebar (logo, 6 abas com contador + botão Calculadora de prazo) + `.topbar` (título, subtítulo, botão primário "Novo…") + `.stats` (pílulas do acordeão) + `#content` + `#syncStatus` + `#idxTicker`.
2. **Toda tarefa nasce vinculada a um atendimento ou a um processo** (campos obrigatórios). Logo, Atendimentos e Processos precisam existir antes de Tarefas.
3. Tarefa no B: 7 estados (A confirmar, Pendente, Em execução, Revisão, Concluída com sucesso, Concluída sem sucesso, Cancelada), prioridade **Urgente/Alta/Normal/Baixa**, subtítulo com sugestões, Partes e Responsáveis como listas dinâmicas, Criado por, Data base, Prazo de entrega, **Prazo fatal**, Observações, Documentos (PDF), "Tarefa vinculada" (filha) e exclusão com senha.
4. Códigos de ficha (`PES.`, `ATE.`, `PRO.`, `TAR.`) aparecem em listas e selects.
5. O acordeão de Tarefas deixa **vários painéis abertos ao mesmo tempo**; o da Agenda abre **um por vez**. Alternância é no navegador (sem recarregar), como no B.

## Ordem (cada plano é executado e verificado antes do seguinte)

| # | Plano | Módulos do B lidos por inteiro |
|---|---|---|
| 1B | Casca idêntica ao B + referência visual do B com dados fictícios | `index.html`, `painel.css`, `02`, `05`, `07`, `13`, `15`, `16`, `99` |
| 2 | Pessoas e Processos idênticos | `14-pessoas`, `09-processos`, `03`, `04`, `06` |
| 3 | Atendimentos e Tarefas idênticos | `10-atendimentos`, `08-tarefas`, `06` |
| 4 | Agenda, Audiências, Diligências idênticas | `11-agenda`, `05`, `07` |
| 5 | Intimações (DJEN) | `11-agenda` (Intimações), servidor `intimacoes` |
| 6 | Monitor DataJud + "Estado das coletas" | `16`, servidor `monitorprocessual`, `sincronizador` |
| 7 | Calculadora de prazo e calendário forense | `12-prazo`, servidor `motorprazo` (só como contraste) |
| 8 | Índices BCB (rodapé), CEP e fechamento | `13`, `14` (CEP), servidor `indices`, `cep` |

Decisão de forma para o que o B não tem: **Minha conta, Usuários, Auditoria, Prazos e Sair** (exigências de segurança do v2) ficam num segundo grupo no pé da sidebar, com o mesmo estilo dos itens do B. "Hoje" some: a **Agenda é a página inicial**, como no B; os prazos a conferir aparecem na faixa `.notice` do B, no topo da Agenda.
