# PLANO DE AÇÃO E EXECUÇÃO — Painel Teodoro

Fonte: `RELATORIO-AUDITORIA-2026-11-08.md` (raiz do projeto). Nesta fase, somente planejamento. Nenhum código foi alterado.

---

## SEÇÃO A — RESUMO EXECUTIVO DO PLANO

**Trava de início.** Relatório localizado: `RELATORIO-AUDITORIA-2026-11-08.md`, na raiz do projeto. A estrutura universo → problema → detalhes está presente: seções 1 a 10, com gravidade, estado, localização, evidência e correção em cada achado. Lacuna: o campo causa-raiz não aparece em 24 dos 30 achados. Tratei como não bloqueante, porque a evidência de cada item permite identificar a causa. Essa inferência está marcada na Seção F. Se a trava for lida como estrita, este é o ponto de parada.

**Ambiente.** Repositório git, branch `main`, HEAD `1635de0`. Há 3 alterações rastreadas (`cmd/painel/main.go`, `go.mod`, `internal/auth/commonpw.go`) e 12 entradas não rastreadas, entre elas `painel.exe` (29 MB) e pacotes inteiros de `internal/`. Ferramentas usadas no relatório: Go 1.27.0 local (o `go.mod` declara 1.26.10), staticcheck v0.8.2, gosec v2.30.0, govulncheck v1.9.0 e PostgreSQL 17 em contêiner. Não executadas ou sem acesso: golangci-lint (roda só na CI), gitleaks, trivy, check-links (python3 ausente), navegador, CI remota, VM e produção. Tudo isso é A CONFIRMAR no ambiente.

**O desconfortável primeiro.** O achado de maior efeito prático (1.1) está ativo na árvore de trabalho agora: as coletas DJEN, DataJud e BCB ligam sem a flag. O relatório o classifica como P1, e não há nenhum P0. O plano o corrige na Onda 2, depois da linha de base e do teste que o prova. Nada é corrigido antes da autorização da Onda 0.

**Totais por gravidade.**

| Gravidade | Achados | Observação |
|---|---|---|
| P0 | 0 | — |
| P1 | 3 | 1.1; 4.1 (a validar pelo advogado); 5.1 |
| P2 | 9 | — |
| P3 | 12 | inclui 1.2 (ambiente) e 5.2 (hipótese) |
| P4 | 6 | — |
| **Total** | **30** | mais a lacuna 7.2, que não é achado |

**Tarefas por onda.** Onda 0: 5 · Onda 1: 0 · Onda 2: 3 · Onda 3: 8 · Onda 4: 12. Total: 28. Achados com tarefa: 28. Não tratados, com justificativa: 2 (3.2 e 9.4).

**Caminho crítico.** T-0.01 → T-0.02 → T-0.03 → T-3.01 → T-3.02 → T-3.03 → T-3.07 → T-4.08. Ramo jurídico: D-8 → T-2.02 → T-3.05 e T-3.06 → T-4.04. O ramo jurídico depende da decisão do advogado, não da execução.

**Principais riscos do plano.**
1. T-3.02 remove código não versionado. Sem histórico no git, só a cópia de T-0.01 permite reverter.
2. T-2.03 pode causar perda de dados se existir banco com registros. D-11 vem antes.
3. O plano não afirma o conteúdo do CPC art. 229 nem da Lei 14.365/2022. Essas decisões são do advogado.
4. T-3.04 altera controle de acesso. Os serviços de lista, agenda e tarefas foram lidos só por amostra no relatório.
5. A linha de base depende de PostgreSQL 17 e de um ambiente Windows com `core.autocrlf` (achado 1.2). Não há CI verificada.

**Decisões que bloqueiam a execução.** D-1 e D-2 (antes da Onda 0); D-8 (T-2.02); D-11 (T-2.03); D-3 (T-3.01); D-4 (T-3.03); D-5 (T-3.04); D-12 (T-3.07). Lista completa na Seção D.

**Execução.** Segue o protocolo de execução da seção 9 do comando, somente após autorização expressa de cada onda.

---

## SEÇÃO B — ONDAS E TAREFAS

**Critério de parada (vale para todas as ondas).** A execução para e volta ao usuário se: (a) um teste da linha de base que passava passa a falhar; (b) uma migração pode perder dados e não há backup verificado; (c) o código real diverge do relatório (localização inexistente ou comportamento diferente do descrito); (d) a prova real não passa e a correção dentro da própria tarefa não resolve; (e) a tarefa exige ampliar escopo ou permissões.

### ONDA 0 — PREPARAÇÃO E LINHA DE BASE

**0.1. T-0.01 — Cópia de segurança da árvore atual**
- Achado(s) de origem: 1.3 (arquivos não versionados, sem histórico) e 1.1 (alteração não commitada).
- Objetivo: preservar a árvore de trabalho inteira antes de qualquer alteração.
- Tipo: infraestrutura.
- Localização: raiz do projeto, inclusive `.git`, arquivos não rastreados e `painel.exe`. Destino fora do repositório (D-2).
- Pré-condições e dependências: D-2.
- Passos de execução: (1) gravar `git status`, `git log -1` e `git diff` em arquivos; (2) copiar a árvore inteira, incluindo `.git`; (3) listar os não rastreados com SHA-256; (4) repetir a comparação na cópia.
- Risco de regressão: cópia incompleta; cópia em pasta sincronizada pelo OneDrive (ver Seção E).
- Prova real: SHA-256 de `cmd/painel/main.go` e de `painel.exe` iguais na origem e na cópia; mesma contagem de arquivos; `git status` da cópia igual ao da origem.
- Critério de conclusão: as três comparações iguais, registradas em PROGRESSO_EXECUCAO.md.
- Reversão: não se aplica. Esta cópia é a reversão das demais tarefas.
- Esforço relativo: P (estimativa, sem medição).
- Papel responsável: SRE/DevOps.
- Autorização necessária do usuário: não, além da autorização da Onda 0.
- Explicação técnica: a cópia com `.git`, mais o patch das alterações rastreadas e o hash dos não rastreados, é a única forma de voltar atrás em arquivos sem histórico.
- Analogia: é como tirar cópia integral dos autos antes de mexer na juntada: as folhas soltas, sem protocolo no sistema, só existem na cópia.

**0.2. T-0.02 — Branch dedicada a partir do HEAD**
- Achado(s) de origem: 1.1 (estado de partida).
- Objetivo: isolar as correções em branch própria, sem tocar nos arquivos.
- Tipo: infraestrutura.
- Localização: repositório git, branch `main`.
- Pré-condições e dependências: T-0.01; D-1.
- Passos de execução: (1) criar a branch `remediacao/auditoria-2026-11-08` a partir do HEAD, sem commit e sem alterar arquivos; (2) confirmar que as alterações não commitadas continuam como estavam. Elas só serão tratadas na tarefa própria (T-2.01).
- Risco de regressão: commit acidental de não rastreados por `git add .`. Mitigado por T-3.01 e pela regra de commit explícito.
- Prova real: `git rev-parse --abbrev-ref HEAD` retorna a branch nova; `git rev-parse HEAD` igual ao da `main` antes da troca; `git status --porcelain` igual ao registrado em T-0.01.
- Critério de conclusão: as três verificações passam.
- Reversão: `git switch main`. Sem perda, pois as alterações acompanham a troca de branch.
- Esforço relativo: P.
- Papel responsável: SRE/DevOps.
- Autorização necessária do usuário: não, além da autorização da Onda 0.
- Explicação técnica: uma branch nova a partir do HEAD isola as correções e preserva o `main` como referência.
- Analogia: é como abrir um volume novo dos autos para as correções, sem misturar as peças já juntadas com as que ainda estão em análise.

