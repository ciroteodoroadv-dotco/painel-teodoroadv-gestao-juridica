#!/usr/bin/env bash
# Segunda etapa, rodada UMA vez na VM como painel-admin (depois do cloud-init): instala o firewall,
# os timers de backup/teste de restauração e confere os pré-requisitos. Idempotente.
# Uso: sudo ./bootstrap.sh SUBREDE_ADMIN     ex.: sudo ./bootstrap.sh 10.0.1.0/24
set -euo pipefail
cd "$(dirname "$0")"
[[ $EUID -eq 0 ]] || { echo "rode com sudo"; exit 2; }
ADMIN_CIDR="${1:?informe a sub-rede do OCI Bastion/administração (ex.: 10.0.1.0/24)}"
[[ "$ADMIN_CIDR" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+$ ]] || { echo "sub-rede inválida: $ADMIN_CIDR"; exit 2; }

echo "== Conferindo pré-requisitos"
command -v docker >/dev/null || { echo "docker não instalado (o cloud-init terminou?)"; exit 1; }
v="$(docker compose version --short 2>/dev/null || echo 0)"
printf '%s\n2.24.0\n' "$v" | sort -V -C -r || { echo "docker compose $v é antigo: preciso de 2.24 ou mais novo"; exit 1; }
echo "docker compose $v ok"

echo "== Firewall (nftables)"
sed "s#^define ADMIN_CIDR = .*#define ADMIN_CIDR = $ADMIN_CIDR#" firewall.nft > /etc/nftables.painel.nft
nft -c -f /etc/nftables.painel.nft
grep -q 'nftables.painel.nft' /etc/nftables.conf || echo 'include "/etc/nftables.painel.nft"' >> /etc/nftables.conf
nft -f /etc/nftables.painel.nft
echo "regras ativas:"; nft list table inet painel_fw | sed 's/^/  /'

echo "== Timers de backup e de teste de restauração"
install -m 0644 painel-backup.service painel-backup.timer painel-restore-test.service painel-restore-test.timer painel-alerta@.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now painel-backup.timer painel-restore-test.timer
systemctl list-timers 'painel-*' --no-pager

echo "== Pronto. Próximo passo: RUNBOOK-ORACLE.md, seção 'Primeira subida'."
