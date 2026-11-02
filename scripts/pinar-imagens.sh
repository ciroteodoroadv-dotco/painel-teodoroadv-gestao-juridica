#!/usr/bin/env bash
# Mostra o digest ATUAL de cada imagem base usada (tag → sha256), para atualizar os arquivos à mão.
# Ao trocar um digest: reconstruir, rodar os testes e os dois scripts de teste de deploy/ antes de commitar.
# Quando o repositório estiver no GitHub, o Dependabot (.github/dependabot.yml) faz isto por pull request.
set -euo pipefail
cd "$(dirname "$0")/.."
echo "Imagens referenciadas hoje:"
grep -rnE '^FROM |^ +image: ' deploy/Dockerfile deploy/backup/Dockerfile deploy/docker-compose.yml deploy/docker-compose.prod.yml \
  | grep -v 'PAINEL_' | sed 's/^/  /'
echo; echo "Digests atuais (índice multi-arquitetura):"
for img in golang:1.26.10-bookworm gcr.io/distroless/static-debian12:nonroot alpine:3.21 postgres:17-alpine cloudflare/cloudflared:2025.8.1; do
  d="$(docker buildx imagetools inspect "$img" 2>/dev/null | awk '/^Digest:/{print $2; exit}')"
  printf '  %-48s %s\n' "$img" "${d:-(não consegui resolver)}"
done
