#!/usr/bin/env bash
# PRIMEIRO USO COM DADOS REAIS (computador local). Rode UMA vez, por você, no terminal do WSL:
#     cd ~/projetos/painel-teodoro-v2/deploy && ./comecar-uso-real.sh
#
# O que faz (e pede confirmação antes do que apaga):
#   1. faz um backup cifrado do estado atual (por segurança);
#   2. APAGA o banco e os documentos de DEMONSTRAÇÃO (pessoas "Exemplo", contas de teste) e recria tudo limpo;
#   3. cria o SEU usuário administrador (você digita a senha; ela não aparece nem é gravada em lugar nenhum);
#   4. faz o primeiro backup do sistema limpo.
# Os segredos (deploy/secrets) e as chaves de backup são MANTIDOS.
set -euo pipefail
cd "$(dirname "$0")"
C="docker compose -f docker-compose.yml -f docker-compose.dev.yml"
export PAINEL_DEV_PORT="${PAINEL_DEV_PORT:-18080}"

echo "== 1/4 Backup de segurança do estado atual"
./backup-local.sh --forcar

echo
echo "== 2/4 Zerar a demonstração"
echo "Vou APAGAR o banco e os documentos atuais (são só dados de exemplo)."
read -rp "Digite APAGAR para confirmar (qualquer outra coisa cancela): " ok
[[ "$ok" == "APAGAR" ]] || { echo "Cancelado. Nada foi apagado."; exit 1; }
$C down -v
$C up -d
echo "Aguardando o painel ficar pronto..."
for _ in $(seq 1 60); do
  [[ "$(docker inspect -f '{{.State.Health.Status}}' painel-app-1 2>/dev/null)" == healthy ]] && break; sleep 3
done
curl -fsS "http://localhost:$PAINEL_DEV_PORT/healthz" >/dev/null && echo "painel no ar"

echo
echo "== 3/4 Seu usuário administrador"
read -rp "Seu e-mail de acesso: " email
while true; do
  read -rsp "Senha (12+ caracteres; use uma frase longa): " s1; echo
  read -rsp "Repita a senha: " s2; echo
  [[ "$s1" == "$s2" ]] && break; echo "As senhas não conferem. Tente de novo."
done
printf '%s\n' "$s1" | $C run --rm -T migrate user create "$email" administrador
unset s1 s2

echo
echo "== 4/4 Primeiro backup do sistema limpo"
./backup-local.sh --forcar

cat <<MSG

PRONTO.  Abra http://localhost:$PAINEL_DEV_PORT  (use "localhost", não 127.0.0.1)
 - Entre com o e-mail e a senha provisória; o painel pede para você trocar a senha e cadastrar o segundo fator.
 - GUARDE os códigos de recuperação que aparecerem (uma única vez).
 - Falta ainda: agendar o backup diário (deploy/agendar-backup-windows.ps1) e guardar a chave de recuperação
   (veja docs/operacao/COMECAR-USO-REAL.md).
MSG
