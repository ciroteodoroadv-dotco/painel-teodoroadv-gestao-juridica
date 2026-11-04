#!/usr/bin/env python3
"""Plano 7: compara, texto a texto, o que o painel B escreve na Calculadora de prazo (12-prazo.js) e no botao da
sidebar (07-app-core.js) com o que o v2 escreve. Extrai do B (somente leitura) rotulos, dicas, titulos, botoes e
mensagens; procura cada um nos templates e no codigo do v2. Imprime so o que NAO foi achado; cada item restante tem de
ser justificado em NOTAS-PLANO-7.md.

Uso: python3 tools/refb/textos7.py [pasta web do B]
"""
import html
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[2]
B = Path(sys.argv[1] if len(sys.argv) > 1 else
         "/mnt/d/OneDrive/Desktop/painel-teodoro-entrega-1-1-2026-10-30/painel-teodoro/web")
MODULOS_B = ["js/12-prazo.js", "js/07-app-core.js#calculadora"]
FONTES_V2 = (list((RAIZ / "internal/prazos").rglob("*.html")) + list((RAIZ / "internal/prazos").rglob("*.go")) +
             list((RAIZ / "internal/auth/templates").glob("*.html")) + list((RAIZ / "internal/auth").glob("*.go")) +
             list((RAIZ / "web/static").glob("*.js")))
# Textos que o B monta por expressao (${...}) ou em partes.
COMPOSTOS = {"js/12-prazo.js": ["Número de dias inválido.", "Falha ao calcular o prazo.", "Prazo fatal: ", "vence hoje",
                                "faltam", "dia(s) útil(eis)", "venceu há"],
             "js/07-app-core.js#calculadora": ["Calculadora de prazo"]}
PADROES = [
    (r">([^<>${}\n][^<>${}]*)</(?:label|legend|h2|h3|p|button|th|span|option|div|strong)>", "texto de tag"),
    (r'placeholder="([^"$]+)"', "placeholder"),
    (r"showToast\('([^'$]+)'", "toast"),
]


def main():
    v2 = "\n".join(f.read_text(encoding="utf-8", errors="replace") for f in FONTES_V2)
    v2n = html.unescape(v2)
    faltam, total = {}, 0
    for mod in MODULOS_B:
        src = (B / mod.split("#")[0]).read_text(encoding="utf-8")
        if "#" in mod:  # so a linha do botao da sidebar
            src = "\n".join(l for l in src.splitlines() if "botaoCalculadora" in l)
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
            if t not in v2 and t not in v2n:
                faltam.setdefault(mod, []).append((tipo, t))
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