**0.3. T-0.03 — Linha de base de build, testes e análise**
- Achado(s) de origem: 9.1 (suíte "ok" sem banco), 1.2 (falha conhecida no Windows), 9.3 (gate de análise) e 2.5 (versões).
- Objetivo: registrar, com saídas e códigos de retorno, o estado do projeto antes da primeira correção.
- Tipo: teste.
- Localização: repositório; PostgreSQL 17 descartável; `go`, `staticcheck`, `gosec`, `govulncheck`, `scripts/check-lines.sh`. golangci-lint roda na CI (A CONFIRMAR se existe localmente).
- Pré-condições e dependências: T-0.02; PostgreSQL 17 disponível (A CONFIRMAR no ambiente).
- Passos de execução: (1) rodar tudo em duas árvores: a atual e o HEAD, exportado para pasta fora do projeto; (2) `go build ./...` e `go vet ./...`; (3) `go test ./...` sem `PAINEL_TEST_DB`, registrando os SKIP; (4) `go test -race -count=1 -p 1 ./...` com `PAINEL_TEST_DB` apontando para o PostgreSQL 17; (5) `staticcheck ./...`, `gosec` e `govulncheck ./...`; (6) `scripts/check-lines.sh`; (7) gravar tudo em PROGRESSO_EXECUCAO.md.
- Risco de regressão: linha de base sem banco reproduz o falso "ok" (9.1). Por isso o passo 4 é obrigatório.
- Prova real: relatório da linha de base com contagem de PASS, FAIL e SKIP por pacote, códigos de retorno e contagem de achados das ferramentas. Uma segunda execução deve dar as mesmas contagens. Valores do relatório a reconfirmar: 323 execuções, 179 SKIP sem banco; com banco, 1 falha (CRLF, 1.2) e nenhuma data race; staticcheck com 0; gosec com 18 (8 HIGH, 10 MEDIUM); govulncheck com 13 alcançáveis.
- Critério de conclusão: registro completo das duas árvores; segunda execução com as mesmas contagens.
- Reversão: não se aplica (somente leitura; o banco descartável é destruído ao fim).
- Esforço relativo: M (justificativa: várias ferramentas, banco de dados e duas árvores).
- Papel responsável: QA + SRE/DevOps.
- Autorização necessária do usuário: não, além da autorização da Onda 0.
- Explicação técnica: foto do estado antes de qualquer mudança, com banco obrigatório. Sem ela, nenhuma correção tem comparação objetiva.
- Analogia: é como registrar o estado do processo antes da primeira petição: se alguém perguntar depois o que mudou, há termo de referência para comparar.

**0.4. T-0.04 — Teste de wiring das coletas externas (prova do defeito 1.1)**
- Achado(s) de origem: 9.2 (wiring de produção sem teste) e 1.1 (defeito a provar).
- Objetivo: criar teste automatizado que monta a configuração de produção com as coletas desligadas e falha se houver qualquer chamada externa.
- Tipo: teste.
- Localização: `cmd/painel/main.go:88–117` (relatório); arquivo de teste novo em `cmd/painel/` (nome A CONFIRMAR).
- Pré-condições e dependências: T-0.03. Se a montagem não for testável sem extração, vale D-16.
- Passos de execução: (1) verificar se a montagem de `serve()` é testável sem alterar comportamento; (2) escrever o teste: configuração sem `PAINEL_*_ATIVO`, transporte HTTP que conta chamadas, asserção de zero chamadas e de que o log não diz "LIGADO"; (3) rodar no HEAD: deve passar; (4) rodar na árvore atual: deve falhar pelo motivo esperado (chamada externa), e não por erro de compilação.
- Risco de regressão: extração para testar `main` pode conflitar com a alteração de 1.1. Mitigação: o teste precisa passar no HEAD antes de qualquer mudança.
- Prova real: passa no HEAD; falha na árvore atual com a asserção de chamada externa. As duas saídas ficam registradas.
- Critério de conclusão: teste existe; passa no HEAD; falha na árvore atual pelo motivo esperado.
- Reversão: remover o arquivo de teste criado.
- Esforço relativo: M (justificativa: a testabilidade de `main` pode exigir extração).
- Papel responsável: QA + AppSec.
- Autorização necessária do usuário: não. Se a extração em `main.go` for necessária, sim (D-16).
- Explicação técnica: teste de caracterização do wiring. Prova que, na configuração padrão, nenhuma das três coletas faz chamada externa. Falha na árvore atual e passa no HEAD.
- Analogia: é como um laudo que reproduz o defeito antes de alegá-lo: o mesmo laudo prova o defeito antes da correção e a sua ausência depois.

**0.5. T-0.05 — Caracterização do cálculo de prazos**
- Achado(s) de origem: 4.1, 4.2, 4.3 e 4.4 (áreas que serão alteradas).
- Objetivo: fixar em testes o resultado atual de casos de referência, antes de qualquer mudança de regra.
- Tipo: teste.
- Localização: `internal/prazos/calculo` (`regras.go`, `contagem.go`, `corridos`, calendário); `internal/prazos/handler.go:45`; `internal/app/app.go:101` (relatório). Cobertura atual A CONFIRMAR.
- Pré-condições e dependências: T-0.03.
- Passos de execução: (1) listar casos de referência: prazo simples, suspensão de fim de ano, feriado, litisconsortes; (2) registrar o resultado atual de cada caso como esperado; (3) marcar como "caracteriza o atual, não a norma" os casos cujo resultado o relatório questiona; (4) rodar.
- Risco de regressão: fixar comportamento errado como padrão. Mitigação: casos marcados e validados pelo advogado (D-8).
- Prova real: a suíte de caracterização passa na linha de base. Depois das Ondas 2 e 3, só falham os casos cuja mudança estiver decidida e documentada.
- Critério de conclusão: casos registrados com resultado esperado; zero falhas na linha de base.
- Reversão: remover o arquivo de teste.
- Esforço relativo: M (justificativa: volume de regras e casos a validar com o advogado).
- Papel responsável: QA + Analista de requisitos.
- Autorização necessária do usuário: não para criar; sim para validar os casos com o advogado (D-8).
- Explicação técnica: testes que fixam o resultado atual de casos de referência. Uma mudança futura de regra aparece como falha, e não como surpresa.
- Analogia: é como registrar dez casos já decididos antes de mudar a tabela de prazos: qualquer mudança que altere um desses resultados aparece no confronto.

**Portão de saída da Onda 0:** T-0.01 verificada por hash; T-0.02 concluída com a árvore inalterada; T-0.03 registrada nas duas árvores e repetível; T-0.04 falha na árvore atual e passa no HEAD; T-0.05 passa na linha de base; PROGRESSO_EXECUCAO.md atualizado; autorização para a Onda 2.

### ONDA 1 — P0: BLOQUEANTES E RISCOS CRÍTICOS

Sem tarefas. O relatório não tem achado P0. O portão passa quando a Onda 0 passa.

### ONDA 2 — P1: ALTO

**2.1. T-2.01 — Restaurar o controle da coleta externa**
- Achado(s) de origem: 1.1 (P1). Fecha 9.2 junto com T-0.04.
- Objetivo: DJEN, DataJud e BCB só rodam com a flag explícita da configuração.
- Tipo: correção.
- Localização: `cmd/painel/main.go:88–117`; `internal/monitor/agendador.go:15` (`Iniciar`) (relatório).
- Pré-condições e dependências: T-0.01, T-0.04, D-1.
- Passos de execução: (1) rodar T-0.04 na árvore atual e registrar a falha; (2) comparar o trecho atual com o HEAD (`git diff HEAD -- cmd/painel/main.go`); (3) restaurar a leitura de `Ativo` e dos intervalos a partir de `cfg`, como no HEAD; (4) corrigir a espera passada ao `Iniciar`, que hoje recebe 5 s no lugar do intervalo, e a mensagem de log; (5) se o teste precisar de intervalo curto, usar variável de ambiente explícita, nunca aplicada em produção; (6) rodar T-0.04 e a suíte com banco.
- Risco de regressão: quem dependia da coleta automática no ambiente local vai notar que ela parou. Isso é o pretendido. `iniciar.ps1` (2.4) pode ligar a coleta (ver T-3.01). Valores padrão das flags no HEAD: A CONFIRMAR.
- Prova real: ANTES: T-0.04 falha na árvore atual. DEPOIS: T-0.04 passa; um teste positivo (flag `true`, transporte falso) confirma que a coleta liga quando pedida; a suíte fica igual à linha de base.
- Critério de conclusão: as verificações acima passam; `git diff HEAD -- cmd/painel/main.go` vazio ou justificado linha a linha.
- Reversão: restaurar `main.go` da cópia de T-0.01. Sem efeito em dados.
- Esforço relativo: P (estimativa: trecho pequeno; o custo está no teste).
- Papel responsável: AppSec + SRE/DevOps.
- Autorização necessária do usuário: não, além da autorização da Onda 2.
- Explicação técnica: a configuração de produção liga as três coletas sem ler as flags, e o agendador recebe a espera inicial em vez do intervalo. A correção restaura a leitura de `cfg` do HEAD.
- Analogia: é como um escritório que liga o protocolo eletrônico automaticamente no primeiro minuto do expediente, sem validação do acesso: o painel diz "ativo", mas o procedimento de validação nunca aconteceu.

