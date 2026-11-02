#!/usr/bin/env bash
# Funções comuns. Nunca imprime senha, chave ou conteúdo de cliente.
set -euo pipefail

log() { printf '%s [%s] %s\n' "$(date -u +%FT%TZ)" "${0##*/}" "$*"; }
die() { log "ERRO: $*"; exit 1; }

# Lê o valor de um segredo: variável VAR_FILE (arquivo) tem prioridade sobre VAR.
secret() {
  local name="$1" file_var="${1}_FILE"
  if [[ -n "${!file_var:-}" ]]; then
    [[ -r "${!file_var}" ]] || die "não consigo ler o arquivo de segredo de $name"
    tr -d '\r\n' < "${!file_var}"
  elif [[ -n "${!name:-}" ]]; then
    printf '%s' "${!name}"
  else
    die "segredo ausente: defina $name ou ${name}_FILE"
  fi
}

# Nome padrão: painel-AAAAMMDDTHHMMSSZ.tar.age (UTC) — a política de retenção lê a data deste nome.
backup_name() { printf 'painel-%s.tar.age' "$(date -u +%Y%m%dT%H%M%SZ)"; }
