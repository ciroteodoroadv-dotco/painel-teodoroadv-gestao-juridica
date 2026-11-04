---
name: web-design-guidelines
description: Revisa código de interface web quanto à conformidade com as Web Interface Guidelines. Use quando pedirem "revise minha UI", "confira acessibilidade", "audite o design", "revise a UX" ou "confira meu site contra as boas práticas".
metadata:
  author: vercel
  version: "1.0.0"
  argument-hint: <arquivo-ou-padrao>
---

# Web Interface Guidelines

Revisa arquivos quanto à conformidade com as Web Interface Guidelines.

## Como funciona

1. Busca a versão mais recente das diretrizes na URL-fonte abaixo
2. Lê os arquivos indicados (ou pergunta ao usuário quais arquivos/padrão usar)
3. Confere cada regra das diretrizes buscadas
4. Devolve os achados no formato enxuto `arquivo:linha`

## Fonte das diretrizes

Busque as diretrizes atualizadas antes de cada revisão:

```
https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md
```

Use o WebFetch para buscar as regras mais recentes. O conteúdo buscado já traz todas as regras
e as instruções de formato de saída.

## Uso

Quando o usuário indicar um arquivo ou padrão:
1. Busque as diretrizes na URL-fonte acima
2. Leia os arquivos indicados
3. Aplique todas as regras das diretrizes buscadas
4. Devolva os achados no formato especificado pelas diretrizes

Se nenhum arquivo for indicado, pergunte ao usuário quais arquivos revisar.