**2.2. T-2.02 — Prazo em dobro (CPC art. 229): reprodução e decisão do advogado**
- Achado(s) de origem: 4.1 (P1, fortemente indicado: passo de reprodução antes da correção).
- Objetivo: a regra de dobro fica decidida pelo advogado e aparece no cálculo como alternativa visível, se ele assim decidir.
- Tipo: decisão do usuário + correção.
- Localização: `docs/juridico/REGRAS-DE-PRAZO.md` (emenda de 04/11); `internal/prazos/calculo/regras.go` (relatório).
- Pré-condições e dependências: T-0.05; D-8.
- Passos de execução: (1) o advogado consulta o texto integral do CPC art. 229; o relatório não o consultou, e este plano não afirma seu conteúdo; (2) reproduzir o caso com litisconsortes de escritórios distintos e registrar a saída atual ("prazo em dobro não considerado", sem alternativa); (3) o advogado decide por escrito se o dobro entra como alternativa visível ou se a exclusão se mantém, com fundamento; (4) se entrar, implementar a alternativa no padrão das outras leituras já existentes e criar o caso de teste; (5) rodar T-0.05 e confirmar que só mudaram os casos previstos na decisão.
- Risco de regressão: a alteração atinge todas as telas de prazo. A alternativa não pode substituir o vencimento principal sem decisão. Prazos já criados não são recalculados (A CONFIRMAR).
- Prova real: ANTES: o caso de litisconsórcio sai sem alternativa com dobro (saída registrada). DEPOIS, se a decisão for incluir: o caso sai com a alternativa, e o vencimento principal dos demais casos de T-0.05 não muda. Registro escrito da decisão, assinado pelo advogado.
- Critério de conclusão: decisão registrada; caso passa; mudanças em T-0.05 limitadas aos casos previstos.
- Reversão: reverter o commit. Se o cálculo não gravar dados (A CONFIRMAR), isso basta.
- Esforço relativo: M (justificativa: a parte técnica é pequena; a decisão jurídica é o trabalho).
- Papel responsável: Advogado (decisão) + Analista de requisitos + QA.
- Autorização necessária do usuário: sim, decisão jurídica do advogado.
- Explicação técnica: o relatório aponta, sem consultar a norma, que a exclusão do dobro contraria o art. 229, e o aviso aparece só no resultado. A verificação cabe ao advogado. O cálculo deve mostrar a alternativa, e não apenas uma nota.
- Analogia: é como uma tabela de prazos que ignora o dobro para litisconsortes com procuradores distintos: o vencimento sai mais cedo que o real. O erro aponta para o lado seguro, mas a premissa precisa ser checada pelo advogado.

**2.3. T-2.03 — Migrações 0008 a 0014 aplicáveis sobre dados existentes**
- Achado(s) de origem: 5.1 (P1, operacional).
- Objetivo: um banco com registros sobe para esta versão por procedimento documentado e verificado.
- Tipo: correção + decisão do usuário.
- Localização: `migrations/0008_pessoas_processos_do_b.sql:16`, `0009:14`, `0010:15`, `0011:76`, `0012:53`, `0013:76`, `0014:43/61/64` (relatório); `PENDENCIAS.md`.
- Pré-condições e dependências: T-0.01, T-0.03 (PostgreSQL), D-11.
- Passos de execução: (1) ANTES: num banco descartável com PostgreSQL 17, aplicar 0001 a 0007, inserir uma linha em cada tabela afetada e aplicar 0008; registrar o `RAISE EXCEPTION`; (2) D-11 define o método: (a) roteiro de conversão com migração nova de conversão, sem editar migração já aplicada (recomendado), ou (b) alterar as guardas, somente se nenhum banco aplicou as migrações; (3) escrever o roteiro: ordem, verificação e reversão; (4) implementar o método escolhido; (5) DEPOIS: aplicar 0001 a 0014 com dados de teste e contar linhas por tabela antes e depois; (6) executar o roteiro numa cópia do banco.
- Risco de regressão: perda de dados. Editar migração já aplicada quebra o histórico de migrações. O Down, protegido por dados, continua exigindo roteiro.
- Prova real: ANTES: 0008 falha com `RAISE EXCEPTION` sobre tabela com linha. DEPOIS: 0001 a 0014 aplicam com dados de teste; contagem de linhas por tabela igual antes e depois; roteiro executado do zero numa cópia com resultado idêntico.
- Critério de conclusão: provas antes e depois registradas; contagens iguais; roteiro revisado pelo usuário.
- Reversão: backup do banco de teste e de qualquer banco real antes da tarefa; restaurar o backup. Migração de conversão com Down ou roteiro de reversão escrito.
- Esforço relativo: G (justificativa: sete migrações, dados e risco de perda; o método depende de decisão).
- Papel responsável: Banco de dados e migrações + SRE/DevOps.
- Autorização necessária do usuário: sim, por tratar migração com potencial perda de dados.
- Explicação técnica: as migrações 0008 a 0014 recusam aplicar se a tabela já tiver linhas, e o Down recusa se houver dados. Sem roteiro de conversão, nenhum banco com registros sobe para esta versão.
- Analogia: é como um sistema de processo que só aceita migrar se o acervo de destino estiver vazio: sem roteiro de conversão, o acervo real não sobe.

**Portão de saída da Onda 2:** as três tarefas com prova aprovada; suíte sem regressão em relação à linha de base da Onda 0; T-0.04 passando; PROGRESSO_EXECUCAO.md atualizado; autorização para a Onda 3.

### ONDA 3 — P2: MÉDIO

**3.1. T-3.01 — Versionamento e regras de ignore**
- Achado(s) de origem: 1.3 (P2) e 2.4 (P3). Mesma causa-raiz: arquivos não ignorados e fora do versionamento. Tarefa única.
- Objetivo: decidir o que entra no repositório e impedir a entrada acidental de binário, segredo e arquivo de desenvolvimento.
- Tipo: decisão do usuário + infraestrutura.
- Localização: raiz (`Dockerfile`, `docker-compose.yml`, `iniciar.ps1`, `painel.exe`, `RELATORIO-AUDITORIA-2026-11-08.md`); `.gitignore` (A CONFIRMAR se existe e o que contém).
- Pré-condições e dependências: T-0.02; D-3.
- Passos de execução: (1) D-3 define a lista do que é versionado; (2) acrescentar ao `.gitignore` o que D-3 mandar ignorar, incluindo `*.exe`; (3) se D-3 aprovar, gerar os segredos de desenvolvimento por script e não por arquivo versionado (2.4); (4) rodar `git add -n .` e revisar a lista; (5) não commitar nesta tarefa.
- Risco de regressão: regra de ignore que esconde arquivo que deveria ser versionado; `iniciar.ps1`, usado no dia a dia, sai do repositório.
- Prova real: `git check-ignore -v painel.exe` retorna a regra; `git add -n .` não lista `painel.exe` nem os arquivos de segredo; `git ls-files` não contém binário.
- Critério de conclusão: as três verificações passam; D-3 registrada.
- Reversão: remover as linhas acrescentadas ao `.gitignore`. Nenhum arquivo é apagado.
- Esforço relativo: P.
- Papel responsável: SRE/DevOps + Full-stack.
- Autorização necessária do usuário: sim, D-3.
- Explicação técnica: `*.exe` e os arquivos da raiz com segredos de desenvolvimento não estão ignorados. Um `git add .` levaria 29 MB e chaves previsíveis para o histórico.
- Analogia: é como juntar ao processo eletrônico tudo o que está na mesa, inclusive minutas e um pen drive com senhas: depois de juntado, o histórico guarda o que entrou, e retirar exige rastro.

**3.2. T-3.02 — Remover pacotes sem importador e o subsistema de busca sem autenticação**
- Achado(s) de origem: 2.1 (P2, latente), 3.1 (P3), 2.3 (P3) e 9.3 (P3). Mesma causa-raiz: código gerado fora da arquitetura, sem remoção do que foi substituído. Tarefa única, na onda do achado de maior gravidade (2.1).
- Objetivo: cada função tem uma implementação ativa, e sai o subsistema de busca sem autenticação.
- Tipo: correção (remoção).
- Localização (relatório): `internal/worker/{bcb,datajud,djen}` (incluindo `client.go:15`); `internal/calculator/prazos`; `internal/crypto/vault`; `internal/middleware`; `internal/router`; `internal/sse`; `internal/handler/live`. Somente os subpacotes listados. O restante dos diretórios é A CONFIRMAR.
- Pré-condições e dependências: T-0.01 (sem histórico git; a cópia é a reversão), T-0.03, T-3.01, D-3, D-19.
- Passos de execução: (1) provar que nada importa os pacotes: busca pelo caminho de import em todos os `*.go`, scripts e Dockerfiles; `go list -deps ./cmd/painel` não pode conter os pacotes; (2) confirmar a autorização expressa de exclusão (D-19); (3) remover os subpacotes listados; (4) `go build ./...` e `go vet ./...`; (5) `gosec` sobre o código restante; (6) confirmar que `/api/processos/busca` e `/events` não existem mais no código.
- Risco de regressão: pacote usado por teste, script ou Dockerfile que o grafo de importação não mostra. `internal/middleware` não é cópia exata de `platform/httpx` (A CONFIRMAR).
- Prova real: ANTES: `gosec` lista 3 HIGH e 2 MEDIUM nesses pacotes (9.3); `grep` encontra `/api/processos/busca` em `router.go` e em `live/search.go`. DEPOIS: `grep` dos caminhos de import removidos retorna zero; `go build ./...` e `go vet ./...` passam; testes com banco iguais à linha de base, exceto os testes dos pacotes removidos, listados; `gosec` sem HIGH e MEDIUM no código restante; `grep` de `/api/processos/busca` retorna zero.
- Critério de conclusão: provas acima; lista de pacotes removidos registrada em PROGRESSO_EXECUCAO.md.
- Reversão: copiar os diretórios de volta da cópia de T-0.01. Não há histórico git para esses arquivos.
- Esforço relativo: M (justificativa: sete grupos de pacotes, 17 arquivos e verificação de não uso).
- Papel responsável: Arquiteto + AppSec + Full-stack.
- Autorização necessária do usuário: sim. Exclusão de código sem histórico git (D-19).
- Explicação técnica: os pacotes não são importados e duplicam implementações ativas. Um deles é um canal de busca e eventos sem o `auth.Router`, com HTML montado sem escape. Outro tem a constante que o `gosec` marca como credencial.
- Analogia: é como manter três cópias da mesma procuração em gavetas diferentes: não se sabe qual é a válida, e uma delas pode ser a que o cartório usa.

