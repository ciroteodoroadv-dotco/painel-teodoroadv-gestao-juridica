#!/usr/bin/env bash
# Gera os segredos locais em deploy/secrets/ (ignorado pelo Git). Nunca sobrescreve.
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p secrets
chmod 700 secrets
gerar() { # nome, comando
  if [ -e "secrets/$1" ]; then
    echo "já existe, mantido: secrets/$1"
  else
    ( umask 077; eval "$2" > "secrets/$1" )
    chmod 444 "secrets/$1" # o contêiner roda sem root e precisa ler
    echo "criado: secrets/$1"
  fi
}
gerar db_password "openssl rand -hex 24"
gerar db_app_password "openssl rand -hex 24"
gerar session_key "openssl rand -base64 32"
gerar field_key "openssl rand -base64 32"
echo "Pronto. Guarde cópia destes 4 arquivos em local seguro: sem field_key os dados cifrados não abrem."
