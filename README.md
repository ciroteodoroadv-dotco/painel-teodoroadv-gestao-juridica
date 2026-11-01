# Painel Teodoro v2

Reescrita do painel do escritório para Linux/Oracle Cloud, privado, com LGPD por desenho.

**Estado:** fase de especificação. Não há código ainda.

## Onde ler

- `docs/ESPECIFICACAO.md` — o que será construído e como.
- `docs/PLANO-IMPLEMENTACAO.md` — marcos M0 a M6 e critérios de aceite.
- `docs/adr/` — decisões de arquitetura e o porquê de cada uma.
- `docs/lgpd/` — modelos (em branco) para o advogado preencher.

## Regras permanentes

- Nunca colocar credenciais, PDFs reais, CPF, conteúdo processual ou `.env` no Git, em prompt, log ou captura.
- Arquivos de produção têm no máximo 300 linhas.
- Integrações externas (WhatsApp, e-mail, DataJud, DJEN) ficam desligadas até revisão específica.
- Automação não calcula prazo nem decide providência jurídica.