**3.3. T-3.03 — Alinhar a versão do Go e reexecutar o govulncheck**
- Achado(s) de origem: 2.5 (P2). Inclui o subitem de `golang.org/x/crypto` v0.58.0 (pacote `openpgp`, não chamado), que é P4 dentro do próprio 2.5 e não eleva a gravidade.
- Objetivo: `go.mod`, CI, imagem de produção e máquina de desenvolvimento usam a mesma versão corrigida, e o `govulncheck` não encontra vulnerabilidade alcançável.
- Tipo: correção (configuração) + decisão do usuário.
- Localização: `go.mod` (diretiva `go`, declarada 1.26.10); arquivo de CI (A CONFIRMAR nome e forma de leitura da versão); `deploy/Dockerfile` (`golang:1.27.3`); toolchain local (fora do repositório).
- Pré-condições e dependências: T-3.02 (o `govulncheck` deve refletir o código final); D-4; D-19 para o push.
- Passos de execução: (1) D-4 define a versão; (2) alterar a diretiva `go` em `go.mod` e a fonte de versão da CI; (3) atualizar a toolchain local; (4) `go mod tidy` e revisão de `go.mod` e `go.sum`, incluindo `golang.org/x/crypto` se a versão alvo exigir; (5) `govulncheck ./...`; (6) validar a CI numa branch remota (ação externa; exige confirmação).
- Risco de regressão: alterar a diretiva pode quebrar builds em máquinas que não atualizam a toolchain. `go.mod` aparece modificado por causa do CRLF (1.2), o que suja o diff. A CI não foi verificada.
- Prova real: ANTES: `govulncheck ./...` reporta 13 alcançáveis (relatório; reconfirmar em T-0.03). DEPOIS: `govulncheck ./...` sem alcançáveis; `go version` local igual à versão de D-4; a mesma versão aparece em `go.mod`, na CI e em `deploy/Dockerfile` (`grep`); CI verde na branch remota (A CONFIRMAR: sem acesso à CI nesta auditoria).
- Critério de conclusão: as três verificações; CI verde.
- Reversão: reverter o commit de `go.mod` e da CI. A toolchain local volta à versão anterior.
- Esforço relativo: P (estimativa: a mudança é pequena; o custo está na validação da CI).
- Papel responsável: SRE/DevOps + AppSec.
- Autorização necessária do usuário: sim. Escolha de versão (D-4) e push (D-19).
- Explicação técnica: `go.mod` declara 1.26.10, a CI lê esse arquivo e a imagem usa 1.27.3. As vulnerabilidades alcançáveis aparecem corrigidas em 1.27.2 ou 1.27.3. A CI testa com 1.26.10, que não foi verificado nesta auditoria.
- Analogia: é como testar a peça com a versão antiga do CPC: o resultado pode estar certo, mas não prova nada sobre a versão que será protocolada.

**3.4. T-3.04 — Isolamento de acesso por responsável ou equipe**
- Achado(s) de origem: 2.2 (P2). Decisão de produto pendente (auditoria de 05/11, item 2.4).
- Objetivo: advogado e assistente veem processo, pessoa e documento apenas dentro da regra definida pelo escritório.
- Tipo: decisão do usuário + correção.
- Localização: repositórios de processos, pessoas e documentos (ex.: `documentos/handler.go:158`, `Baixar`) (relatório). Caminhos exatos A CONFIRMAR.
- Pré-condições e dependências: D-5; T-0.03. Migração só se a regra exigir campo novo (A CONFIRMAR se já existe campo de responsável).
- Passos de execução: (1) D-5 define a regra; (2) mapear as consultas de leitura nos três repositórios e no download de documento; (3) escrever teste com duas contas de papel `advogado` em equipes distintas e um processo de cada, e rodá-lo antes da correção; (4) aplicar o filtro na camada de consulta, não só no handler; (5) repetir o teste e rodar as telas de lista, agenda e tarefas.
- Risco de regressão: lista, agenda, tarefas, relatórios e busca podem perder dado visível legitimamente. Esses serviços foram lidos só por amostra (Seção F). Consulta sem cobertura pode escapar do filtro.
- Prova real: ANTES: a conta da equipe B lê processo, pessoa e documento da equipe A (defeito). DEPOIS: negativa (código de erro conforme D-5) para os três recursos; a conta responsável continua lendo; suíte com banco igual à linha de base.
- Critério de conclusão: provas antes e depois nos três recursos; regra registrada.
- Reversão: reverter o commit. Se houver migração, Down com plano de dados.
- Esforço relativo: G (justificativa: três domínios, consultas espalhadas e decisão de negócio pendente).
- Papel responsável: AppSec + Analista de requisitos + Full-stack + QA.
- Autorização necessária do usuário: sim. A regra de visibilidade é decisão do escritório.
- Explicação técnica: a permissão é por papel. Qualquer `advogado` ou `assistente` lê todo processo, pessoa e documento. A correção filtra na consulta, por responsável ou equipe.
- Analogia: hoje todo advogado tem a chave do arquivo geral; a correção entrega a chave só dos processos em que ele atua ou da equipe, como faz a distribuição por responsável.

**3.5. T-3.05 — Calculadora liberada sem validação das regras**
- Achado(s) de origem: 4.2 (P2). Relações: 4.1 e 4.3.
- Objetivo: o cálculo não vira prazo no sistema enquanto as regras não estiverem validadas, ou o risco fica aceito por escrito.
- Tipo: decisão do usuário + correção.
- Localização: `internal/app/app.go:101`; `internal/prazos/handler.go:45` (rota `POST /prazos/calcular`); `calculadora.go:119` (banner) (relatório).
- Pré-condições e dependências: D-6; T-0.05; T-2.02 (o texto da saída depende da decisão do dobro).
- Passos de execução: (1) D-6 decide; (2) se bloquear: escrever teste que cria prazo a partir do cálculo com `PAINEL_CALCULADORA_REGRAS_VALIDADAS` desligada, esperando recusa; (3) implementar o bloqueio na criação, e não só no banner; (4) se aceitar o risco: registrar em `docs/` com data e responsável; (5) repetir o teste.
- Risco de regressão: bloqueio impede o uso legítimo enquanto a flag não é ligada; mudança no fluxo de criação.
- Prova real: ANTES: com a flag desligada, o teste consegue criar prazo a partir do cálculo (defeito). DEPOIS, se bloquear: recusa com mensagem, e com a flag ligada o fluxo volta ao normal. DEPOIS, se aceitar: o registro escrito existe e o banner aparece no resultado.
- Critério de conclusão: prova conforme a escolha; decisão registrada.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: Full-stack + AppSec; advogado para a decisão.
- Autorização necessária do usuário: sim (D-6).
- Explicação técnica: a rota de cálculo está registrada sempre que o serviço existe. A flag só troca o aviso. A validação jurídica é declarada como pré-condição, mas o controle técnico é cosmético.
- Analogia: é como liberar a minuta de petição ao cliente antes da revisão do sócio, com uma faixa de "não use" no topo: o aviso existe, mas o arquivo já saiu da mesa.

**3.6. T-3.06 — Suspensão do processo penal (20/12 a 20/01) não aplicada**
- Achado(s) de origem: 4.3 (P2).
- Objetivo: a suspensão fica implementada ou mantida como aviso, por decisão do advogado.
- Tipo: decisão do usuário + correção.
- Localização: `docs/juridico/REGRAS-DE-PRAZO.md` (linha 9 e divergência 4); `internal/prazos/calculo/contagem.go` (`duvidasCorridos`) (relatório).
- Pré-condições e dependências: D-9; T-0.05; T-2.02.
- Passos de execução: (1) o advogado confirma a Lei 14.365/2022 e as exceções dos incisos I a III, não verificadas nesta auditoria; (2) D-9 decide; (3) se implementar: casos dentro e fora da janela, e casos de exceção; (4) se manter: teste que confirma o aviso no resultado; (5) atualizar `REGRAS-DE-PRAZO.md`.
- Risco de regressão: mudança de vencimento em processo penal; prazos já exibidos podem mudar de valor.
- Prova real: ANTES: caso penal com prazo dentro de 20/12 a 20/01 retorna vencimento sem suspensão (registrado em T-0.05). DEPOIS, se implementar: casos de T-0.05 e os novos passam conforme a regra decidida. Se manter: o aviso aparece em teste.
- Critério de conclusão: decisão registrada; prova do caminho escolhido.
- Reversão: reverter o commit.
- Esforço relativo: M (justificativa: regra jurídica com exceções e revisão do advogado).
- Papel responsável: Advogado + QA + Full-stack.
- Autorização necessária do usuário: sim (D-9).
- Explicação técnica: o sistema não aplica a suspensão do processo penal e mostra o vencimento mais curto. O erro é conservador, mas o vencimento pode ficar antes do real.
- Analogia: é como um calendário de prazos que não desconta a suspensão de fim de ano: o vencimento aparece antes do real. O erro vai a favor da segurança, mas continua sendo erro.

