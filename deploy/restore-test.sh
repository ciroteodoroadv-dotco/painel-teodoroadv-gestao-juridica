#!/usr/bin/env bash
# Teste de restauração: restaura um backup num banco e numa pasta TEMPORÁRIOS, isolados da internet e da produção,
# e confere tudo com `painel restore-check` (auditoria íntegra + todos os documentos decifrados).
# Backup que "existe" mas não restaura não vale nada: este teste é a prova.
#
# Uso:   AGE_IDENTITY=/caminho/chave-privada.txt ./restore-test.sh [ARQUIVO.tar.age]
# Opcionais: BACKUP_DIR (padrão /var/backups/painel)  FIELD_KEY_FILE (padrão ./secrets/field_key)
#            IMAGE_APP (padrão painel-app:latest)  IMAGE_BACKUP (padrão painel-backup:latest)
set -euo pipefail
cd "$(dirname "$0")"

BACKUP_DIR="${BACKUP_DIR:-/var/backups/painel}"
FIELD_KEY_FILE="${FIELD_KEY_FILE:-./secrets/field_key}"
IMAGE_APP="${IMAGE_APP:-$(cat .estado-deploy.atual 2>/dev/null || echo painel-app:latest)}" # em produção: a versão em uso
IMAGE_BACKUP="${IMAGE_BACKUP:-painel-backup:latest}"
: "${AGE_IDENTITY:?defina AGE_IDENTITY com o caminho da chave PRIVADA (ela não fica no servidor)}"
[[ -r "$AGE_IDENTITY" ]] || { echo "chave privada não encontrada: $AGE_IDENTITY"; exit 2; }
[[ -r "$FIELD_KEY_FILE" ]] || { echo "field_key não encontrada: $FIELD_KEY_FILE"; exit 2; }

arq="${1:-$(ls -1 "$BACKUP_DIR"/painel-*.tar.age 2>/dev/null | sort | tail -1)}"
# Os backups pertencem ao usuário 65532 (0600): quem lê é o contêiner, não o shell do operador.
[[ -n "$arq" && -e "$arq" ]] || { echo "nenhum backup encontrado em $BACKUP_DIR"; exit 2; }
arq="$(readlink -f "$arq")"
echo "== Testando a restauração de: $(basename "$arq")"

id="$$-$(date +%s)"
net="painel-restore-$id"; dbc="painel-restore-db-$id"
tmp="$(mktemp -d)"; chmod 755 "$tmp"; mkdir -p "$tmp/docs"; chmod 777 "$tmp/docs"
dbpass="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')"
apppass="$(head -c 24 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')"
sesskey="$(head -c 32 /dev/urandom | base64)"
t0=$(date +%s)

cleanup() { docker rm -f "$dbc" >/dev/null 2>&1 || true; docker network rm "$net" >/dev/null 2>&1 || true; rm -rf "$tmp"; }
trap cleanup EXIT

docker network create --internal "$net" >/dev/null   # sem saída para a internet: nada vaza
docker run -d --name "$dbc" --network "$net" -e POSTGRES_USER=painel -e POSTGRES_DB=painel \
  -e POSTGRES_PASSWORD="$dbpass" postgres:17-alpine >/dev/null
for _ in $(seq 1 60); do docker exec "$dbc" pg_isready -U painel -d painel -h 127.0.0.1 >/dev/null 2>&1 && break; sleep 1; done
sleep 2
docker exec "$dbc" psql -U painel -d painel -qc "CREATE ROLE painel_app LOGIN" >/dev/null  # papel do cluster, fora do dump

echo "== 1/2 Restaurando (banco + documentos)"
docker run --rm --network "$net" -v "$(dirname "$arq")":/backups:ro -v "$AGE_IDENTITY":/run/age/identity:ro \
  -v "$tmp/docs":/restored/docs -e AGE_IDENTITY_FILE=/run/age/identity -e PGHOST="$dbc" -e PGUSER=painel \
  -e PGDATABASE=painel -e PGPASSWORD="$dbpass" -e DOCS_DIR=/restored/docs \
  --entrypoint /usr/local/bin/restore.sh "$IMAGE_BACKUP" "/backups/$(basename "$arq")"

# Comparação opcional com o original (Plano 8): REFERENCIA é um arquivo com linhas "T tabela N" (contagens) e "F sha256 arquivo"
# (hash de cada documento CIFRADO do volume). A cópia restaurada precisa voltar com as mesmas contagens e os mesmos hashes.
if [[ -n "${REFERENCIA:-}" ]]; then
  echo "== Comparando contagens e hashes com o original"
  atual="$tmp/atual.txt"; : > "$atual"
  while read -r tipo nome _; do
    [[ "$tipo" == T ]] || continue
    echo "T $nome $(docker exec "$dbc" psql -U painel -d painel -Atc "SELECT count(*) FROM $nome")" >> "$atual"
  done < "$REFERENCIA"
  docker run --rm -v "$tmp/docs":/d:ro alpine sh -c 'cd /d && sha256sum * 2>/dev/null' | awk '{print "F",$1,$2}' >> "$atual"
  if diff <(sort "$REFERENCIA") <(sort "$atual") >/dev/null; then
    echo "comparação: contagens e hashes dos documentos IDÊNTICOS ao original ($(grep -c '^T' "$REFERENCIA") tabelas, $(grep -c '^F' "$REFERENCIA") documento(s))"
  else
    echo "FALHA: a cópia restaurada difere do original:"; diff <(sort "$REFERENCIA") <(sort "$atual") || true; exit 1
  fi
fi

echo "== 2/2 Conferindo a cópia restaurada (somente leitura, sem internet)"
docker run --rm --network "$net" -v "$tmp/docs":/restored/docs:ro -v "$(readlink -f "$FIELD_KEY_FILE")":/run/secrets/field_key:ro \
  -e PAINEL_ENV=dev -e PAINEL_DB_HOST="$dbc" -e PAINEL_DB_NAME=painel -e PAINEL_DB_USER=painel \
  -e PAINEL_DB_PASSWORD="$dbpass" -e PAINEL_DB_APP_PASSWORD="$apppass" -e PAINEL_SESSION_KEY="$sesskey" \
  -e PAINEL_FIELD_KEY_FILE=/run/secrets/field_key -e PAINEL_DOCS_DIR=/restored/docs \
  "$IMAGE_APP" restore-check

echo "== Concluído em $(( $(date +%s) - t0 )) s. Nenhum dado de produção foi tocado."
