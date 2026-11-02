# Roteiro — prazos e integrações (DJEN, DataJud) — proposta de 02/11/2026

**Prioridade do sistema, por decisão do advogado:** acompanhamento real de prazos. Perda de prazo é falta grave e gera
responsabilidade civil. **Este roteiro é proposta: nada aqui foi implementado nem aprovado.**

## O que as fontes oficiais entregam (conferido por pesquisa; parâmetros exatos a conferir no Swagger)

| Fonte | O que é | Limites que importam |
|---|---|---|
| **DJEN** — `comunicaapi.pje.jus.br/api/v1/comunicacao` | API pública, sem autenticação; consulta por OAB+UF, nome, nº do processo, tribunal e período | limite por IP (`429` → esperar 1 min); páginas de 5 ou 100; máx. 10.000 resultados por busca; **não é "tempo real"** (é consulta repetida); abuso com vários IPs é vedado. Se traz conteúdo de processo em **segredo de justiça**: **não verificado** — testar com OAB real |
| **DataJud** — `api-publica.datajud.cnj.jus.br/api_publica_<tribunal>/_search` | API pública com chave pública do CNJ (publicada na wiki, pode mudar); capa e **movimentos** | **não traz processo sigiloso**; não traz o teor da intimação; frequência de atualização **não confirmada** (tratar como defasada); um endpoint por tribunal |
| Segredo de justiça | exige credencial do advogado (PJe, MNI, Domicílio Judicial Eletrônico, Portal PDPJ) | decisão jurídica e técnica própria — ver Fase D |

## Princípios (inegociáveis para esta frente)

1. **O sistema nunca é a única fonte de prazo.** Mantém-se a redundância (painel do tribunal, e-mail do PJe, agenda do advogado).
2. **Detectar é automático; sugerir é automático e rotulado; confirmar é humano e obrigatório.** O prazo só "vale" no sistema
   depois que o advogado responsável confirma (quem, quando, fonte). Isto **revisa a decisão 5 do plano-base** ("automação não
   calcula prazo") e **depende da aprovação do advogado**.
3. **Monitor silencioso é pior que nenhum monitor.** Todo coletor tem batimento (heartbeat) e alarme se parar.
4. **"Sem resultado" nunca significa "sem prazo".** Processo sigiloso ou fora do alcance da fonte aparece como "SEM COBERTURA", não como "nada a fazer".
5. **Coleta educada:** respeitar limites, 1 IP, backoff; sem tentar burlar.

## Fases (ordem sugerida)

| Fase | Entrega | Critério de aceite |
|---|---|---|
| **A — uso diário em localhost (antes do uso real)** | entidade **Prazo** própria (data fatal, fundamento, fonte, responsável, confirmado por/quando, cumprido); tela "Prazos críticos" (hoje, 3, 5, 10 dias, atrasados); alertas visuais; backup local agendado + cópia fora do PC; checklist de segurança do PC | prazo vencendo nunca fica escondido; backup restaurado em teste |
| **B — DJEN por OAB** | cadastro de advogados parceiros (OAB/UF, **consentimento** registrado); coletor periódico (ex.: a cada 15–30 min); armazenamento deduplicado das comunicações; vínculo ao processo pelo nº CNJ; **pendência "a conferir"** com sugestão de prazo; heartbeat | comunicação nova gera pendência visível em minutos; parada do coletor gera alarme; testes com respostas gravadas |
| **C — DataJud** | monitor de movimentos dos processos cadastrados (diferença entre leituras); "SEM COBERTURA" para sigilosos | movimento novo vira aviso; sigiloso nunca aparece como "sem movimentação" |
| **D — segredo de justiça** | decisão: provedor comercial autorizado **ou** credencial do próprio advogado (certificado/gov.br) por API oficial; piloto com 1 advogado | consentimento por escrito; segredo (certificado/token) fora do Git e cifrado; auditoria de cada consulta |
| **E — calculadora de prazo** | regras do CPC (dias úteis, exclusão do dia do começo, termo inicial da intimação, prazo em dobro, recesso) + **calendário forense por tribunal/comarca**; sugestão sempre rotulada | validada em casos reais conferidos por advogado antes de ligar; divergência registrada |
| **F — notificações** | e-mail interno escalonado (D-5, D-3, D-1, dia); depois outros canais **após revisão própria** (WhatsApp segue vedado) | alerta chega; falha de envio é visível |

## Antes de qualquer dado real

O M0 (inventário LGPD, guarda das chaves, DPA) continua pendente. Dado real em `localhost` é decisão do advogado, **sob
risco assumido**; ver o checklist mínimo na resposta de 02/11/2026 e a seção correspondente do handoff.
