# Fusão — Plano 4C: nada se apaga (cancelar) e processo duplicado com justificativa

**Determinação do advogado (04/11/2026):**
1. **Tarefa é ATA e não se apaga.** O administrador **não** pode excluir tarefa; a opção é o estado **Cancelada** (já existe em "Mover para…").
2. **Audiências, Aulas (compromissos), Intimações e Diligências também não se apagam:** a opção é **Cancelado(a)**, com registro de quem, quando e por quê; o item **continua listado**, com selo.
3. **Processos:** ao salvar, se o número já existir, o sistema **sinaliza o marcador "PROCESSO DUPLICADO" antes de salvar** e **exige justificativa** para a duplicata.

O B vale na forma (mesmos drawers, classes e selos); estas três regras valem sobre o B (regra do v2/determinação do advogado). Método: planos anteriores, `NOTAS-PLANO-*.md`. TDD com vermelho real; mesmas restrições globais (CSP, ≤ 300 linhas, B somente leitura, sem credenciais reais, commits com `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`, sem push).

## Task 1 — Tarefas: remover a exclusão

- Remover rotas `GET/POST /tarefas/{id}/excluir`, o botão "Excluir", o drawer "Excluir tarefa", o serviço de exclusão e o uso de `Reauth` nesse fluxo; remover `PermTarefaDelete` de **todos** os papéis e a permissão do código; ajustar `rbac_test.go`.
- O estado "Cancelada" continua em "Mover para…" (selo `badge-danger`); a tarefa cancelada permanece no acordeão "Cancelada". Mover para Cancelada encerra o prazo ligado (já existe).
- Testes: não existe rota de exclusão (404/405); nenhum papel tem permissão de apagar; cancelar mantém a linha, o código e o histórico; auditoria `tarefa.estado_alterado` com de→para.

## Task 2 — Cancelamento de Aulas/Reuniões/Ligações, Diligências e Intimações

Migração `0012_cancelamentos_e_duplicidade.sql`:
- `agenda_compromissos`: `cancelado boolean not null default false`, `cancelamento_motivo text not null default ''`, `cancelado_em timestamptz`, `cancelado_por uuid references users`; check: cancelado ⇒ motivo não vazio.
- `diligencias`: mesmas colunas; o `status` passa a aceitar também `Cancelada` (manter os valores do B).
- `intimacoes`: trocar `descartada` por **`cancelada`** (migrar valores existentes), com as colunas de cancelamento acima.
- Auditoria: `compromisso.cancelado`, `diligencia.cancelada`, `intimacao.cancelada`.

Comportamento (forma do B, rótulos novos):
- **Compromisso:** o botão "Excluir" (que no B só mostrava "Ainda não implementado…") vira **"Cancelar"** (`btn-danger-text`) e abre o drawer "Cancelar compromisso": subtítulo "O registro é preservado com o motivo do cancelamento." + **Motivo do cancelamento \*** (obrigatório) e botões "Voltar" / "Cancelar compromisso". Cancelado: selo `badge-neutral` "Cancelado" no lugar das ações, linha permanece; **não bloqueia horário** e não entra no contador de "em aberto" (mesma regra da audiência cancelada). Toast "Compromisso cancelado; o registro foi preservado."
- **Diligência:** botão **"Cancelar"** (drawer idêntico, "Cancelar diligência"); status passa a "Cancelada"; selo `badge-neutral`; toast "Diligência cancelada; o registro foi preservado."
- **Intimação:** a ação "Descartar" do B passa a **"Cancelar"** (`btn-danger-text`), com o mesmo drawer de motivo; status `cancelada`, rótulo "Cancelada", classe `badge-neutral`; permanece no histórico; toast "Intimação cancelada e mantida no histórico." Só advogado.
- **Audiência:** já cancela (motivo + senha); manter. Nenhuma rota de exclusão em nenhum desses domínios (testar por enumeração de rotas).
- Permissões novas: `PermCompromissoCancel`, `PermDiligenciaCancel` (advogado e assistente podem cancelar compromisso/diligência? **só advogado**, como a audiência); intimação: `PermIntimacaoDecide` já é do advogado.

