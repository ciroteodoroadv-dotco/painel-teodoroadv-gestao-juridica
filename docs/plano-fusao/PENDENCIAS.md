# Pendências consolidadas da fusão (Planos 1B a 8)

Fonte: `NOTAS-PLANO-*.md`, `ACEITE-VISUAL.md`, `docs/operacao/*.md`, `docs/juridico/REGRAS-DE-PRAZO.md`. "NP-n Nk" = nota k do plano n. O risco é avaliação do desenvolvedor, não está nas fontes.
Levantamento feito por subagente de leitura e conferido por amostra. **Nenhuma pendência dos NOTAS-PLANO foi omitida de propósito**; se faltar alguma, acrescente aqui.

## Plano 1B
| Item | Quem decide | Risco |
|---|---|---|
| Login do advogado: casca logada nunca vista por quem vai usá-la (AV Plano 1) | advogado | médio |
| Comparação tela a tela com o B só por leitura de código; capturas 1440x900 e 400x800 não feitas (AV, todos os planos) | advogado / desenvolvedor | baixo |
| Lista de prazos continua no corpo da Agenda, não na faixa (NP-1B N2) | advogado | baixo |
| Subtítulos de Agenda, Processos e Tarefas diferem do B (NP-1B N4) | advogado | baixo |
| `#syncStatus`: JS do B não portado (NP-1B N8); contador da Agenda soma só eventos (N9) | desenvolvedor | baixo |
| `-race` só em contêiner (N12); TDD parcial em Tarefas 2–4 (N13) | desenvolvedor | baixo |
| Envio de formulário em drawer não exercitado por clique no Chrome (AV 1B, 2, 3) | desenvolvedor | médio |

## Plano 2
| Item | Quem decide | Risco |
|---|---|---|
| Omissão do campo "Nome do arquivo da ficha" (NP-2 N1) | advogado | baixo |
| "Responsável pelo cadastro" cai em "Parte adversa" (N3) | advogado | médio |
| Texto/botão de exclusão diferem do B (N2); páginas `/pessoas/{id}` e `/processos/{id}` não existem no B (N12) | advogado | baixo |
| CEP e drawers de substituição de endereço: resolvidos no Plano 8 (N13); "Atendimento de origem" e "monitoramento" resolvidos nos Planos 3 e 6 (N14, N15) | desenvolvedor | baixo (conferir) |
| Migração 0008 recusa aplicar com pessoas/processos no banco (N9) | administrador | **alto** |
| Commits das Tarefas 1 e 2 não verdes isolados (N16) | desenvolvedor | baixo |

## Plano 3
| Item | Quem decide | Risco |
|---|---|---|
| Fluxo do recibo (abrir "Novo processo" a partir do PDF) não reproduzido (NP-3 N1) | advogado | médio |
| Exclusão de tarefa superada pelo 4C (N2) | advogado | baixo |
| `prazo_fatal` da tarefa não acompanha cumprir/cancelar feitos em Prazos (N5); falha do Prazos após gravar a tarefa, sem transação única (N6) | desenvolvedor | médio |
| Administrador não escreve atendimentos/tarefas (N8, N18); assistente não vê "Novo cliente" (N9) | advogado | baixo |
| Itens a mais que o B: A CONFERIR, "Confirmar prazo", "Tarefa vinculada a" (N11); busca sem acento (N14) | advogado | baixo |
| Upload de PDF e "Novo cliente" no atendimento não exercitados no Chrome (AV 3) | desenvolvedor | médio |
| Migração 0009 só aplica com `tarefas` vazia (N16) | administrador | **alto** |

## Plano 4 e 4C
| Item | Quem decide | Risco |
|---|---|---|
| Conflito de horário: só audiência bloqueia, hora exata, edição não checa (NP-4 N2); audiência cancelada não bloqueia; erro é 400 e não 409 | advogado / desenvolvedor | médio |
| Tipo "Diligência" na edição de compromisso aparece no select e é recusado (N4); edição de audiência sem "Observações" (N7) | desenvolvedor / advogado | baixo |
| Permissões: audiência/diligência escritas por advogado e assistente; cancelar audiência só advogado, com senha (N6, N8) | advogado | médio |
| `Reauth` da demonstração aceita qualquer senha (N14) — só `SeedDemo`, em dev | desenvolvedor | médio |
| Cancelar compromisso/diligência/intimação: motivo sem senha (NP-4C N2); cancelado continua nos contadores (N3) | advogado | médio |
| Intimação `descartada` virou `cancelada`; reversão da 0012 recusa com cancelamentos/duplicatas (N4, N8) | administrador | médio |
| Coluna `excluido_em` de tarefas sem uso (N1) | desenvolvedor | baixo |
| Processo duplicado: regra só no serviço, sem trava no banco (N6); justificativa anterior só na auditoria; consulta por POST (N7) | advogado | médio |
| JS do número duplicado e cliques reais não vistos no Chrome (AV 4C); capturas (AV 4) | desenvolvedor | médio |

