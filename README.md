# Painel Teodoro v2

Reescrita do painel do escritório para Linux/Oracle Cloud, privado, com LGPD por desenho.

**Estado (04/11/2026):** fundação M1 concluída; M2 (identidade, RBAC, auditoria) e M3 (núcleo funcional) concluídos; fusão visual com o painel B em andamento (`docs/plano-fusao/00-SEQUENCIA.md`).
Como ligar: `docs/operacao/RUNBOOK.md`.

## Onde ler

- `docs/plano-fusao/ESPECIFICACAO-FUSAO.md` e `docs/plano-fusao/00-SEQUENCIA.md` — o que governa o visual e a sequência atual.
- `docs/plano-fusao/PENDENCIAS.md` — pendências com quem decide e o risco.
- `docs/ESPECIFICACAO.md` e `docs/PLANO-IMPLEMENTACAO.md` — histórico da base técnica (M0 a M6).
- `docs/adr/` — decisões de arquitetura e o porquê de cada uma.
- `docs/lgpd/` — modelos (em branco) para o advogado preencher.

## Regras permanentes

- Nunca colocar credenciais, PDFs reais, CPF, conteúdo processual ou `.env` no Git, em prompt, log ou captura.
- Arquivos de produção têm no máximo 300 linhas.
- Integrações externas (WhatsApp, e-mail, DataJud, DJEN) ficam desligadas até revisão específica.
- Automação não calcula prazo processual nem decide providência jurídica. Pode criar prazo interno de confirmação (criação + 2 dias corridos), sempre marcado A CONFERIR até confirmação do advogado (ADR 0012).
