#!/usr/bin/env bash
# Restaura um backup em um banco e uma pasta de documentos DE DESTINO (nunca no de produção sem decisão humana).
# Uso: restore.sh ARQUIVO.tar.age
# Variáveis: AGE_IDENTITY_FILE (chave PRIVADA, fornecida só na hora) PGHOST PGUSER PGDATABASE PGPASSWORD(_FILE) DOCS_DIR
source "$(dirname "$0")/lib.sh"

arq="${1:?uso: restore.sh ARQUIVO.tar.age}"
: "${AGE_IDENTITY_FILE:?}" "${PGHOST:?}" "${PGUSER:?}" "${PGDATABASE:?}" "${DOCS_DIR:?}"
[[ -r "$arq" ]] || die "backup não encontrado: $arq"
[[ -r "$AGE_IDENTITY_FILE" ]] || die "chave privada não encontrada"
PGPASSWORD="$(secret PGPASSWORD)"; export PGPASSWORD

if [[ -r "$arq.sha256" ]]; then
  ( cd "$(dirname "$arq")" && sha256sum -c "$(basename "$arq").sha256" >/dev/null ) || die "o arquivo de backup está corrompido (SHA-256 não confere)"
  log "SHA-256 do arquivo confere"
fi

work="$(mktemp -d /work/rs.XXXXXX)"
trap 'rm -rf "$work"' EXIT
umask 077

log "decifrando"
age -d -i "$AGE_IDENTITY_FILE" "$arq" | tar -C "$work" -xf - || die "não consegui decifrar (chave privada errada ou arquivo adulterado)"
[[ -s "$work/db.dump" && -s "$work/MANIFEST" ]] || die "backup incompleto"
# shellcheck disable=SC1091
source <(grep -E '^(sha256_db|sha256_docs|esquema_versao|criado_em)=' "$work/MANIFEST")
[[ "$(sha256sum "$work/db.dump" | cut -d' ' -f1)" == "$sha256_db" ]] || die "db.dump não confere com o manifesto"
[[ "$(sha256sum "$work/docs.tar" | cut -d' ' -f1)" == "$sha256_docs" ]] || die "docs.tar não confere com o manifesto"
log "backup de $criado_em (esquema v$esquema_versao) íntegro"

log "restaurando o banco em $PGHOST/$PGDATABASE"
pg_restore --exit-on-error --no-owner -d "$PGDATABASE" "$work/db.dump" || die "falha ao restaurar o banco"
n="$(psql -Atqc "SELECT count(*) FROM pg_tables WHERE schemaname='public'")"
[[ "$n" -gt 0 ]] || die "o banco restaurado está vazio"

log "restaurando os documentos em $DOCS_DIR"
mkdir -p "$DOCS_DIR"
tar -C "$DOCS_DIR" --no-overwrite-dir -xf "$work/docs.tar"
log "restauração concluída ($n tabelas, $(find "$DOCS_DIR" -type f | wc -l) arquivos)"
