#!/usr/bin/env bash
# Retenção: mantém os 7 mais recentes + o mais novo de cada uma das últimas 4 semanas ISO
# + o mais novo de cada um dos últimos 12 meses. Apaga o resto.   Uso: prune.sh DIR [--dry-run]
source "$(dirname "$0")/lib.sh"

dir="${1:?uso: prune.sh DIR [--dry-run]}"
dry="${2:-}"
mapfile -t files < <(cd "$dir" && ls -1 painel-*.tar.age 2>/dev/null | sort -r)
((${#files[@]})) || { log "nada para podar"; exit 0; }

declare -A keep=()
n=0
for f in "${files[@]}"; do
  ((n++ < 7)) && keep[$f]=daily
done

# O mais novo de cada semana/mês (a lista já vem do mais novo para o mais antigo).
declare -A seen_w=() seen_m=()
for f in "${files[@]}"; do
  d="${f#painel-}"; d="${d:0:8}"
  w="$(date -u -d "${d:0:4}-${d:4:2}-${d:6:2}" +%G-W%V)"
  m="${d:0:6}"
  if [[ -z "${seen_w[$w]:-}" && ${#seen_w[@]} -lt 4 ]]; then seen_w[$w]=$f; keep[$f]="${keep[$f]:-weekly}"; fi
  if [[ -z "${seen_m[$m]:-}" && ${#seen_m[@]} -lt 12 ]]; then seen_m[$m]=$f; keep[$f]="${keep[$f]:-monthly}"; fi
done

removed=0
for f in "${files[@]}"; do
  if [[ -z "${keep[$f]:-}" ]]; then
    if [[ "$dry" == "--dry-run" ]]; then echo "APAGARIA $f"; else rm -f -- "$dir/$f" "$dir/$f.sha256"; fi
    removed=$((removed + 1))
  fi
done
log "mantidos $((${#files[@]} - removed)), ${dry:+(simulação) }removidos $removed"
