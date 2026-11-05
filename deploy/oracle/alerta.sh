#!/usr/bin/env bash
# Alerta ativo de falha de uma rotina do painel (backup, teste de restauração). Chamado pelo systemd (OnFailure=).
# Auditoria 05/11, item 3.3: antes a falha só deixava um arquivo em /var/lib/painel e ninguém era avisado.
# Uso: alerta.sh NOME_DA_UNIDADE
#  - sempre: registra no diário do sistema com prioridade CRÍTICA e grava /var/lib/painel/alerta.ultimo;
#  - se existir deploy/secrets/alerta_webhook (uma URL https, tratada como segredo): envia {"text": "..."} por POST
#    (formato aceito por Slack, Mattermost, Google Chat e Discord via /slack). Nenhum dado de cliente vai na mensagem.
set -uo pipefail
unidade="${1:-desconhecida}"
quando="$(date -u +%FT%TZ)"
host="$(hostname -s 2>/dev/null || echo servidor)"
msg="PAINEL TEODORO: a rotina ${unidade} FALHOU em ${host} às ${quando} (UTC). Veja: journalctl -u ${unidade} e ./oracle/status.sh"
logger -p user.crit -t painel-alerta -- "$msg" 2>/dev/null || echo "$msg" >&2
mkdir -p /var/lib/painel 2>/dev/null && printf '%s %s\n' "$quando" "$unidade" > /var/lib/painel/alerta.ultimo 2>/dev/null
hook_file="$(dirname "$0")/../secrets/alerta_webhook"
if [[ -r "$hook_file" ]]; then
  url="$(tr -d '[:space:]' < "$hook_file")"
  if [[ "$url" == https://* ]]; then
    body="$(printf '{"text":"%s"}' "${msg//\"/\'}")"
    curl -fsS -m 15 --retry 3 -H 'Content-Type: application/json' -d "$body" "$url" >/dev/null \
      || logger -p user.crit -t painel-alerta -- "não consegui enviar o alerta pelo webhook" 2>/dev/null
  else
    logger -p user.crit -t painel-alerta -- "alerta_webhook ignorado: precisa ser uma URL https" 2>/dev/null
  fi
fi
exit 0