**3.7. T-3.07 — Validação da infraestrutura de produção numa VM de teste**
- Achado(s) de origem: 8.1 (P2, operacional).
- Objetivo: provar que o runbook sobe a infraestrutura e que o backup restaura, antes de qualquer dado real.
- Tipo: validação (verificação manual com procedimento).
- Localização: `deploy/` (cloud-init, nftables, timers systemd, túnel Cloudflare, backup e restauração); `PLANO-IMPLEMENTACAO.md` item 118 (relatório).
- Pré-condições e dependências: D-12; T-2.01 (coleta desligada no padrão); T-2.03 (migrações restauráveis); T-3.01; T-3.03.
- Passos de execução: (1) subir VM de teste conforme o runbook; (2) cloud-init e firewall: listar regras e portas abertas; (3) verificar timers systemd ativos; (4) túnel Cloudflare: acesso só pelo Access (verificação manual); (5) fazer backup, restaurar num banco novo e comparar contagens por tabela; (6) registrar em PROGRESSO_EXECUCAO.md.
- Risco de regressão: custo de recurso em nuvem; exposição indevida se o firewall estiver errado. A VM deve ser isolada e sem dados reais.
- Prova real: procedimento com saída registrada: regras do nftables com política padrão de bloqueio (A CONFIRMAR no runbook); `ss` mostrando só as portas previstas; `systemctl is-active` dos timers igual a `active`; contagens por tabela iguais entre origem e restauração.
- Critério de conclusão: checklist do runbook com saídas registradas; restauração verificada.
- Reversão: destruir a VM de teste. Sem dado real.
- Esforço relativo: G (justificativa: infraestrutura real, vários componentes e acesso externo).
- Papel responsável: SRE/DevOps + AppSec.
- Autorização necessária do usuário: sim. Cria recursos em nuvem e acesso externo.
- Explicação técnica: a infraestrutura nunca rodou numa VM real. Build da imagem e compose foram verificados. Cloud-init, firewall, timers, túnel e restauração não.
- Analogia: é como ensaiar o protocolo de contingência num cartório de teste antes de confiar nele em processos reais: o roteiro escrito não prova que a sala funciona.

**3.8. T-3.08 — Suíte local que aprova sem banco**
- Achado(s) de origem: 9.1 (P2).
- Objetivo: `go test` sem banco não retorna "ok" quando autorização, migrações, auditoria e fluxos de negócio não foram executados.
- Tipo: correção (processo de teste).
- Localização: pacote `internal/app` (testes de banco); script ou alvo de teste (A CONFIRMAR nome); variável `PAINEL_TEST_DB`.
- Pré-condições e dependências: T-0.03; D-7.
- Passos de execução: (1) prova antes: `go test ./...` sem `PAINEL_TEST_DB` (relatório: `ok` com 179 SKIP, a reconfirmar); (2) aplicar D-7: falha explícita com mensagem clara quando o banco faltar, ou alvo separado com nome distinto; (3) não remover `t.Skip` nem alterar testes para pular; (4) rodar com e sem banco.
- Risco de regressão: quem não tem banco não roda a suíte inteira. A CI precisa do banco configurado (A CONFIRMAR).
- Prova real: ANTES: `go test ./...` sem banco retorna `ok` com SKIP. DEPOIS: sem banco, código de saída diferente de zero e mensagem clara (se D-7 for falha por padrão); com banco, zero SKIP em `internal/app` e o mesmo número de execuções da linha de base.
- Critério de conclusão: provas acima.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: QA + SRE/DevOps.
- Autorização necessária do usuário: sim (D-7).
- Explicação técnica: sem `PAINEL_TEST_DB`, 179 das 323 execuções são puladas e o resultado sai como `ok`. As verificações de autorização, migrações e auditoria estão no pacote que só roda com banco.
- Analogia: é como declarar "todas as audiências realizadas" quando 179 foram canceladas sem registro: o relatório diz "ok" e esconde o que não foi examinado.

**Portão de saída da Onda 3:** as oito tarefas com prova aprovada; `go build`, `go vet` e testes com banco iguais à linha de base, exceto os testes dos pacotes removidos listados em PROGRESSO_EXECUCAO.md; `govulncheck` sem alcançáveis; `gosec` sem HIGH nem MEDIUM no código versionado; PROGRESSO_EXECUCAO.md atualizado; autorização para a Onda 4.

### ONDA 4 — P3 E P4: BAIXO E MELHORIAS (SOMENTE COM AUTORIZAÇÃO)

**4.1. T-4.01 — Teste de fim de linha (CRLF) no Windows**
- Achado(s) de origem: 1.2 (P3, ambiente).
- Objetivo: `go test` é sinal confiável neste PC, sem depender de fim de linha.
- Tipo: correção (teste).
- Localização: `internal/app/processos_test.go:124–125`; template `internal/pessoas/templates/lista.html` (relatório).
- Pré-condições e dependências: T-0.03 (falha conhecida registrada); D-14.
- Passos de execução: (1) prova antes: rodar o teste no Windows; (2) opção A: normalizar `\r\n` na comparação do teste (recomendada, mudança mínima); opção B: `.gitattributes` com `* text=auto eol=lf` (altera o repositório inteiro); (3) rodar no Windows e na CI Linux.
- Risco de regressão: a opção B pode gerar diff grande de fim de linha. A opção A pode esconder fim de linha se o template mudar, o que é aceitável numa comparação.
- Prova real: ANTES: o teste falha no Windows com as quatro asserções (relatório). DEPOIS: passa no Windows e passa na CI Linux (A CONFIRMAR: sem acesso à CI).
- Critério de conclusão: teste passa nos dois ambientes; a linha de base com banco perde essa falha.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: QA.
- Autorização necessária do usuário: sim, a escolha de D-14. A opção B mexe no repositório inteiro.
- Explicação técnica: a comparação exige `</div>\n    </div>`, mas com `core.autocrlf=true` o Windows grava `\r\n` na cópia de trabalho. A saída está correta; a comparação é literal demais.
- Analogia: é como conferir um despacho pela vírgula exata: a decisão está certa, mas a comparação literal falha pela formatação.

**4.2. T-4.02 — Processos sigilosos com texto integral na tabela de intimações**
- Achado(s) de origem: 2.6 (P3).
- Objetivo: a tabela não guarda a comunicação integral de processo sigiloso, salvo base legal decidida.
- Tipo: decisão do usuário + correção.
- Localização: `internal/intimacoes/sigiloso.go`; `internal/monitor/sigilosos.go:169–170` (campo `Texto`) (relatório).
- Pré-condições e dependências: D-10; T-0.01 (backup antes de qualquer expurgo).
- Passos de execução: (1) D-10 decide; (2) prova antes: registro sigiloso de teste grava `Texto` integral; (3) se resumir: gravar só o resumo; se manter: registrar a base legal no ROPA; (4) se houver registros existentes: plano de expurgo com backup prévio.
- Risco de regressão: expurgo é irreversível sem backup. Resumo pode perder informação necessária para o prazo.
- Prova real: ANTES: o registro sigiloso de teste grava `Texto` integral. DEPOIS: o registro sigiloso de teste grava só o resumo, e a consulta não retorna o texto integral; intimações não sigilosas iguais à linha de base.
- Critério de conclusão: decisão registrada no ROPA; prova acima.
- Reversão: restaurar do backup se houve expurgo; reverter o commit para o código.
- Esforço relativo: M (justificativa: decisão de base legal e possível expurgo).
- Papel responsável: Advogado/DPO + AppSec + Full-stack.
- Autorização necessária do usuário: sim.
- Explicação técnica: o campo `Texto` guarda a comunicação integral de processo sigiloso na tabela de intimações. Quem lê a tabela tem acesso ao conteúdo sigiloso.
- Analogia: é como guardar a íntegra de um processo sigiloso na pasta de rotina da secretaria: quem acessa a rotina passa a ver o que só o advogado da causa deveria ler.