## Plano 5 (DJEN)
| Item | Quem decide | Risco |
|---|---|---|
| "Base da autorização" no lugar de "Consentimento"; precisa constar no inventário LGPD (NP-5 N1) | advogado | **alto** |
| Swagger do DJEN deu 403; parâmetros vieram do código do B, [Provável] (N2) | desenvolvedor | **alto** |
| Paginação: página repetida/vazia vira erro (N3); janela por OAB (N4) | advogado / desenvolvedor | médio |
| Autor das tarefas automáticas = quem cadastrou a OAB (N5); tarefa sem processo não aceita PDF (N6); coluna mostra a chave "a-confirmar" (N7) | advogado | baixo |
| WhatsApp não reproduzido (N12); segredo de justiça por credencial própria fora do plano (N12) | advogado | médio |
| Tabela de Intimações montada na Agenda (N10); teste de suíte falhou uma vez sem reprodução (N16) | desenvolvedor | médio |
| Teste manual orientado antes de ligar a coleta (AV 5) | advogado | **alto** |

## Plano 6 (DataJud)
| Item | Quem decide | Risco |
|---|---|---|
| "Uma tarefa por processo todos os dias" interpretada como "no dia em que há novidade" (NP-6 N1); consolidada também no DJEN, só com processo cadastrado (N2) | advogado | médio |
| Prazo interno de 2 dias corridos nasce A CONFERIR (N3); falha ao encerrá-lo deixa estado parcial | advogado / desenvolvedor | médio |
| Backfill da tarefa de monitoramento para processos antigos (N5) | advogado | **alto** |
| Texto limitado a 100.000 caracteres e sem quebras de parágrafo (N6); DataJud não dá o teor (N7) | desenvolvedor / advogado | médio |
| DataJud: documentação deu 403; formato do coletor do B nunca testado na API real (N9, MD pend. 1) | desenvolvedor | **alto** |
| Alias `tre-dft` é [Chute] (MD pend. 2); `numeroProcesso` do DJEN é [Provável] (MD pend. 3) | desenvolvedor | médio |
| STF e CNJ sem endpoint: processo "bloqueado" (N10) | advogado | médio |
| Primeira leitura com zero hits fica "parcial" (N11); sigilosos: janela de 30 dias e dependem de DJEN ligado (N13) | advogado | médio |
| Chave ausente não impede a subida (N14) | administrador | baixo |
| Teste manual com 1 processo e conferência do advogado (AV 6; MD) | advogado | **alto** |

## Plano 7 (Calculadora e calendário)
| Item | Quem decide | Risco |
|---|---|---|
| Sem prazo em dobro (CPC 180, 183, 186, 229); aparece em "Não considerado" (NP-7 N1) | advogado | **alto** |
| Fontes "Não lida": CPP 798, CC 132, CP 10, Lei 5.010/66 art. 62, Lei 662/49, Lei 9.093/95, Lei 11.419/06 (N2) | advogado | **alto** |
| Sábado não prorrogado no CPP e nos materiais [VERIFICAR] (N3) | advogado | **alto** |
| CPP no modo DJe: dois termos, exibe o mais curto [VERIFICAR] (N4); prazo material ignora o modo (N5) | advogado | **alto** / médio |
| Recesso 20/12–20/01 só no CPC [VERIFICAR] (N6); calendário do tribunal fora do CPC só "feriado" (N7) | advogado | **alto** / médio |
| Norma de cada feriado fixo a conferir (N9); Sexta-feira Santa [VERIFICAR a norma] (N9) | advogado | médio |
| Calendário da Justiça Federal só na demonstração e [VERIFICAR]; produção nasce vazio (N10) | advogado | médio |
| Ponto facultativo só por advogado (N13); prazo da calculadora nasce A CONFERIR (N14); teto 3650 dias (N19) | advogado / desenvolvedor | baixo |
| `PAINEL_CALCULADORA_REGRAS_VALIDADAS` ausente = aviso vermelho; coluna "Validado por" em branco (N18) | advogado | **alto** |
| Gatilho contra DELETE nos calendários; 0014 só reverte sem dados (N12) | administrador | médio |

