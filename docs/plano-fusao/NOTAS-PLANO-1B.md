# Notas de decisão — Plano 1B (03/11/2026)

Decisões conservadoras em pontos ambíguos do plano.

1. **Faixa `.notice` antes da `.topbar`, não dentro de `#content`.** O CSS do B dá `padding:8px 28px` ao `.notice`; dentro de `#content` (que já tem 28px) o texto ficaria recuado em dobro.
2. **A lista de prazos críticos continua no corpo da Agenda inicial.** O plano diz que o widget "vira a faixa". Retirar a lista esconderia prazo vencido na tela de entrada. A faixa resume ("N prazo(s) a conferir · M vencido(s) ou a vencer em até 5 dias — ver Prazos") e a lista fica abaixo. Texto da faixa diz "vencido(s) ou a vencer" porque a contagem inclui atrasados (mais honesto que "vencem").
3. **Home = Agenda**: `GET /` renderiza o shell da Agenda (cabeçalho, chip, faixa) com os widgets de antes (prazos, minhas tarefas, agenda de 7 dias); os testes de regra de negócio da antiga tela "Hoje" continuam valendo. `/agenda` segue sendo a lista de eventos.
4. **Subtítulos**: os do B quando verdadeiros no v2 (Pessoas). Agenda, Processos e Tarefas têm texto próprio porque o do B descreve campos/categorias que o v2 ainda não tem (classe, valor da causa, vínculo obrigatório com atendimento/processo).
5. **Botão do topo** é link (`<a class="btn btn-primary">`) até as telas virarem drawer (planos 2 a 4); `ActionDrawer` já existe.
6. **Logo sem link** (como no B) e alt "Teodoro Advocacia Estratégica". O link para a página inicial saiu; o item Agenda do menu cumpre o papel.
7. **`Sair`** continua um formulário POST (precisa de CSRF) logo abaixo do grupo "sistema", não um item `<a>` do menu.
8. **`#syncStatus`**: o JS `16-sync-status.js` do B não foi portado (o v2 não tem `/api/sincronizacao/status`); o servidor devolve o texto honesto fixo.
9. **Contadores** do menu contam linhas não excluídas (Agenda: eventos; o B soma audiências e diligências, que o v2 ainda não separa).
10. **`painel.css` (316 linhas) é cópia literal do B**: `scripts/check-lines.sh` passou a ignorá-lo.
11. **Teste de migração 0007** deixava linhas que impediam o `Reset` do teste seguinte; ganhou limpeza (`t.Cleanup`). Pré-existente, sem ele `go test ./...` não fechava em banco limpo.
12. **`-race` precisa de gcc**, ausente no WSL: `go test -race -p 1 ./...` rodou em contêiner `golang:1.26.10-bookworm` (mesma imagem do Dockerfile) com Postgres descartável. `-p 1` porque os pacotes compartilham o banco de teste.
13. **TDD**: Tarefas 2 e 4 tiveram testes escritos antes do código mas não executados em vermelho (o código foi escrito na sequência, em lote); Tarefa 3 teve os testes e a implementação no mesmo passo. Os testes passaram e foram conferidos em Chrome.
