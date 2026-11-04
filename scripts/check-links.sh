#!/usr/bin/env bash
# Verifica referências internas da documentação: links relativos [texto](caminho)
# e caminhos de repositório citados em crase (`docs/...`, `scripts/...` etc.).
# Ignora docs/arquivo/ (registro histórico) e links http(s)/mailto/âncoras.
set -euo pipefail
cd "$(dirname "$0")/.."
python3 - <<'PY'
import os, re, sys, glob
raiz = os.getcwd()
arquivos = ["README.md"] + [p for p in glob.glob("docs/**/*.md", recursive=True)
                             if not p.startswith("docs/arquivo/")]
prefixos = ("docs/", "scripts/", "internal/", "cmd/", "deploy/", "migrations/",
            "web/", "referencia/", ".github/", ".claude/")
quebrados = []
excecoes = set()
if os.path.exists("scripts/check-links.excecoes"):
    for linha in open("scripts/check-links.excecoes", encoding="utf-8"):
        if linha.strip() and not linha.startswith("#"):
            f, alvo, _ = linha.rstrip("\n").split("|", 2)
            excecoes.add((f, alvo))
for f in sorted(arquivos):
    texto = open(f, encoding="utf-8").read()
    for m in re.finditer(r"\[[^\]]*\]\(([^)\s]+)\)", texto):
        alvo = m.group(1).split("#")[0]
        if not alvo or re.match(r"^(https?:|mailto:)", alvo):
            continue
        destino = os.path.normpath(os.path.join(os.path.dirname(f), alvo))
        if not os.path.exists(destino):
            quebrados.append((f, alvo))
    for m in re.finditer(r"`([^`\s]+)`", texto):
        cand = m.group(1)
        if not cand.startswith(prefixos) or any(c in cand for c in "*<>{}"):
            continue
        if not os.path.exists(cand.split("#")[0]):
            quebrados.append((f, cand))
novos = [(f, a) for f, a in quebrados if (f, a) not in excecoes]
print(f"Arquivos verificados: {len(arquivos)}")
print(f"Exceções registradas (scripts/check-links.excecoes): {len(quebrados) - len(novos)}")
for f, alvo in novos:
    print(f"QUEBRADO: {f} -> {alvo}")
print(f"Referências quebradas sem exceção: {len(novos)}")
sys.exit(1 if novos else 0)
PY
