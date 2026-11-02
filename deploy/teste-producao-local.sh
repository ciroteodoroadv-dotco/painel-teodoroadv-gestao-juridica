#!/usr/bin/env bash
# Teste local do modo PRODUÇÃO e do deploy com rollback, num projeto Compose separado ("painelprod").
# Não mexe na pilha de desenvolvimento. Sem Cloudflare real: o JWT é exigido, então toda requisição de fora é negada.
set -uo pipefail
cd "$(dirname "$0")"
PROJ=painelprod
ok=1
passa() { echo "  ✔ $*"; }
falha() { echo "  ✘ $*"; ok=0; }
export PAINEL_ACCESS_TEAM_DOMAIN=teste.cloudflareaccess.com PAINEL_ACCESS_AUD=aud-de-teste PAINEL_PUBLIC_URL=https://painel.exemplo.test
touch secrets/tunnel_token secrets/rclone.conf secrets/age_recipients
C="docker compose -p $PROJ -f docker-compose.yml -f docker-compose.prod.yml"
limpar() { PAINEL_IMAGE=painel-app:latest $C down -v >/dev/null 2>&1; docker rmi painel-app:good painel-app:bad >/dev/null 2>&1; rm -f ./.estado-teste.*; }
trap limpar EXIT
limpar

# imagens: "good" = a atual; "bad" = mesma imagem, mas com um comando inexistente (morre ao iniciar)
docker tag painel-app:latest painel-app:good
printf 'FROM painel-app:latest\nENTRYPOINT ["/painel","comando-inexistente"]\n' | docker build -q -t painel-app:bad - >/dev/null

echo "== 1. Produção SEM as variáveis do Cloudflare Access deve recusar iniciar"
saida="$(PAINEL_IMAGE=painel-app:good PAINEL_ACCESS_AUD= PAINEL_ACCESS_TEAM_DOMAIN= docker compose -p $PROJ -f docker-compose.yml -f docker-compose.prod.yml run --rm --no-deps -e PAINEL_ACCESS_AUD= -e PAINEL_ACCESS_TEAM_DOMAIN= app 2>&1 | tail -4)"
echo "$saida" | grep -q 'PAINEL_ACCESS_TEAM_DOMAIN e PAINEL_ACCESS_AUD' && passa "recusou iniciar e explicou o motivo" || falha "deveria recusar: $saida"

# o banco já roda em produção quando o deploy acontece; aqui ele precisa ser iniciado primeiro
PAINEL_IMAGE=painel-app:good $C up -d --wait db >/dev/null 2>&1

echo "== 2. Deploy da versão boa (sem versão anterior) — sem backup/pull no teste local"
ESPERA=70 LOCAL=1 SKIP_BACKUP=1 STATE_FILE=./.estado-teste COMPOSE="$C" ./oracle/deploy.sh painel-app:good > /tmp/deploy1.log 2>&1; rc=$?
tail -3 /tmp/deploy1.log | sed 's/^/    /'
[[ $rc == 0 ]] && passa "deploy da versão boa concluído" || falha "deploy da versão boa falhou (rc=$rc)"

echo "== 3. Comportamento do app em produção (de dentro da rede, sem passar pelo Cloudflare)"
NET="${PROJ}_borda"
req() { docker run --rm --network "$NET" curlimages/curl -s -o /dev/null -w "%{http_code}" "$@"; }
[[ "$(req http://app:8080/login)" == 403 ]] && passa "/login sem JWT do Access → 403" || falha "/login sem JWT deveria dar 403"
[[ "$(req http://app:8080/pessoas)" == 403 ]] && passa "/pessoas sem JWT → 403" || falha "/pessoas sem JWT deveria dar 403"
[[ "$(req http://app:8080/static/app.css)" == 403 ]] && passa "até o /static exige o Access" || falha "/static deveria exigir o JWT"
[[ "$(req http://app:8080/healthz)" == 200 ]] && passa "/healthz interno (sem Cf-Ray) → 200" || falha "/healthz interno deveria dar 200"
[[ "$(req -H 'Cf-Ray: abc123' http://app:8080/healthz)" == 403 ]] && passa "/healthz vindo pelo Cloudflare (Cf-Ray) → 403 (não exposto ao mundo)" || falha "/healthz com Cf-Ray deveria dar 403"
[[ "$(req -H 'Cf-Access-Jwt-Assertion: lixo.lixo.lixo' http://app:8080/login)" == 403 ]] && passa "JWT inválido → 403" || falha "JWT inválido deveria dar 403"
publicadas="$(PAINEL_IMAGE=painel-app:good $C ps --format '{{.Ports}}' | grep -c '0.0.0.0\|:::' || true)"
[[ "$publicadas" == 0 ]] && passa "nenhuma porta publicada no host" || falha "há porta publicada no host"
cid="$(PAINEL_IMAGE=painel-app:good $C ps -q app)"
[[ "$(docker inspect -f '{{.HostConfig.ReadonlyRootfs}} {{.Config.User}}' "$cid")" == "true nonroot:nonroot" ]] && passa "app: raiz somente leitura e usuário sem privilégios" || falha "app sem endurecimento esperado"

echo "== 4. Deploy de versão QUEBRADA deve fazer ROLLBACK automático"
ESPERA=35 LOCAL=1 SKIP_BACKUP=1 STATE_FILE=./.estado-teste COMPOSE="$C" ./oracle/deploy.sh painel-app:bad > /tmp/deploy2.log 2>&1; rc=$?
grep -E "FALHA|ROLLBACK|rollback" /tmp/deploy2.log | sed 's/^/    /'
[[ $rc == 1 ]] && passa "deploy reportou falha (exit 1)" || falha "deploy deveria terminar com exit 1 (rc=$rc)"
grep -q 'rollback concluído' /tmp/deploy2.log && passa "rollback concluído" || falha "rollback não concluído"
cid="$(PAINEL_IMAGE=painel-app:good $C ps -q app)"
img="$(docker inspect -f '{{.Config.Image}}' "$cid" 2>/dev/null)"; st="$(docker inspect -f '{{.State.Health.Status}}' "$cid" 2>/dev/null)"
[[ "$img" == "painel-app:good" && "$st" == healthy ]] && passa "a versão boa voltou e está saudável" || falha "após o rollback: imagem=$img saúde=$st"
[[ "$(cat ./.estado-teste.atual)" == "painel-app:good" ]] && passa "estado do deploy continua apontando para a versão boa" || falha "estado do deploy incorreto"

echo "== 5. Deploy por tag móvel (sem digest) deve ser recusado fora do modo local"
ESPERA=5 SKIP_BACKUP=1 STATE_FILE=./.estado-teste COMPOSE="$C" ./oracle/deploy.sh painel-app:latest > /tmp/deploy3.log 2>&1; rc=$?
[[ $rc == 2 ]] && grep -q 'digest' /tmp/deploy3.log && passa "recusou tag sem digest" || falha "deveria recusar tag sem digest (rc=$rc)"

echo
((ok)) && echo "RESULTADO GERAL: produção local e deploy com rollback passaram." || { echo "RESULTADO GERAL: HOUVE FALHAS."; exit 1; }
