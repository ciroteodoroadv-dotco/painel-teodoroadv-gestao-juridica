---
projeto: 2026-tec-fusao-paineis
tipo: especificacao
data: 2026-11-03
estado: RASCUNHO para aprovação do advogado
decisao-base: "sim" do advogado em 03/11/2026 à opção B (visual do painel B portado para templates Go + htmx no v2)
---

# Especificação da fusão — visual do B sobre o backend do v2

## 1. Objetivo
O advogado vê e usa o painel **como desenhou no painel B** (layout, cores, tipografia, componentes, comportamentos de interação), rodando sobre o **backend seguro do v2** (login com 2º fator, RBAC, auditoria, PostgreSQL, backup cifrado). O B não é alterado nem apagado.

## 2. Não-objetivos
- Copiar o JS, o backend ou os dados do B para dentro do v2 sem passar pelas portas G0–G5 do handoff.
- Reaproveitar o calendário de feriados / `motorprazo` do B como motor legal (inventário §3).
- Ligar WhatsApp, F13 ou integrações externas nesta fase.

## 3. Invariantes (do v2, não negociáveis)
CSP estrita (sem inline script/style, sem `unsafe-*`); html/template com escape; CSRF por HMAC; RBAC deny-by-default; auditoria encadeada; domínios só conversam por interfaces; arquivos de produção ≤ 300 linhas; prazo só vira "confirmado" por advogado; sistema nunca é a única fonte de prazo.

## 4. Estratégia de portar o visual
1. **Tokens** do B (`--ink, --paper, --panel, --wine, --brass, --text, --border…`, raio 3px, Arial, 14px, sidebar 230px, drawer 480px) viram o `app.css` do v2 por substituição controlada; cores soltas do B (`#F7F7F5`, `#D9DEE2`, `#c0392b`…) entram como tokens nomeados.
2. **Componentes** a reproduzir em templates/partials: sidebar preta com marca latão e logo circular; pílulas de categoria (`acc-head`) com contador; tabelas densas zebradas; selos (`badge-*`) sempre com texto; drawer lateral que empurra a página (sem véu, **sem ESC e sem clique fora — pedido do advogado**, com pilha de drawers); toast 3,2 s; faixa de índices no rodapé; painel "Estado das coletas"; estados vazios tracejados.
3. **Interação** por htmx + um `app.js` pequeno com **delegação de eventos e `data-*`** (zero `onclick`, zero `style=`). Isso fecha também a classe de XSS por nome de arquivo em handler inline do B.
4. **Navegação:** abas Agenda (padrão), Atendimentos, Audiências, Pessoas, Processos, Tarefas + calculadora de prazo (drawer). Melhoria permitida sem alterar o visual: URL com hash/rota (F5 não volta à Agenda).
5. **Logo** reduzido (hoje 1200×1200/108 KB exibido a 150 px) sem mudar o aspecto.
6. **Acessibilidade:** corrigir apenas o que não altera o visual (anel de foco apagado por `all:unset`, `label for`, contraste do ticker) — cada item listado e aprovado.

## 5. Mapeamento de telas B → domínios v2
| Tela do B | Domínio v2 hoje | Lacuna |
|---|---|---|
| Tarefas | `tarefas` | estados do B (6 pastas), prioridade, prazo fatal, documentos PDF |
| Processos | `processos` | financeiro, próxima providência, andamentos, partes/responsáveis |
| Pessoas | `pessoas` | CEP (ViaCEP), máscaras/validação CPF/CNPJ (já há `internal/platform/br`) |
| Atendimentos | — | **novo domínio** |
| Audiências | — | **novo domínio** (6 tipos, cancelamento com reautenticação) |
| Diligências | — | **novo domínio** |
| Agenda | `agenda` | categorias Reunião/Ligação/Aula/Prazo fatal, conflito de horário (crítico = audiência/prazo fatal) |
| Intimações | — | **novo** (depende da Fase B do roteiro: coletor DJEN) |
| Prazos / calculadora | `prazos` | calendário forense por tribunal (Fase E) |
| Índices BCB | — | pequeno coletor + faixa no rodapé |

Cada novo domínio segue `handler→service→repo`, migração goose, permissão RBAC e auditoria; a fronteira entre domínios é por interface.

## 6. Prazos (regra de ouro mantida)
- Calculadora do v2 mostra sempre **"A CONFERIR"** até confirmação por advogado; não herda "Prazo fatal" sem ressalva do B.
- Calendário forense por tribunal com **fonte citada por feriado/suspensão** (Fase E); até lá, aviso explícito de que feriados locais, dobro e termo inicial por juntada/citação/carga não são tratados.
- Intimação detectada → **sugere** prazo → advogado confirma (decisão de 02/11).

## 7. Portas de qualidade (G0–G5, do handoff — documento externo, não versionado)
- **G0** decisões fechadas (arquitetura = B, aprovada; destino dos dados reais = **pendente**).
- **G1** inventário do B — **concluído** (`01-referencias/painel-B-inventario.md`).
- **G2** auditoria do que for reaproveitado (coletores DJEN/DataJud, lógica de dedupe) antes de portar.
- **G3** tokens + componentes no v2 e **comparação lado a lado por captura de tela** de cada tela com o B (critério de aceite visual).
- **G4** domínios novos + testes; `go vet`, gosec, staticcheck, govulncheck, `-race` limpos; teste de fronteiras.
- **G5** uso real em localhost com ficha atualizada; só então migração de dados.

## 8. Critérios de aceite
1. Cada tela do B tem tela equivalente no v2, aprovada pelo advogado por comparação visual.
2. Nenhum `on*=` nem `style=` em HTML gerado; CSP sem relaxar.
3. Nenhuma alteração em `painel-teodoro-entrega-1-1-2026-10-30` (conferível por hash antes/depois).
4. Todo prazo exibido como "a conferir" até confirmação; testes provam que assistente não confirma.
5. Backup e restauração continuam passando (`restore-check`).

## 9. Riscos
- Perder detalhes de interação que o advogado ajustou à mão (drawer, acordeões): mitigado por G3.
- Escopo grande (5 domínios novos): mitigado por fatias verticais (uma tela por vez, começando por Agenda e Tarefas).
- Dados reais do B em claro no OneDrive: nada é migrado antes do G5 e da decisão de destino.

## 10. Decisão pendente do advogado
Destino dos dados reais do B: **migrar para o v2** (exige F13 antes) ou **v2 começa limpo** e recadastra. Não impede G2–G4.