**4.3. T-4.03 — Premissas do CPC 224 §1º e da Lei 11.419 art. 5º não implementadas**
- Achado(s) de origem: 4.4 (P3).
- Objetivo: as premissas não calculadas ficam explícitas no resultado e na documentação, com revisão após validação.
- Tipo: decisão do usuário + correção documental.
- Localização: `docs/juridico/REGRAS-DE-PRAZO.md`, seção "não considera" (relatório; linhas A CONFIRMAR).
- Pré-condições e dependências: T-2.02 (mesma documentação); validação do advogado.
- Passos de execução: (1) o advogado valida as premissas: expediente encerrado antes do horário, indisponibilidade do sistema e consulta em até 10 dias; (2) manter o aviso no resultado, sem mudança de cálculo; (3) atualizar a documentação com a decisão.
- Risco de regressão: documentação desatualizada em relação ao cálculo.
- Prova real: ANTES: o resultado mostra "não considera" para os três itens (registro em T-0.05). DEPOIS: mesmo resultado; a documentação lista os três itens com decisão e data.
- Critério de conclusão: documento atualizado com a decisão; resultado mantém o aviso.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: Advogado + Redator técnico.
- Autorização necessária do usuário: sim (validação jurídica).
- Explicação técnica: três premissas processuais não são calculadas e aparecem só como nota. A correção recomendada é manter o aviso e revisar após validação.
- Analogia: é como uma tabela de prazos que avisa "não considera indisponibilidade do sistema" sem calcular a alternativa: o advogado precisa conhecer a premissa para conferir a conta.

**4.4. T-4.04 — README contradiz o sistema**
- Achado(s) de origem: 4.5 (P3).
- Objetivo: o README descreve a regra real do sistema, sem frase duplicada.
- Tipo: correção (documentação).
- Localização: `README.md`, seção "Regras permanentes" (relatório).
- Pré-condições e dependências: T-2.02, T-3.05 e T-3.06 (decisões de prazo).
- Passos de execução: (1) reescrever a regra conforme as decisões de 4.1, 4.2 e 4.3; (2) remover a frase duplicada.
- Risco de regressão: texto novo que contradiz a decisão do advogado. Ler a decisão antes de escrever.
- Prova real: ANTES: a frase duplicada aparece em duas ocorrências (A CONFIRMAR o texto exato). DEPOIS: a frase duplicada aparece uma vez; a regra descrita bate com a decisão registrada (revisão manual).
- Critério de conclusão: `grep` verde; revisão manual registrada.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: Redator técnico + Full-stack.
- Autorização necessária do usuário: não, além da autorização da Onda 4.
- Explicação técnica: o README afirma que a automação não calcula prazo processual, mas o Plano 7 calcula prazos do CPC. A última frase da seção está duplicada.
- Analogia: é como um manual interno que diz que a secretaria não calcula prazo, quando ela calcula: quem confia no manual deixa de conferir a conta.

**4.5. T-4.05 — Validar índices em chaves estrangeiras**
- Achado(s) de origem: 5.2 (P3, HIPÓTESE A VALIDAR).
- Objetivo: confirmar ou refutar, por medição, a falta de índice nas chaves estrangeiras listadas.
- Tipo: validação (com correção condicionada ao resultado).
- Localização: 22 colunas `uuid REFERENCES`, 14 sem índice (relatório); exemplos `tarefa_pai_id`, `evento_id`, `oab_id`, `duplicado_de` (A CONFIRMAR nas migrações).
- Pré-condições e dependências: T-2.03; D-15.
- Passos de execução: (1) criar banco de teste com o volume de D-15; (2) `EXPLAIN ANALYZE` das remoções em cascata e das buscas pelas chaves; (3) critério de decisão: se confirmada (varredura sequencial relevante no volume realista), criar nova migração apenas com os índices confirmados, sem editar migrações antigas; se refutada, encerrar com registro.
- Risco de regressão: índice tem custo de escrita e de espaço. Criar sem medição pode piorar a escrita.
- Prova real: relatório de `EXPLAIN ANALYZE` antes, na mesma máquina e com o mesmo volume. Se confirmada, o plano depois da migração mostra uso de índice nas mesmas consultas.
- Critério de conclusão: decisão registrada com a evidência do plano.
- Reversão: `DROP INDEX` na migração nova (Down).
- Esforço relativo: M (justificativa: medição com volume realista).
- Papel responsável: Banco de dados e migrações.
- Autorização necessária do usuário: não para medir; sim para criar a migração, se confirmada.
- Explicação técnica: heurística sobre 22 colunas, não medida. Pode ser irrelevante em volume pequeno. A confiança do relatório nesse item é baixa.
- Analogia: é como verificar, antes de contratar mais um funcionário, se o arquivo está lento de verdade: a suspeita pode ser refutada por medição.

**4.6. T-4.06 — IDs malformados viram 500 em vez de 404**
- Achado(s) de origem: 6.1 (P3).
- Objetivo: ID fora do formato UUID responde 404, como já faz `GET /tarefas/{id}/editar`.
- Tipo: correção.
- Localização: `GET /processos/{id}`, `GET /pessoas/{id}`, `GET /prazos/{id}/editar`, `GET /documentos/{id}/baixar` (relatório); `webx.UUID`. Lista completa das rotas com `{id}`: A CONFIRMAR NO CÓDIGO.
- Pré-condições e dependências: T-0.03. Recomendado depois de T-3.04, porque toca os mesmos handlers.
- Passos de execução: (1) listar todas as rotas com `{id}` no registro de rotas; (2) prova antes: sonda `GET` com `abc` em cada rota; (3) escrever teste que espera 404 para formato inválido; (4) validar com `webx.UUID` antes da consulta, em cada rota; (5) repetir a sonda e o teste.
- Risco de regressão: mudança de código de resposta de rota que hoje responde outra coisa. Rotas que devolvem 404 para ID válido inexistente continuam assim.
- Prova real: ANTES: `GET /processos/abc` retorna 500 (sonda). DEPOIS: 404 para `abc` em todas as rotas listadas; ID UUID válido inexistente continua 404; ID válido existente continua 200 (suíte).
- Critério de conclusão: sonda e testes verdes para todas as rotas listadas.
- Reversão: reverter o commit.
- Esforço relativo: P (estimativa; sobe para M se a lista de rotas for longa).
- Papel responsável: Full-stack + QA.
- Autorização necessária do usuário: não, além da autorização da Onda 4.
- Explicação técnica: o id chega ao PostgreSQL sem validação. O erro de conversão de uuid vira 500 e entra no log como falha do sistema.
- Analogia: é como um protocolo que, diante de número digitado errado, responde "erro do sistema" em vez de "processo não encontrado": o erro é de entrada, e o log passa a registrar falha que não é do sistema.

**4.7. T-4.07 — Dockerfile e compose da raiz divergem da imagem oficial**
- Achado(s) de origem: 8.2 (P3).
- Objetivo: a raiz não mantém uma segunda definição de imagem e banco fora de `deploy/`.
- Tipo: correção (remoção ou apontar para `deploy/`).
- Localização: `Dockerfile` e `docker-compose.yml` da raiz (não versionados); `deploy/Dockerfile`; compose de `deploy/` (relatório).
- Pré-condições e dependências: T-3.01 (decisão sobre arquivos da raiz); T-0.01.
- Passos de execução: (1) prova antes: `grep -n "FROM" Dockerfile` mostra `golang:alpine` e `alpine:latest`; `grep -n "postgres:" docker-compose.yml` mostra `postgres:16` (relatório); (2) conforme D-3: remover ou apontar para `deploy/`; (3) se mantido para desenvolvimento: alinhar o Postgres à versão 17 da CI e da produção; (4) repetir os `grep`.
- Risco de regressão: quem usa a raiz para desenvolver perde o fluxo. Trocar Postgres 16 por 17 em desenvolvimento altera o volume de dados local.
- Prova real: ANTES: os `grep` acima. DEPOIS, se removido: os arquivos não existem e `docker compose config` da compose de `deploy/` funciona; se mantido: `grep` mostra `postgres:17` e nenhuma tag móvel.
- Critério de conclusão: prova conforme a escolha.
- Reversão: restaurar da cópia de T-0.01 (não versionados).
- Esforço relativo: P.
- Papel responsável: SRE/DevOps.
- Autorização necessária do usuário: sim. Remoção de arquivos não versionados (D-3).
- Explicação técnica: a raiz tem um Dockerfile com tag móvel e execução como root, e um compose com Postgres 16. A CI e a produção usam 17 e imagem por digest.
- Analogia: é como ter duas versões do regimento interno do escritório, uma com "versão atual" sem data: quem segue a errada aplica regra que muda sem aviso.