## Task 3 — Processo duplicado (marcador + justificativa)

- Migração `0012`: remover o índice único parcial de `processos.numero_cnj`; criar índice comum; acrescentar `duplicado boolean not null default false`, `duplicado_justificativa text not null default ''`, `duplicado_de uuid null` (processo original, o mais antigo com o mesmo número), check: `duplicado` ⇒ justificativa não vazia (≥ 10 caracteres úteis).
- **Detecção:** número CNJ **já existente** em processo não excluído (comparar só dígitos). O servidor **sempre** valida no salvar (criar e editar); o formulário também consulta ao sair do campo (`data-verifica-duplicado` → `GET /processos/duplicado?numero=…&ignorar=ID`, resposta fragmento/JSON simples) para avisar **antes** do clique em salvar.
- **Tela (forma do B):** no drawer "Novo processo"/"Editar processo", quando houver duplicata aparece, logo abaixo do campo "Número do processo (CNJ)", um bloco `.form-block` com o selo `badge badge-danger` **"PROCESSO DUPLICADO"**, o texto "Já existe o processo PRO. 0000012 (NOME DO CLIENTE) com este número." e o campo obrigatório **"Justificativa da duplicidade \*"** (`textarea`). Sem justificativa, o salvar é recusado (servidor) com o toast "Informe a justificativa da duplicidade." Com justificativa, grava `duplicado=true`, `duplicado_de` e a justificativa, e **audita** `processo.duplicado_justificado`.
- **Lista:** o processo marcado exibe o selo **"PROCESSO DUPLICADO"** (`badge-danger`) na coluna "Nº processo" (como o selo de Pendências, abaixo do número) e a justificativa em `title`. Apagar o número duplicado ou corrigir não remove o histórico de auditoria.
- Número vazio ("em branco se ainda não distribuído") **não** conta como duplicata.
- Testes: criar com número novo → sem marcador; criar com número existente **sem** justificativa → recusado; com justificativa → criado com `duplicado=true`; editar um processo para o número de outro → mesma regra; editar o próprio processo sem mudar o número → não marca duplicata de si mesmo; comparação ignora pontuação; número vazio duplicado é permitido; marcador aparece na lista e o `title` traz a justificativa; auditoria registrada.

## Task 4 — Pendências da conferência independente

- Adicionar a `forms.js` as mensagens do B "Ação registrada, mas houve falha ao recarregar a agenda / as audiências / as diligências."; teste que lê a tabela.
- Teste que enumera **todas** as rotas (`rt.Routes()`) e prova: nenhuma POST/DELETE sem permissão; nenhuma rota com "excluir"/"apagar"/"remover" para tarefa, audiência, compromisso, diligência, intimação.
- Corrigir o nome do campo `RepCPFMask` (ou aplicar máscara) e o comentário do `rbac.go` ("assistente escreve só em tarefas e agenda" está desatualizado).
- Atualizar `RELATORIO-*`/`NOTAS-PLANO-4C.md`, a ficha do acervo (estado) e `ACEITE-VISUAL.md`.

## Verificação

`gofmt`, `go vet`, `go test -race -p 1 ./...` (contêiner, rodada final), `scripts/check-lines.sh`, `sha256sum -c HASHES-B.txt`; paridade de estilo do drawer "Cancelar …" e do bloco "PROCESSO DUPLICADO" no Chrome contra os componentes do B (`.form-block`, `.badge-danger`, `.field`, `.drawer-actions`); antes de reconstruir o app dev, confirmar via `psql` na socket local que `processos`/`agenda_compromissos`/`diligencias`/`intimacoes` não têm dado de negócio (se tiverem, a migração deve preservar os dados e não apagar nada).
