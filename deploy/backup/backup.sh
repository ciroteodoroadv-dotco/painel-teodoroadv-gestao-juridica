#!/usr/bin/env bash
# Backup completo: banco (pg_dump) + documentos (tar) + manifesto, cifrados com age.
# O servidor guarda SÓ a chave pública (AGE_RECIPIENTS_FILE). Quem restaura usa a chave privada, que fica fora dele.
#
# Variáveis: PGHOST PGUSER PGDATABASE PGPASSWORD(_FILE) AGE_RECIPIENTS_FILE DOCS_DIR BACKUP_DIR
#            RCLONE_DEST, RCLONE_DEST_2 (opcionais: cópia fora do servidor/da Oracle)
source "$(dirname "$0")/lib.sh"

: "${PGHOST:?}" "${PGUSER:?}" "${PGDATABASE:?}" "${AGE_RECIPIENTS_FILE:?}"
DOCS_DIR="${DOCS_DIR:-/data/documentos}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"
[[ -s "$AGE_RECIPIENTS_FILE" ]] || die "AGE_RECIPIENTS_FILE vazio: sem chave pública não há backup cifrado"
grep -q '^age1' "$AGE_RECIPIENTS_FILE" || die "AGE_RECIPIENTS_FILE não contém chave pública age (age1...)"
PGPASSWORD="$(secret PGPASSWORD)"; export PGPASSWORD

# O dump (texto do banco) fica no tmpfs /work (memória, nunca em disco em claro). O tar dos documentos — que já estão
# cifrados pelo app, com nome aleatório — vai para uma pasta de trabalho em DISCO, ao lado dos backups: assim o backup
# não falha quando os documentos passarem do tamanho do tmpfs (auditoria 05/11, item 4.1).
# Sobras de uma execução interrompida (ex.: falta de energia) há mais de 12 h.
find "$BACKUP_DIR" -maxdepth 1 -type d -name '.trabalho.*' -mmin +720 -exec rm -rf {} + 2>/dev/null || true
work="$(mktemp -d /work/bk.XXXXXX)"
disco="$(mktemp -d "$BACKUP_DIR/.trabalho.XXXXXX")"
trap 'rm -rf "$work" "$disco"' EXIT
umask 077

log "1/5 despejando o banco"
pg_dump -Fc -Z 6 -f "$work/db.dump"
[[ -s "$work/db.dump" ]] || die "dump vazio"

# O tar dos documentos vem DEPOIS do dump: arquivos só crescem, então tudo que o banco cita está presente.
log "2/5 empacotando os documentos"
if [[ -d "$DOCS_DIR" ]]; then
  # só arquivos (sem a entrada "."): a restauração não mexe nas permissões da pasta de destino
  find "$DOCS_DIR" -mindepth 1 -type f ! -name '*.tmp' -printf '%P\n' | tar -C "$DOCS_DIR" -cf "$disco/docs.tar" -T -
else
  tar -cf "$disco/docs.tar" -T /dev/null
fi

log "3/5 manifesto"
migr="$(psql -Atqc 'SELECT coalesce(max(version_id),0) FROM goose_db_version WHERE is_applied' 2>/dev/null || echo ?)"
{
  echo "criado_em=$(date -u +%FT%TZ)"
  echo "esquema_versao=$migr"
  echo "arquivos_documentos=$(tar -tf "$disco/docs.tar" | grep -vc '/$' || true)"
  echo "sha256_db=$(sha256sum "$work/db.dump" | cut -d' ' -f1)"
  echo "sha256_docs=$(sha256sum "$disco/docs.tar" | cut -d' ' -f1)"
} > "$work/MANIFEST"

log "4/5 cifrando com age"
name="$(backup_name)"
out="$BACKUP_DIR/$name"
# Mesmos nomes e mesma ordem de antes (db.dump, docs.tar, MANIFEST): restore.sh não muda.
tar -cf - -C "$work" db.dump -C "$disco" docs.tar -C "$work" MANIFEST | age -R "$AGE_RECIPIENTS_FILE" -o "$out.partial"
mv "$out.partial" "$out"
chmod 0600 "$out" 2>/dev/null || log "AVISO: o sistema de arquivos de destino não aceita chmod (ex.: pasta do Windows); a proteção é a cifra"
( cd "$BACKUP_DIR" && sha256sum "$name" > "$name.sha256" )
log "gerado: $name ($(du -h "$out" | cut -f1))"

log "5/5 retenção local e cópia externa"
"$(dirname "$0")/prune.sh" "$BACKUP_DIR" || log "AVISO: a retenção local falhou"
rc=0
for dest in "${RCLONE_DEST:-}" "${RCLONE_DEST_2:-}"; do
  [[ -z "$dest" ]] && continue
  if rclone copyto "$out" "$dest/$name" --quiet && rclone copyto "$out.sha256" "$dest/$name.sha256" --quiet; then
    log "cópia externa ok"
  else
    log "ERRO: falha na cópia externa (o backup local foi mantido)"; rc=1
  fi
done
[[ -z "${RCLONE_DEST:-}" ]] && log "AVISO: nenhuma cópia externa configurada (RCLONE_DEST): o backup só existe neste servidor"
exit $rc