**4.8. T-4.08 — Sudo sem senha e grupo docker no operador**
- Achado(s) de origem: 8.3 (P3). Sem correção recomendada no relatório: a decisão é do usuário.
- Objetivo: decidir e registrar se o operador único mantém o privilégio ou se a função é separada.
- Tipo: decisão do usuário.
- Localização: `cloud-init.yaml` (`painel-admin`: `NOPASSWD:ALL`; grupo `docker`) (relatório).
- Pré-condições e dependências: D-13; T-3.07 (VM para verificar).
- Passos de execução: (1) D-13 decide; (2) se aceitar, registrar o risco com responsável e data na documentação de segurança; (3) se separar, ajustar o cloud-init (remover `NOPASSWD:ALL` ou o grupo `docker`) e validar na VM.
- Risco de regressão: separar exige rever a rotina de operação. Manter aceita um risco equivalente a root.
- Prova real: ANTES: `sudo -l` e `id painel-admin` na VM de teste mostram `NOPASSWD` e o grupo `docker`. DEPOIS, se separar: os mesmos comandos não mostram esses privilégios, e o deploy segue funcionando na VM. DEPOIS, se aceitar: o registro escrito existe.
- Critério de conclusão: decisão registrada; prova da escolha.
- Reversão: reverter a alteração do cloud-init e reaplicar na VM.
- Esforço relativo: M se separar; P se aceitar.
- Papel responsável: SRE + AppSec.
- Autorização necessária do usuário: sim (D-13).
- Explicação técnica: `painel-admin` tem sudo sem senha e pertence ao grupo `docker`, o que equivale a root no host.
- Analogia: é como dar a um único funcionário a chave do cofre e a combinação: aceitável num escritório de uma pessoa, mas sem conferência de dupla.

**4.9. T-4.09 — Restauração executa o manifesto com `source`**
- Achado(s) de origem: 8.4 (P4).
- Objetivo: a restauração lê as chaves do manifesto sem executá-las.
- Tipo: correção.
- Localização: `deploy/backup/restore.sh` (relatório).
- Pré-condições e dependências: T-0.03.
- Passos de execução: (1) prova antes: num manifesto de teste, uma linha com substituição de comando cria um arquivo-sentinela (defeito); (2) trocar o `source` por leitura chave-valor com validação de formato; (3) rodar a restauração com manifesto válido e com o manifesto malicioso de teste.
- Risco de regressão: valores com `=` ou espaços podem quebrar a restauração. Testar com manifesto real de teste.
- Prova real: ANTES: o arquivo-sentinela é criado. DEPOIS: o arquivo-sentinela não é criado; o manifesto válido produz as mesmas variáveis que o `source` produzia (comparação de saída). Rodar num ambiente Linux (A CONFIRMAR).
- Critério de conclusão: as duas provas.
- Reversão: reverter a alteração.
- Esforço relativo: P.
- Papel responsável: SRE + AppSec.
- Autorização necessária do usuário: não, além da Onda 4.
- Explicação técnica: o restore executa qualquer linha do manifesto. A mitigação é o manifesto estar cifrado com `age`, o que deixa o risco dependente de quem tem a chave.
- Analogia: é como um cartório que executa as ordens escritas numa ficha de processo em vez de apenas ler os campos: quem escreve a ficha passa a mandar no cartório.

**4.10. T-4.10 — htmx sem versão registrada e sem SRI**
- Achado(s) de origem: 7.1 (P4).
- Objetivo: o arquivo vendorizado tem versão registrada e integridade verificável.
- Tipo: correção (integridade e documentação).
- Localização: `web/static/htmx.min.js`; `base.html` (relatório).
- Pré-condições e dependências: nenhuma.
- Passos de execução: (1) identificar a versão e a origem (A CONFIRMAR: o arquivo não tem cabeçalho); (2) registrar versão e hash SHA-384 no repositório; (3) acrescentar o atributo `integrity` à tag `<script>` em `base.html`; (4) não trocar o arquivo sem decisão.
- Risco de regressão: se o arquivo não bater com nenhuma versão oficial, o hash não corresponde à origem. Nesse caso, tratar como A CONFIRMAR e não substituir sem decisão.
- Prova real: `openssl dgst -sha384 -binary web/static/htmx.min.js | openssl base64 -A` igual ao valor de `integrity`; página carrega sem erro de console (navegador, manual).
- Critério de conclusão: hash coincide; página carrega.
- Reversão: remover o atributo `integrity`.
- Esforço relativo: P.
- Papel responsável: Full-stack + AppSec.
- Autorização necessária do usuário: não, além da Onda 4.
- Explicação técnica: o htmx está minificado, sem versão, e é carregado sem SRI. Não há como provar que o arquivo servido é o que foi revisado.
- Analogia: é como arquivar uma peça sem data nem número de juntada: depois não se sabe se é a versão que foi assinada.

**4.11. T-4.11 — Cache de CEP sem expurgo**
- Achado(s) de origem: 2.7 (P4).
- Objetivo: o cache tem validade e teto de entradas.
- Tipo: correção.
- Localização: `internal/cep/handler.go:79, 93` (relatório).
- Pré-condições e dependências: T-0.03.
- Passos de execução: (1) prova antes: teste insere N entradas já expiradas e verifica que todas permanecem; (2) remover expirados e definir teto; (3) repetir o teste com `-race`.
- Risco de regressão: teto baixo aumenta consultas externas. Expurgo concorrente exige sincronização (A CONFIRMAR se já há mutex).
- Prova real: ANTES: após N inserções expiradas, o tamanho do cache é N. DEPOIS: o tamanho fica até o teto; expirados removidos; teste passa com `-race`.
- Critério de conclusão: testes verdes com `-race`.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: Full-stack + QA.
- Autorização necessária do usuário: não, além da Onda 4.
- Explicação técnica: o cache é um mapa que só cresce. Teto e expurgo impedem o crescimento sem limite.
- Analogia: é como uma pasta de consulta rápida que nunca é esvaziada: cresce até o armário não fechar.

**4.12. T-4.12 — Limitador de taxa sem teto de chaves**
- Achado(s) de origem: 2.8 (P4).
- Objetivo: o mapa de chaves do limitador tem teto rígido.
- Tipo: correção.
- Localização: `internal/platform/ratelimit/ratelimit.go:33` (relatório).
- Pré-condições e dependências: T-0.03.
- Passos de execução: (1) prova antes: teste insere K chaves distintas e mostra crescimento sem teto; (2) definir teto e remoção de chaves expiradas antes de qualquer corte; (3) repetir; (4) verificar que o limite por chave continua valendo.
- Risco de regressão: remover chave ativa zera o contador e permite exceder o limite. Por isso só se removem chaves expiradas antes de cortar.
- Prova real: ANTES: após K inserções, o mapa tem K chaves. DEPOIS: o mapa fica até o teto; o teste de limite por chave passa; `-race` limpo.
- Critério de conclusão: as provas acima.
- Reversão: reverter o commit.
- Esforço relativo: P.
- Papel responsável: AppSec + Full-stack.
- Autorização necessária do usuário: não, além da Onda 4.
- Explicação técnica: a limpeza só roda acima de 10.000 chaves e não há limite rígido. O impacto é baixo porque o acesso passa antes pelo Cloudflare Access.
- Analogia: é como um livro de controle de visitas sem limite de páginas: funciona até encher, e a falta de limite só aparece com o tempo.

**Portão de saída da Onda 4:** cada tarefa com prova aprovada; suíte sem regressão em relação à linha de base; PROGRESSO_EXECUCAO.md atualizado.

---

## SEÇÃO C — MATRIZ DE RASTREABILIDADE

| Achado | Gravidade | Tarefa(s) | Situação |
|---|---|---|---|
| 1.1 | P1 | T-0.04, T-2.01 | Correção |
| 1.2 | P3 | T-0.03, T-4.01 | Correção |
| 1.3 | P2 | T-0.01, T-3.01 | Correção (D-3) |
| 2.1 | P2 | T-3.02 | Correção (remoção) |
| 2.2 | P2 | T-3.04 | Correção (D-5) |
| 2.3 | P3 | T-3.02 | Correção (remoção) |
| 2.4 | P3 | T-3.01 | Correção |
| 2.5 | P2 | T-3.03 | Correção (D-4) |
| 2.6 | P3 | T-4.02 | Correção (D-10) |
| 2.7 | P4 | T-4.11 | Correção |
| 2.8 | P4 | T-4.12 | Correção |
| 3.1 | P3 | T-3.02 | Correção |
| 3.2 | P4 | — | Não tratado: auditoria de 05/11 aceitou adiar (D-17) |
| 4.1 | P1 | T-0.05, T-2.02 | Correção condicionada à reprodução e à decisão do advogado |
| 4.2 | P2 | T-3.05 | Correção (D-6) |
| 4.3 | P2 | T-3.06 | Correção (D-9) |
| 4.4 | P3 | T-4.03 | Correção documental (validação do advogado) |
| 4.5 | P3 | T-4.04 | Correção documental |
| 5.1 | P1 | T-2.03 | Correção (D-11) |
| 5.2 | P3 | T-4.05 | Validação (hipótese) |
| 6.1 | P3 | T-4.06 | Correção |
| 7.1 | P4 | T-4.10 | Correção |
| 7.2 | — | — | Lacuna de cobertura, não é achado (D-18) |
| 8.1 | P2 | T-3.07 | Validação manual |
| 8.2 | P3 | T-4.07 | Correção (D-3) |
| 8.3 | P3 | T-4.08 | Decisão do usuário (D-13) |
| 8.4 | P4 | T-4.09 | Correção |
| 9.1 | P2 | T-3.08 | Correção (D-7) |
| 9.2 | P2 | T-0.04, T-2.01 | Teste e correção |
| 9.3 | P3 | T-3.02 | Correção (gate) |
| 9.4 | P4 | — | Não tratado: ambiente, sem correção no código (D-17) |

