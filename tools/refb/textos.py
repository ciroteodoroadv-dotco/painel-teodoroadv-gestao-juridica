#!/usr/bin/env python3
"""Compara, texto a texto, o que o painel B escreve nas telas Pessoas e Processos com o que o v2 escreve.

Extrai do B (somente leitura) os módulos 14-pessoas.js e 09-processos.js (e as listas de 05/06):
rótulos, legendas, títulos, subtítulos, dicas, placeholders, textos de botão, de cabeçalho, de toast,
de confirmação e de opção de lista. Procura cada um nos templates e no código do v2.
Imprime só o que NÃO foi achado; cada item restante tem de ser justificado em NOTAS-PLANO-2.md.

Uso: python3 tools/refb/textos.py [pasta web do B]
"""
import html
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
B = Path(sys.argv[1] if len(sys.argv) > 1 else
         "/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/painel-teodoro/web")
MODULOS_B = ["js/14-pessoas.js", "js/09-processos.js"]
LISTAS_B = [("js/05-dominio.js", ["estadoCivil", "classeProcessual", "materiaDireito", "estadoProcesso"]),
            ("js/06-helpers-vinculos.js", ["PAPEIS_PROCESSUAIS"]),
            ("js/09-processos.js", ["SISTEMAS_PROCESSO", "INSTANCIAS_PROCESSO"])]
# Onde o v2 escreve: templates, Go dos dois domínios e os JS comuns.
FONTES_V2 = (list((RAIZ / "internal/pessoas").rglob("*.html")) + list((RAIZ / "internal/pessoas").glob("*.go")) +
             list((RAIZ / "internal/processos").rglob("*.html")) + list((RAIZ / "internal/processos").glob("*.go")) +
             list((RAIZ / "web/static").glob("*.js")))
# Textos que o B monta por expressão (`${...}`) e que o v2 escreve por inteiro.
COMPOSTOS = {"js/14-pessoas.js": ["Salvar alterações", "Salvar cadastro", "Nova pessoa", "Editar pessoa"],
             "js/09-processos.js": ["Salvar alterações", "Salvar processo", "Novo processo", "Editar processo"]}

PADROES = [
    (r">([^<>${}\n][^<>${}]*)</(?:label|legend|h2|h3|p|button|th|span|option|div|strong)>", "texto de tag"),
    (r'placeholder="([^"$]+)"', "placeholder"),
    (r"showToast\('([^'$]+)'", "toast"),
    (r"(?:titulo|mensagem|labelConfirmar):\s*'([^'$]+)'", "confirmação"),
    (r"(?:mensagemOk|mensagemErro)?\s*'((?:Estado do processo|Andamento|Dado financeiro|Próxima providência|Falha)[^'$]*)'", "mensagem de ação"),
]


def v2_texto():
    return "\n".join(f.read_text(encoding="utf-8", errors="replace") for f in FONTES_V2)


def listas(b):
    saida = {}
    for arq, nomes in LISTAS_B:
        src = (b / arq).read_text(encoding="utf-8")
        for n in nomes:
            m = re.search(r"(?:\b%s\s*[:=]\s*)\[([^\]]*)\]" % re.escape(n), src)
            if m:
                saida[n] = re.findall(r"'([^']+)'", m.group(1))
    return saida


def main():
    v2 = v2_texto()
    v2n = html.unescape(v2)
    faltam, total = {}, 0
    for mod in MODULOS_B:
        src = (B / mod).read_text(encoding="utf-8")
        achados = []
        for pad, tipo in PADROES:
            for m in re.finditer(pad, src):
                t = html.unescape(m.group(1)).strip()
                if len(t) >= 2 and not re.fullmatch(r"[\W\d_]+", t):
                    achados.append((tipo, t))
        for t in COMPOSTOS[mod]:
            achados.append(("composto", t))
        for tipo, t in sorted(set(achados)):
            total += 1
            if t not in v2 and t not in v2n and t.replace("\\'", "'") not in v2n:
                faltam.setdefault(mod, []).append((tipo, t))
    for nome, itens in listas(B).items():
        for t in itens:
            total += 1
            if t not in v2n:
                faltam.setdefault("lista " + nome, []).append(("opção", t))
    print(f"{total} textos do B conferidos contra o v2")
    n = 0
    for mod, itens in faltam.items():
        for tipo, t in itens:
            n += 1
            print(f"FALTA [{mod}] ({tipo}): {t}")
    print(f"{n} diferença(s) a justificar" if n else "zero diferenças")
    return 1 if n else 0


if __name__ == "__main__":
    sys.exit(main())
