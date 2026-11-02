#!/usr/bin/env bash
# Teste da política de retenção com arquivos FICTÍCIOS (vazios). Roda dentro da imagem de backup.
set -euo pipefail
d="$(mktemp -d)"
trap 'rm -rf "$d"' EXIT

mk() { : > "$d/painel-$1T030000Z.tar.age"; : > "$d/painel-$1T030000Z.tar.age.sha256"; }
# 1 backup por dia durante 400 dias, terminando em 2026-11-01.
for i in $(seq 0 399); do mk "$(date -u -d "2026-11-01 -$i days" +%Y%m%d)"; done

"$(dirname "$0")/prune.sh" "$d" >/dev/null
total=$(ls "$d"/painel-*.tar.age | wc -l)
falha=0
check() { if [[ -e "$d/painel-$1T030000Z.tar.age" ]]; then [[ "$2" == manter ]] || { echo "ERRO: $1 deveria ter sido apagado"; falha=1; }
          else [[ "$2" == apagar ]] || { echo "ERRO: $1 deveria ter sido mantido"; falha=1; }; fi; }

# Os 7 mais recentes ficam.
for i in $(seq 0 6); do check "$(date -u -d "2026-11-01 -$i days" +%Y%m%d)" manter; done
# Um dia comum de 40 dias atrás, que não é o mais novo do mês nem da semana, some.
check 20250101 apagar
check 20260920 apagar
# O mais novo de cada mês fica: 31/10, 30/09 (último dia de cada mês nos últimos meses).
check 20261031 manter; check 20260930 manter; check 20260831 manter
# Mais velho que 12 meses (exceto os mantidos) some.
check 20250601 apagar
# O .sha256 acompanha o arquivo apagado.
[[ ! -e "$d/painel-20250101T030000Z.tar.age.sha256" ]] || { echo "ERRO: sidecar .sha256 sobrou"; falha=1; }
# Teto: 7 + 3 semanas extras (no máx.) + 12 meses (no máx.) = no máximo 7+4+12.
(( total <= 23 )) || { echo "ERRO: sobraram $total arquivos (máx. 23)"; falha=1; }
(( total >= 12 )) || { echo "ERRO: sobraram só $total arquivos"; falha=1; }
# Simulação não apaga nada.
before=$(ls "$d" | wc -l); "$(dirname "$0")/prune.sh" "$d" --dry-run >/dev/null; [[ "$(ls "$d" | wc -l)" == "$before" ]] || { echo "ERRO: --dry-run apagou"; falha=1; }
# Pasta vazia não dá erro.
e="$(mktemp -d)"; "$(dirname "$0")/prune.sh" "$e" >/dev/null || { echo "ERRO: pasta vazia"; falha=1; }; rmdir "$e"

((falha)) && exit 1
echo "OK: política de retenção conforme (mantidos $total de 400)"
