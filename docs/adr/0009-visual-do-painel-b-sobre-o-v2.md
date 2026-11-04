# ADR 0009 — Visual do painel B sobre o v2

**Status:** aceita em 03/11/2026 (determinação do advogado: "o frontend do painel B prevalece, sem exceção"); executada nos Planos 1B a 8.

**Contexto:** o advogado já trabalha no painel B e quer a mesma tela, os mesmos textos e os mesmos fluxos. O v2 tem o que o B não tem: regras de segurança
(2º fator, papéis, auditoria encadeada, dados pessoais cifrados, nada se apaga). Fundir os dois exige decidir quem manda em quê.

**Decisão:** **o B vale na forma; o v2 vale na regra.**

1. A casca (sidebar, topbar, estatísticas, conteúdo, rodapé de coletas, faixa de índices, drawer, toast), o CSS (`painel.css`, copiado com hash idêntico) e os textos vêm do B,
   conferidos por leitura do código do B e por comparação de estilo computado (relatórios em `docs/plano-fusao/paridade/`).
2. Onde o B conflita com a segurança do v2, a tela fica igual e a regra é a do v2 (exemplos: o selo "A CONFERIR" aparece no campo do prazo fatal; a tarefa não tem exclusão).
3. O que o B não tem (Minha conta, Usuários, Auditoria, Prazos, Sair, Calendário forense, Coleta DJEN, Índices) fica num segundo grupo do menu, com o mesmo estilo dos itens do B.
4. O B é **somente leitura** (hashes em `HASHES-B.txt`; nenhum dado do B é aberto).

**O que mudou em relação ao v2 anterior:** casca e menu novos; "Hoje" deixou de existir (a Agenda é a página inicial); o **modo escuro foi removido** (o B não tem);
`web/static/compat.css` guarda o que o v2 precisa além do B (classes de aviso, de erro de formulário, do selo, do rodapé de índices), sem alterar `painel.css`.
Nada de `on*=`, `style=` ou script inline: a CSP continua estrita (`default-src 'self'`), então onde o B usa `onclick` o v2 usa `data-*` e JS em arquivo.

**Consequências:** qualquer ajuste visual novo parte do B; diferença de estilo só entra se justificada em relatório de paridade. O custo é manter o `compat.css` pequeno.
