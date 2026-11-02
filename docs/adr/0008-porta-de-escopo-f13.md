# ADR 0008 — "Porta" para o controle de acesso por equipe/carteira/cliente (F13)

**Status:** aceita em 02/11/2026 como **decisão de desenho**; a implementação (F13) fica para depois, por decisão do
advogado (o painel precisa estar em uso no escritório sem esperar por isso).

**Contexto:** hoje o acesso é "do escritório inteiro": qualquer advogado vê todas as pessoas, processos, tarefas, agenda
e documentos. Serve a um escritório pequeno. Para grandes bancas (e para a minimização da LGPD) será preciso limitar
por **equipe, carteira ou cliente**. Fazer isso depois é barato **se** hoje houver um ponto único por onde passa toda
leitura e toda escrita — e caro se cada tela tiver que ser reescrita.

**Decisão (o que vale desde já, sem código novo):**

1. **Todo acesso a dado de domínio passa pelo `service` do domínio**, nunca direto pelo handler ou por outro domínio.
   (Já é verdade e já é testado por `internal/app/fronteiras_test.go`.)
2. **Quando a F13 for feita, o gancho entra nos `service`**, em um pacote próprio `internal/acesso`, com duas operações:
   `Pode(ctx, ator, ação, tipo, id) bool` (decisão pontual) e `Filtro(ctx, ator, tipo)` (condição SQL parametrizada
   para listas e buscas). A implementação inicial é "escritório inteiro" (permite tudo); a F13 troca a implementação
   sem mexer nas telas.
3. **O ator já chega aos `service`** (`webx.Actor{UserID, RequestID}`): a F13 acrescenta `Equipes` e papel ali.
4. **Nada de escopo por tela ou por menu.** Esconder botão não é controle de acesso; vale só o que o `service` decide.
5. **Dados que a F13 vai precisar (e que NÃO serão criados agora):** `equipes`, vínculo usuário↔equipe, e uma coluna de
   escopo (`equipe_id` ou `carteira_id`) em `pessoas`, `processos`, `tarefas`, `agenda_eventos` e `documentos`.
   Migração futura: preencher tudo com a equipe "Escritório" e só então ligar o filtro.
6. **Regras que o desenho futuro deverá cobrir (perguntas abertas):** quem vê tudo (sócio/administrador)? o acesso é por
   equipe, por carteira ou por cliente? como se transfere um caso entre advogados? e como se registra o acesso "de
   emergência" (quebra de vidro) na auditoria?

**Consequência:** nenhuma mudança de código nesta etapa. Ao implementar, o primeiro passo é um teste que prove que **toda**
leitura de domínio passa pelo gancho (lista de rotas × chamadas ao `acesso`), antes de ligar qualquer restrição.

**Risco aceito:** até a F13, não usar o sistema com mais de uma equipe que precise de separação entre si.
