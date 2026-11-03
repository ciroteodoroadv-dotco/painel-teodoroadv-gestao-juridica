#!/usr/bin/env bash
# Falha se algum arquivo de produção passar de 300 linhas (ADR 0006).
set -euo pipefail
cd "$(dirname "$0")/.."
LIMITE=300
falhou=0
while IFS= read -r -d '' f; do
  n=$(wc -l < "$f")
  if [ "$n" -gt "$LIMITE" ]; then
    echo "ERRO: $f tem $n linhas (limite $LIMITE)"
    falhou=1
  fi
done < <(find . \( -name .git -o -name node_modules -o -path ./internal/db/gen \) -prune -o \
  -type f \( -name '*.go' -o -name '*.html' -o -name '*.css' -o -name '*.sql' \) \
  ! -name '*_test.go' ! -name painel.css -print0)
[ "$falhou" -eq 0 ] && echo "OK: nenhum arquivo de produção acima de $LIMITE linhas."
exit "$falhou"
