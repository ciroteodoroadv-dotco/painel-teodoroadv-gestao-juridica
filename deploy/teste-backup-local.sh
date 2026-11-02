#!/usr/bin/env bash
# Teste ponta a ponta de backup e restauração na pilha de DESENVOLVIMENTO (dados fictícios).
# Pré-requisito: pilha dev rodando (docker compose ... up), com o volume painel_docs.
set -uo pipefail
cd "$(dirname "$0")"
P="${COMPOSE_PROJECT_NAME:-painel}"
IMG_B=painel-backup:dev; IMG_A=painel-app:latest
work="$(mktemp -d)"; chmod 755 "$work"; mkdir -p "$work/bk" "$work/keys"; chmod 777 "$work/bk"
trap 'rm -rf "$work"' EXIT
ok=1
passa() { echo "  ✔ $*"; }
falha() { echo "  ✘ $*"; ok=0; }

echo "== Gerando um par de chaves age descartável (a privada fica FORA do 'servidor')"
docker run --rm --entrypoint age-keygen "$IMG_B" 2> /dev/null > "$work/keys/identity.txt" || true
grep -q AGE-SECRET-KEY "$work/keys/identity.txt" || { docker run --rm --entrypoint age-keygen "$IMG_B" > "$work/keys/identity.txt" 2>/dev/null; }
grep '^# public key:' "$work/keys/identity.txt" | sed 's/^# public key: //' > "$work/keys/recipients.txt"
chmod 644 "$work/keys/"*
docker run --rm --entrypoint age-keygen "$IMG_B" > "$work/keys/outra.txt" 2>/dev/null; chmod 644 "$work/keys/outra.txt"

fazer_backup() {
  docker run --rm --network "${P}_banco" -v "${P}_docs":/data/documentos:ro -v "$work/bk":/backups \
    -v "$work/keys/recipients.txt":/run/age/recipients:ro -v "$PWD/secrets/db_password":/run/secrets/db_password:ro \
    -e PGHOST=db -e PGUSER=painel -e PGDATABASE=painel -e PGPASSWORD_FILE=/run/secrets/db_password \
    -e AGE_RECIPIENTS_FILE=/run/age/recipients -e DOCS_DIR=/data/documentos \
    --entrypoint /usr/local/bin/backup.sh "$IMG_B" 2>&1 "$@"
}

echo "== 1. Backup"
sleep 1
saida="$(fazer_backup)"; echo "$saida" | sed 's/^/    /'
bk="$(ls -1 "$work/bk"/painel-*.tar.age 2>/dev/null | head -1)"
[[ -n "$bk" ]] && passa "arquivo de backup criado: $(basename "$bk")" || { falha "nenhum backup criado"; exit 1; }
[[ "$(stat -c %a "$bk")" == 600 ]] && passa "permissão 0600" || falha "permissão do backup deveria ser 0600"
como65532() { docker run --rm -u 65532 -v "$work/bk":/bk -v "$work/adu":/adu --entrypoint sh alpine -c "$1"; }
mkdir -p "$work/adu"; chmod 777 "$work/adu"
como65532 "head -c 22 /bk/$(basename "$bk")" | grep -q 'age-encryption.org' && passa "é um arquivo age (cifrado)" || falha "cabeçalho age ausente"
if como65532 "strings /bk/$(basename "$bk")" | grep -qiE 'PGDMP|pessoas|Cliente Exemplo|VERIFICACAO'; then falha "texto em claro dentro do backup"; else passa "nenhum texto em claro (banco e documentos cifrados)"; fi
echo "$saida" | grep -q 'AVISO: nenhuma cópia externa' && passa "avisou que não há cópia externa configurada" || falha "deveria avisar sobre a falta de cópia externa"
sleep 1; fazer_backup >/dev/null; sleep 1; fazer_backup >/dev/null
[[ "$(ls "$work/bk"/painel-*.tar.age | wc -l)" == 3 ]] && passa "3 backups guardados (retenção não apaga os recentes)" || falha "contagem de backups inesperada"