## Plano 8 (Índices, CEP e fechamento)
| Item | Quem decide | Risco |
|---|---|---|
| Significado de cada série do BCB (1619 anual ou mensal? 1 PTAX de venda ou compra?) não conferido (NP-8 N1) | advogado | **alto** |
| Série 1619 pode ficar "desatualizada" (N2); valor antigo permanece visível com a marca (N3); falha de leitura mantém o último texto (N4) | advogado | médio |
| Consulta real a BCB e ViaCEP não verificada — integrações nunca ligadas (N11) | advogado | **alto** |
| CEP: só advogado consulta; 20/min por usuário; cache de 10 min (N7) | advogado | médio |
| Dependência `golang.org/x/crypto` v0.56.0: GO-2026-5932 (corrigida só em pseudo-versão, não chamada pelo código); reexecutar `scripts/analise-estatica.sh` quando sair a versão marcada | desenvolvedor | baixo |
| Teste ponta a ponta no navegador (Tarefa 5): ver `docs/arquivo/plano-fusao/paridade/E2E-8.md` | desenvolvedor | médio |

## Operação (transversal)
| Item | Quem decide | Risco |
|---|---|---|
| Oracle, Cloudflare, `cloud-init`, `bootstrap.sh`, `nftables`, timers, túnel, Object Storage e `release.yml` NÃO testados em VM real (RUNBOOK-ORACLE) | administrador | **alto** |
| Inventário LGPD aprovado, DPA da Cloudflare e segunda cópia do backup (RUNBOOK-ORACLE) | advogado | **alto** |
| "Não coloque dado real até o M5"; backup só testado localmente (RUNBOOK; COMECAR-USO-REAL) | advogado | **alto** |
| Cópia externa do backup (rclone) e agendamento no Windows não testados (RELATORIO-8) | administrador | médio |

## Decisões do advogado, em linguagem leiga
1. Ler as leis citadas de memória (CPP, Código Civil, Código Penal, leis de feriado e da Justiça Federal, Lei 11.419) e assinar a validação da calculadora.
2. Sábado não prorroga o vencimento do prazo penal e do material (hoje o sistema mostra o dia mais curto): aceita?
3. No prazo penal pelo Diário, vale a publicação ou a disponibilização?
4. O recesso de 20/12 a 20/01 vale para prazos penais e materiais?
5. Confirmar que a calculadora não faz prazo em dobro.
6. Fazer o teste combinado de ligar DJEN e DataJud com um único processo.
7. Criar ou não a tarefa de monitoramento para processos já cadastrados.
8. Confirmar o que é "uma tarefa por processo, todos os dias".
9. Aprovar a retirada do "Nome do arquivo da ficha" e decidir o grupo de "responsável pelo cadastro".
10. Quem cancela o quê e se compromisso, diligência e intimação pedem senha como a audiência.
11. Conferir o significado das séries do Banco Central antes de confiar no rodapé.
12. Aprovar inventário LGPD, contrato da Cloudflare e destino da segunda cópia do backup antes de usar dado real.

## Fundação (M1) — migrado do PLANO-IMPLEMENTACAO.md em 04/11/2026
| Item | Quem decide | Risco |
|---|---|---|
| Lint com golangci-lint v2.5.0 no job `go`: **verificado** no CI run 36 (`df1f679`, 05/11/2026), "0 issues" | desenvolvedor | resolvido |
| Job `migracoes` (ciclo up→down→up): **verificado** no CI run 36 (`df1f679`, 05/11/2026) | desenvolvedor | resolvido |
| SBOM (SPDX) no CI: implementado no job `docker` (05/11/2026) com `anchore/sbom-action` v0.24.3 fixada pelo commit `66cbf4b…` (verificado: tag → commit no histórico de `main`); sem execução de CI ainda | desenvolvedor | baixo |
| trivy (vulnerabilidades) no CI: implementado no job `docker` (05/11/2026) com `aquasecurity/trivy-action` v0.36.0 fixada pelo commit `ed142fd…` (verificado: tag → commit no histórico de `master`); sem execução de CI ainda. Se a imagem tiver CRÍTICA/ALTA com correção, o CI falha — é o comportamento pretendido | desenvolvedor | médio |
| `release.yml` fixa o trivy em `a9c7b0f…`, que é o objeto da tag anotada v0.36.0, não o commit (`ed142fd…`). Não verificado se o Actions aceita SHA de objeto de tag; padronizar para o commit, como as demais ações | desenvolvedor | médio |

