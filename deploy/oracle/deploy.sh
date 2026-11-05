#!/usr/bin/env bash
# Deploy com backup antes, verificação de saúde e ROLLBACK automático.
# Uso:  ./deploy.sh IMAGEM      ex.: ./deploy.sh ghcr.io/ORG/painel@sha256:abc...
#
# Variáveis (opcionais): COMPOSE  ESPERA (segundos, padrão 90)  SKIP_BACKUP=1  LOCAL=1 (não baixa a imagem)  STATE_FILE
# ATENÇÃO: o rollback troca só a IMAGEM. Migrações de banco só andam para frente; por isso toda migração nova
# deve ser compatível com a versão anterior do app (ou o rollback exige restaurar o backup feito aqui).
set -euo pipefail
cd "$(dirname "$0")/.."

COMPOSE="${COMPOSE:-docker compose -f docker-compose.yml -f docker-compose.prod.yml}"
STATE="${STATE_FILE:-./.estado-deploy}"
ESPERA="${ESPERA:-90}"
NOVA="${1:?uso: deploy.sh IMAGEM (preferir imagem por digest: registro/painel@sha256:...)}"
ATUAL="$(cat "$STATE.atual" 2>/dev/null || true)"

log() { printf '%s [deploy] %s\n' "$(date -u +%FT%TZ)" "$*"; }

if [[ "${LOCAL:-0}" != 1 ]]; then
  [[ "$NOVA" == *@sha256:* ]] || { log "ERRO: use a imagem por digest (…@sha256:…), não por tag móvel"; exit 2; }
  log "baixando $NOVA"
  docker pull -q "$NOVA" >/dev/null
fi

# Imagem de backup por digest (auditoria 05/11, item 5.1). Opcional: PAINEL_BACKUP_IMAGE=registro/painel-backup@sha256:...
# Gravada em $STATE.backup e usada pelo timer de backup e pelo teste de restauração. Sem ela, vale a última gravada.
if [[ -n "${PAINEL_BACKUP_IMAGE:-}" ]]; then
  if [[ "${LOCAL:-0}" != 1 ]]; then
    [[ "$PAINEL_BACKUP_IMAGE" == *@sha256:* ]] || { log "ERRO: PAINEL_BACKUP_IMAGE precisa ser por digest (…@sha256:…)"; exit 2; }
    log "baixando $PAINEL_BACKUP_IMAGE"
    docker pull -q "$PAINEL_BACKUP_IMAGE" >/dev/null
  fi
  echo "$PAINEL_BACKUP_IMAGE" > "$STATE.backup"
fi
PAINEL_BACKUP_IMAGE="$(cat "$STATE.backup" 2>/dev/null || echo painel-backup:latest)"
export PAINEL_BACKUP_IMAGE

if [[ "${SKIP_BACKUP:-0}" != 1 ]]; then
  log "backup antes do deploy"
  PAINEL_IMAGE="${ATUAL:-$NOVA}" $COMPOSE --profile backup run --rm backup || { log "ERRO: o backup falhou; deploy cancelado"; exit 1; }
fi

# Espera o contêiner do app ficar saudável (healthcheck do Compose). Devolve 0 se saudável, 1 se não.
esperar() {
  local limite=$((SECONDS + ESPERA)) cid st
  while ((SECONDS < limite)); do
    cid="$(PAINEL_IMAGE="$1" $COMPOSE ps -q app 2>/dev/null || true)"
    if [[ -n "$cid" ]]; then
      st="$(docker inspect -f '{{.State.Status}}/{{if .State.Health}}{{.State.Health.Status}}{{end}}' "$cid" 2>/dev/null || true)"
      [[ "$st" == "running/healthy" ]] && return 0
      [[ "$st" == "running/unhealthy" ]] && return 1
    fi
    sleep 3
  done
  return 1
}

subir() { PAINEL_IMAGE="$1" $COMPOSE up -d --no-build --no-deps app; }

# Migração pelo serviço migrate (único com a senha do dono do banco; auditoria 05/11, item 2.1).
log "migrando o banco com a nova versão"
PAINEL_IMAGE="$NOVA" $COMPOSE run --rm --no-deps migrate || { log "ERRO: a migração falhou; deploy cancelado (a versão em uso continua no ar)"; exit 1; }

log "subindo a nova versão"
subir "$NOVA"
if esperar "$NOVA"; then
  echo "$ATUAL" > "$STATE.anterior"; echo "$NOVA" > "$STATE.atual"
  log "OK: nova versão saudável. Anterior guardada em $STATE.anterior"
  exit 0
fi

log "FALHA: a nova versão não ficou saudável em ${ESPERA}s"
PAINEL_IMAGE="$NOVA" $COMPOSE logs --tail 15 app 2>&1 | sed 's/^/    /' || true
if [[ -n "$ATUAL" ]]; then
  log "ROLLBACK para a versão anterior"
  subir "$ATUAL"
  if esperar "$ATUAL"; then log "rollback concluído: a versão anterior voltou a funcionar"; else log "ERRO GRAVE: a versão anterior também não ficou saudável — siga o plano de incidente"; fi
else
  log "não há versão anterior registrada para voltar"
fi
exit 1
