#!/usr/bin/env bash
# Painel de saúde em uma tela: o que está rodando, idade do último backup e do último teste de restauração.
# Uso: ./status.sh      (rode como painel-admin, de qualquer pasta)
cd "$(dirname "$0")/.."
C="docker compose -f docker-compose.yml -f docker-compose.prod.yml"
idade() { # arquivo -> "há N dia(s)" ou "nunca"
  [[ -s "$1" ]] || { echo "nunca"; return; }
  local s; s=$(( $(date +%s) - $(date -d "$(cat "$1")" +%s) )); echo "há $((s / 86400)) dia(s) e $(( (s % 86400) / 3600 )) h"
}
echo "== Contêineres"
PAINEL_IMAGE="${PAINEL_IMAGE:-$(cat .estado-deploy.atual 2>/dev/null || echo x)}" $C ps --format 'table {{.Service}}\t{{.Status}}'
echo; echo "== Versão em produção"; cat .estado-deploy.atual 2>/dev/null || echo "(nenhum deploy registrado)"
echo; echo "== Backup"
echo "último com sucesso: $(idade /var/lib/painel/backup.ok)"
[[ -s /var/lib/painel/backup.falha ]] && echo "ATENÇÃO: última falha em $(cat /var/lib/painel/backup.falha)"
echo "arquivos: $(ls /var/backups/painel/painel-*.tar.age 2>/dev/null | wc -l)  ·  espaço: $(du -sh /var/backups/painel 2>/dev/null | cut -f1)"
echo; echo "== Teste de restauração"
echo "último com sucesso: $(idade /var/lib/painel/restore-test.ok)"
[[ -s /var/lib/painel/restore-test.falha ]] && echo "ATENÇÃO: última falha em $(cat /var/lib/painel/restore-test.falha)"
[[ -s /var/lib/painel/alerta.ultimo ]] && echo "Último alerta enviado: $(cat /var/lib/painel/alerta.ultimo) (configure deploy/secrets/alerta_webhook para receber no celular)"
echo; echo "== Disco"; df -h / /var/lib/docker /var/backups/painel 2>/dev/null | sort -u
