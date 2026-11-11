#!/usr/bin/env bash
# AnÃ¡lise estÃ¡tica de seguranÃ§a (Plano 8): gosec, staticcheck e govulncheck, no contÃªiner de teste.
# As ferramentas sÃ£o instaladas SÃ“ no contÃªiner (versÃµes fixas); nada entra no projeto nem no binÃ¡rio.
# Grava o relatÃ³rio em docs/plano-fusao/ANALISE-ESTATICA.md. Sai com erro se houver achado alto/mÃ©dio do gosec,
# qualquer achado do staticcheck ou vulnerabilidade alcanÃ§Ã¡vel pelo cÃ³digo (govulncheck).
# Uso: scripts/analise-estatica.sh          (na raiz do repositÃ³rio; precisa do Docker e do python3)
set -uo pipefail
cd "$(dirname "$0")/.."
IMG="${IMAGEM_GO:-golang:1.26.11-bookworm}"
GOSEC=v2.30.0            # github.com/securego/gosec/v2
STATICCHECK=v0.8.2       # honnef.co/go/tools (staticcheck)
GOVULN=v1.9.0            # golang.org/x/vuln (govulncheck)
saida="docs/plano-fusao/ANALISE-ESTATICA.md"
tmp="$(mktemp -d)"; chmod 777 "$tmp"
trap 'rm -rf "$tmp"' EXIT

docker run --rm -v "$PWD":/src:ro -v "$tmp":/saida -w /src -e GOFLAGS=-buildvcs=false -e GOCACHE=/tmp/gocache "$IMG" sh -c "
  go install github.com/securego/gosec/v2/cmd/gosec@$GOSEC &&
  go install honnef.co/go/tools/cmd/staticcheck@$STATICCHECK &&
  go install golang.org/x/vuln/cmd/govulncheck@$GOVULN || exit 99
  gosec -quiet -exclude-generated -track-suppressions -fmt=json -out=/saida/gosec.json ./... >/dev/null 2>&1; echo \$? > /saida/gosec.rc
  staticcheck ./... > /saida/staticcheck.txt 2>&1; echo \$? > /saida/staticcheck.rc
  govulncheck ./... > /saida/govulncheck.txt 2>&1; echo \$? > /saida/govulncheck.rc
" || { echo "falha ao instalar as ferramentas no contÃªiner"; exit 2; }

# Contagem do gosec: achado ABERTO = sem supressÃ£o; suprimido = com #nosec justificado.
cat > "$tmp/gosec.py" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
todos = d.get("Issues", [])
aberto = [i for i in todos if not i.get("suppressions") and not i.get("nosec")]
modo = sys.argv[2]
if modo == "SUP":
    print(len(todos) - len(aberto))
elif modo == "LISTA":
    for i in todos:
        sup = i.get("suppressions") or []
        estado = "ABERTO" if i in aberto else "suprimido: " + "; ".join(x.get("justification", "") for x in sup)
        print("%-6s %s %s:%s [%s]" % (i["severity"], i["rule_id"], i["file"].replace("/src/", ""), i["line"], estado))
else:
    print(sum(1 for i in aberto if i["severity"] == modo))
PY
conta() { python3 "$tmp/gosec.py" "$tmp/gosec.json" "$1"; }
nosec="$(conta SUP)"; alto="$(conta HIGH)"; medio="$(conta MEDIUM)"; baixo="$(conta LOW)"
sc="$(grep -c '(' "$tmp/staticcheck.txt" || true)"; [[ "$(cat "$tmp/staticcheck.rc")" == 0 ]] && sc=0
vc="$(cat "$tmp/govulncheck.rc")"

{
  echo "# AnÃ¡lise estÃ¡tica de seguranÃ§a"
  echo
  echo "Gerado por \`scripts/analise-estatica.sh\` em $(date +%F) (contÃªiner \`$IMG\`; gosec $GOSEC, staticcheck $STATICCHECK, govulncheck $GOVULN)."
  echo
  echo "| Ferramenta | Resultado |"
  echo "|---|---|"
  echo "| gosec | alta: $alto Â· mÃ©dia: $medio Â· baixa: $baixo (achados abertos); suprimidos com \`#nosec\` justificado: $nosec |"
  echo "| staticcheck | $sc achado(s) |"
  echo "| govulncheck | $([[ "$vc" == 0 ]] && echo "nenhuma vulnerabilidade alcanÃ§Ã¡vel pelo cÃ³digo" || echo "VULNERABILIDADE ENCONTRADA (cÃ³digo $vc)") |"
  echo
  echo "## staticcheck"; echo '```'; cat "$tmp/staticcheck.txt"; echo '```'
  echo "## govulncheck"; echo '```'; cat "$tmp/govulncheck.txt"; echo '```'
  echo "## gosec (achados abertos e suprimidos)"; echo '```'; conta LISTA; echo '```'
} > "$saida"

echo "gosec: alta=$alto mÃ©dia=$medio baixa=$baixo (suprimidos=$nosec) Â· staticcheck=$sc Â· govulncheck rc=$vc"
echo "relatÃ³rio: $saida"
[[ "$alto" == 0 && "$medio" == 0 && "$sc" == 0 && "$vc" == 0 ]]