---

## SEÇÃO D — DECISÕES PENDENTES DO USUÁRIO

- **D-1. Alterações rastreadas não commitadas** (`cmd/painel/main.go`, `go.mod`, `internal/auth/commonpw.go`). Recomendação: `main.go` volta ao HEAD em T-2.01; `go.mod` decide-se em T-3.03; `commonpw.go` decide-se após ver o diff (E-1). Bloqueia: Onda 0 e T-2.01.
- **D-2. Local da cópia de segurança** (T-0.01). Recomendação: fora do OneDrive, porque a cópia tem segredos de teste e `painel.exe`. Bloqueia: T-0.01.
- **D-3. O que é versionado** (1.3, 2.4, 8.2). Recomendação: ignorar `painel.exe` e os segredos; manter o relatório fora do repositório; decidir os arquivos da raiz. Bloqueia: T-3.01, T-4.07.
- **D-4. Versão alvo do Go** (2.5). Opções: 1.27.3 (a da imagem, com as correções) ou linha 1.26.x com patch (não verificada). Recomendação: 1.27.3, alinhando `go.mod`, CI e imagem. Bloqueia: T-3.03.
- **D-5. Regra de visibilidade** (2.2): por responsável, por equipe ou combinação. Decisão do escritório, sem recomendação técnica. Bloqueia: T-3.04.
- **D-6. Calculadora** (4.2): bloquear a criação de prazo enquanto a flag estiver desligada, ou aceitar o risco por escrito. Recomendação: bloquear, porque o controle atual é só o banner. Bloqueia: T-3.05.
- **D-7. Suíte sem banco** (9.1): falhar por padrão, ou alvo separado. Recomendação: falhar por padrão. Bloqueia: T-3.08.
- **D-8. Prazo em dobro** (4.1): o advogado consulta o CPC art. 229, decide e valida os casos de T-0.05. Bloqueia: T-2.02.
- **D-9. Suspensão penal** (4.3): implementar com as exceções dos incisos I a III, ou manter o aviso. Decisão do advogado. Bloqueia: T-3.06.
- **D-10. Processos sigilosos** (2.6): avaliação LGPD, base legal e escolha entre resumo ou texto integral. Bloqueia: T-4.02.
- **D-11. Migrações** (5.1): existe hoje algum banco com dados reais? Método: roteiro com migração nova (recomendado) ou alterar as guardas (só se nenhum banco aplicou). Bloqueia: T-2.03.
- **D-12. VM de teste** (8.1, 8.3): onde, quem executa e acesso à Cloudflare e à Oracle (sem acesso no relatório). Bloqueia: T-3.07.
- **D-13. Sudo sem senha e grupo docker** (8.3): aceitar com registro escrito, ou separar funções. Bloqueia: T-4.08.
- **D-14. CRLF** (1.2): opção A, normalizar a comparação no teste (recomendada), ou opção B, `.gitattributes`, que altera o repositório inteiro. Bloqueia: T-4.01.
- **D-15. Volume para o `EXPLAIN`** (5.2): o tamanho de dados que o usuário considera realista. Bloqueia: T-4.05.
- **D-16. Extração em `main.go`** (1.1, 9.2): se T-0.04 não puder testar `serve()` sem extração, autorizar a menor extração possível. Bloqueia: T-0.04, se necessário.
- **D-17. Não tratados** (3.2 e 9.4): aceitar o adiamento já decidido na auditoria de 05/11 e a ausência de ação sobre o ambiente Windows. Não bloqueia tarefas.
- **D-18. Lacuna 7.2** (acessibilidade e responsividade): encomendar validação com navegador ou aceitar a lacuna. Não bloqueia tarefas.
- **D-19. Autorização por onda** (seção 9 do comando): Onda 0, depois Onda 2, e assim por diante. Onda 4 só se autorizada. Exclusões de código (T-3.02), push para a branch remota (T-3.03) e criação de recursos em nuvem (T-3.07) pedem confirmação separada.

---

## SEÇÃO E — OCORRÊNCIAS FORA DO RELATÓRIO

- **E-1.** `internal/auth/commonpw.go` aparece como modificado no `git status` inicial. O relatório não menciona a alteração e cita o arquivo só como lista de dados sem lógica. Não entra no plano. Ver o diff antes de decidir (D-1).
- **E-2.** O relatório menciona segredos de teste em `D:\OneDrive\Desktop\00-REFUGOS\2026-11-08\`, sem analisar o risco de a pasta ser sincronizada pelo OneDrive. Não entra no plano. A cópia de T-0.01 segue D-2.
- **E-3.** O projeto está em `D:\OneDrive\Desktop\PAINEL-TEODORO - Copia`, fora de `2º Cérebro/01-PROJETOS/`, que é o lugar previsto pela regra de onde o trabalho mora. Não é achado de código. Não entra no plano.

---

## SEÇÃO F — COBERTURA E LIMITAÇÕES DO PLANO

- **Cobertura.** 30 achados numerados (7.2 é lacuna, não achado). 28 têm tarefa. 2 não são tratados, com justificativa (3.2 e 9.4). [Certeza]
- **Causa-raiz.** O campo não aparece em 24 dos 30 achados. As causas usadas no plano são inferidas da evidência do relatório. [Provável]
- **Campo "correção recomendada" ausente.** Em 3.2 (há justificativa de adiamento) e em 8.3 (tratado como decisão D-13). Em 9.4 a correção é "nenhuma no código".
- **Dependências.** O relatório declara "Relações" em poucos itens. As dependências do plano são inferidas e estão na Seção B.
- **Lidos por amostra.** Podem conter achados não identificados: agenda, atendimentos, audiências, diligências, tarefas (serviço), processos (serviço e repositório), pessoas (serviço e repositório), intimações (coleta e serviço), monitor (coleta), indices, cep (handler), calendário (serviço), `auth/wa.go`, `auth/totp.go`, `auth/admin.go`, `auth/account.go`, `auth/reauth.go` e `cli.go`. T-3.04 e T-4.06 tocam essa área.
- **Não examinados.** `referencia/painel-b`, `tools/refb`, `docs/` além das ADRs, REGRAS-DE-PRAZO e PENDENCIAS, e a lista de senhas comuns.
- **Não executados.** golangci-lint, gitleaks, trivy, check-links (python3 ausente), `analise-estatica.sh` (grava no projeto), navegador, carga e benchmark, cloud-init e firewall em VM.
- **Sem acesso.** CI remota, Oracle, Cloudflare e produção.
- **Performance (Seção 10).** Não são achados: listas sem paginação e coleta do DataJud sequencial, sem medição. Ficam fora do plano.
- **Verificações que dependem de ambiente.** PostgreSQL 17 (T-0.03, T-2.03, T-4.05); VM (T-3.07, T-4.08); navegador (T-4.10 e lacuna 7.2); CI remota (T-3.03, T-4.01). Se algum estiver indisponível, a tarefa para na prova e volta ao usuário.
- **A CONFIRMAR NO CÓDIGO.** Valores padrão de `PAINEL_*_ATIVO` no HEAD (T-2.01); se `serve()` é testável (T-0.04); se existe `.gitignore` e o que contém (T-3.01); se os diretórios de `internal/` não listados (por exemplo `handler/` e `crypto/`) têm outros pacotes (T-3.02); campo de responsável em processos e pessoas (T-3.04); lista de rotas com `{id}` (T-4.06); cobertura de `prazos/calculo` (T-0.05); arquivo de CI e como lê a versão do Go (T-3.03); origem e versão do htmx (T-4.10); conteúdo do runbook (T-3.07); se o `Dockerfile` da raiz é usado por algo (T-4.07).
- **Normas não verificadas.** Texto do CPC art. 229, do CPC art. 224 §1º, da Lei 11.419 art. 5º e da Lei 14.365/2022. Este plano não afirma o conteúdo delas. Ficam com o advogado.
- **Confiança.** Alta: 1.1 (leitura de código; comportamento não executado com rede), 1.2, 6.1, 9.1, 9.2 e 5.1. Média: 2.2, 2.5 e 3.1 (escopo sujeito às decisões). Baixa: 5.2. Para 4.1, 4.3 e 4.4, a confiança sobre a norma é nula até validação do advogado. A ordem das ondas é [Provável]. Os esforços P, M e G são estimativas sem medição.
- **Cobertura de código.** Não medida. Não há percentual.
