#!/usr/bin/env bash
# Backup do painel rodando NO COMPUTADOR (localhost). Chamado todo dia pelo Agendador de Tarefas do Windows.
# Gera backup cifrado (age) em BACKUP_DIR (padrão: pasta do OneDrive, que o leva para fora do PC — só vai texto cifrado)
# e uma cópia cifrada da field_key. Pula se já existe backup com menos de MIN_HORAS horas.
# Uso manual:   ./backup-local.sh [--forcar]
set -uo pipefail
cd "$(dirname "$0")"

BACKUP_DIR="${BACKUP_DIR:-/mnt/d/OneDrive/Backups-Painel}"
MIN_HORAS="${MIN_HORAS:-12}"
PROJ="${COMPOSE_PROJECT_NAME:-painel}"
IMG="${IMAGE_BACKUP:-painel-backup:dev}"
LOG="$BACKUP_DIR/backup.log"
log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" | tee -a "$LOG" >&2; }
mkdir -p "$BACKUP_DIR"; chmod 777 "$BACKUP_DIR" 2>/dev/null || true

[[ -s secrets/age_recipients ]] || { log "ERRO: secrets/age_recipients ausente (chave pública do backup)"; exit 2; }

# Recente demais? (a menos que --forcar)
if [[ "${1:-}" != "--forcar" && -s "$BACKUP_DIR/ultimo-backup.ok" ]]; then
  idade=$(( $(date +%s) - $(date -d "$(cat "$BACKUP_DIR/ultimo-backup.ok")" +%s 2>/dev/null || echo 0) ))
  if (( idade < MIN_HORAS * 3600 )); then log "backup recente ($((idade / 3600)) h): nada a fazer"; exit 0; fi
fi

# O Docker Desktop pode estar subindo: espera até 3 minutos.
for _ in $(seq 1 36); do docker info >/dev/null 2>&1 && break; sleep 5; done
docker info >/dev/null 2>&1 || { log "ERRO: Docker indisponível; backup NÃO feito"; date -u +%FT%TZ > "$BACKUP_DIR/ultimo-backup.falha"; exit 1; }
docker ps --format '{{.Names}}' | grep -q "^${PROJ}-db-1$" || { log "ERRO: o banco ($PROJ-db-1) não está rodando; backup NÃO feito"; date -u +%FT%TZ > "$BACKUP_DIR/ultimo-backup.falha"; exit 1; }

log "iniciando backup"
if docker run --rm --network "${PROJ}_banco" -v "${PROJ}_docs":/data/documentos:ro -v "$BACKUP_DIR":/backups \
     -v "$PWD/secrets/age_recipients":/run/age/recipients:ro -v "$PWD/secrets/db_password":/run/secrets/db_password:ro \
     -e PGHOST=db -e PGUSER=painel -e PGDATABASE=painel -e PGPASSWORD_FILE=/run/secrets/db_password \
     -e AGE_RECIPIENTS_FILE=/run/age/recipients -e DOCS_DIR=/data/documentos \
     --entrypoint /usr/local/bin/backup.sh "$IMG" 2>&1 | sed 's/^/  /' | tee -a "$LOG" >&2 ; [[ ${PIPESTATUS[0]} -eq 0 ]]; then
  # field_key cifrada junto: sem ela, os documentos e CPFs cifrados do backup não abrem num computador novo.
  docker run --rm -v "$PWD/secrets/field_key":/k:ro -v "$PWD/secrets/age_recipients":/r:ro -v "$BACKUP_DIR":/out --entrypoint sh "$IMG" \
    -c 'age -R /r -o /out/chaves-painel.tar.age.partial /k && mv /out/chaves-painel.tar.age.partial /out/chaves-painel.age' 2>>"$LOG" \
    || log "AVISO: não consegui gravar a cópia cifrada da field_key"
  date -u +%FT%TZ > "$BACKUP_DIR/ultimo-backup.ok"; rm -f "$BACKUP_DIR/ultimo-backup.falha"
  log "backup concluído"
else
  date -u +%FT%TZ > "$BACKUP_DIR/ultimo-backup.falha"; log "ERRO: o backup falhou"; exit 1
fi
