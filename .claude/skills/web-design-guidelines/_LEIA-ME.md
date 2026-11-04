---
titulo: web-design-guidelines — como e por que
tipo: nota-de-ferramenta
data: 2026-09-14
---

# O que é

Skill oficial da Vercel (`vercel-labs/web-interface-guidelines`) para revisar código de
interface web contra as Web Interface Guidelines — um conjunto de boas práticas de UI/UX
e acessibilidade mantido pela Vercel.

# Quando aciona

Ao pedir para revisar UI, conferir acessibilidade, auditar design ou revisar UX de código de
interface web. Recebe um arquivo ou padrão de arquivos (`<arquivo-ou-padrao>`); se nenhum for
indicado, pergunta.

# Como funciona

A cada execução, busca a versão mais recente das diretrizes direto do repositório da Vercel
(via WebFetch) — não usa uma cópia fixa. Lê os arquivos indicados, aplica as regras buscadas
e devolve os achados no formato `arquivo:linha`.

# Tradução (14/09/2026)

O `SKILL.md` foi traduzido do inglês ao português a pedido do usuário, mantendo estrutura,
passos e conteúdo técnico (nada foi adicionado, cortado ou reinterpretado). O `name`
permanece em inglês, conforme protocolo do acervo. O conteúdo buscado em tempo de execução
(as próprias diretrizes, na URL-fonte) não é traduzido — vem sempre em inglês, direto da
fonte, e é isso que garante que a skill nunca fica desatualizada.

# Instalação

Skill **sob demanda**: fica inteira no acervo, sem vínculo em `~/.claude/skills/`, e é
acionada abrindo-se este `SKILL.md` pelo caminho quando a tarefa pedir revisão de UI/UX.
Ver `_SKILLS-SOB-DEMANDA.txt` e `CLAUDE.md`.

# Fonte

https://github.com/vercel-labs/web-interface-guidelines (texto original fornecido pelo
usuário no pedido de criação da skill).