echo "== 2. Restauração com a chave certa (banco + documentos + conferência)"
saida="$(BACKUP_DIR="$work/bk" AGE_IDENTITY="$work/keys/identity.txt" FIELD_KEY_FILE="$PWD/secrets/field_key" IMAGE_BACKUP="$IMG_B" IMAGE_APP="$IMG_A" ./restore-test.sh 2>&1)"; rc=$?
echo "$saida" | sed 's/^/    /'
[[ $rc == 0 ]] && echo "$saida" | grep -q 'RESULTADO: cópia restaurada íntegra' && passa "restauração conferida" || falha "restauração falhou (rc=$rc)"
echo "$saida" | grep -qE 'cadeia ÍNTEGRA' && passa "cadeia de auditoria íntegra na cópia" || falha "cadeia de auditoria não confirmada"
echo "$saida" | grep -qE 'documentos: [1-9][0-9]* registrado' && passa "documentos decifrados e conferidos" || falha "nenhum documento conferido"

echo "== 3. Restauração com a chave ERRADA deve falhar"
if BACKUP_DIR="$work/bk" AGE_IDENTITY="$work/keys/outra.txt" FIELD_KEY_FILE="$PWD/secrets/field_key" IMAGE_BACKUP="$IMG_B" IMAGE_APP="$IMG_A" ./restore-test.sh >/dev/null 2>&1; then falha "chave errada restaurou!"; else passa "chave errada recusada"; fi

echo "== 4. Backup adulterado deve ser detectado"
b="$(basename "$bk")"
como65532 "cp /bk/$b /adu/painel-20260101T000000Z.tar.age; cp /bk/$b.sha256 /adu/painel-20260101T000000Z.tar.age.sha256; printf '\\000' | dd of=/adu/painel-20260101T000000Z.tar.age bs=1 seek=200 conv=notrunc 2>/dev/null; sed -i 's/painel-[0-9TZ]*\\.tar\\.age/painel-20260101T000000Z.tar.age/' /adu/painel-20260101T000000Z.tar.age.sha256"
if BACKUP_DIR="$work/adu" AGE_IDENTITY="$work/keys/identity.txt" FIELD_KEY_FILE="$PWD/secrets/field_key" IMAGE_BACKUP="$IMG_B" IMAGE_APP="$IMG_A" ./restore-test.sh >/dev/null 2>&1; then falha "backup adulterado foi aceito!"; else passa "backup adulterado recusado"; fi

echo "== 5. Backup de estado inconsistente (documento ausente no volume) deve reprovar na conferência"
arq_doc="$(docker run --rm -v "${P}_docs":/d alpine sh -c 'ls /d | head -1')"
if [[ -n "$arq_doc" ]]; then
  docker run --rm -v "${P}_docs":/d alpine sh -c "mv /d/$arq_doc /d/$arq_doc.guardado"
  mkdir -p "$work/bk2"; chmod 777 "$work/bk2"
  docker run --rm --network "${P}_banco" -v "${P}_docs":/data/documentos:ro -v "$work/bk2":/backups \
    -v "$work/keys/recipients.txt":/run/age/recipients:ro -v "$PWD/secrets/db_password":/run/secrets/db_password:ro \
    -e PGHOST=db -e PGUSER=painel -e PGDATABASE=painel -e PGPASSWORD_FILE=/run/secrets/db_password \
    -e AGE_RECIPIENTS_FILE=/run/age/recipients -e DOCS_DIR=/data/documentos \
    --entrypoint /usr/local/bin/backup.sh "$IMG_B" >/dev/null 2>&1
  docker run --rm -v "${P}_docs":/d alpine sh -c "mv /d/$arq_doc.guardado /d/$arq_doc"
  saida="$(BACKUP_DIR="$work/bk2" AGE_IDENTITY="$work/keys/identity.txt" FIELD_KEY_FILE="$PWD/secrets/field_key" IMAGE_BACKUP="$IMG_B" IMAGE_APP="$IMG_A" ./restore-test.sh 2>&1)"; rc=$?
  [[ $rc != 0 ]] && echo "$saida" | grep -q 'arquivo ausente' && passa "documento ausente foi apontado e a conferência reprovou" || falha "inconsistência não detectada (rc=$rc)"
else
  echo "  (sem documentos no volume: etapa pulada)"
fi

echo
((ok)) && echo "RESULTADO GERAL: todos os testes de backup/restauração passaram." || { echo "RESULTADO GERAL: HOUVE FALHAS."; exit 1; }
