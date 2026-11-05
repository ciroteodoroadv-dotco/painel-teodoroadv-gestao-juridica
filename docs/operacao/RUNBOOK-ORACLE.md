# Guia do servidor na Oracle (M5) — passo a passo

Linguagem simples, comandos para copiar e colar. Quem executa: o advogado, com apoio técnico quando indicado.

> **O que já foi provado e o que NÃO foi.** Testados em computador local, com dados fictícios: backup cifrado,
> restauração isolada com conferência completa, retenção 7/4/12, modo produção do app (recusa sem Cloudflare Access),
> deploy com rollback automático, Compose de produção. **NÃO testados numa VM Oracle real** (dependem das contas do M0):
> `cloud-init.yaml`, `bootstrap.sh`, firewall `nftables`, timers do `systemd`, túnel Cloudflare, cópia para o Object
> Storage, `release.yml`. Faça a primeira subida com calma e confira cada "Como saber que deu certo".

## 0. Antes de começar (lista do M0)

| Item | Quem | Observação |
|---|---|---|
| Conta Oracle Cloud (região **São Paulo** ou **Vinhedo**) | advogado | dados em repouso no Brasil |
| Domínio do escritório gerenciado no Cloudflare | advogado | ex.: `painel.seuescritorio.com.br` |
| Conta Cloudflare Zero Trust + **DPA** arquivado | advogado | Cloudflare é operador (ADR 0002) |
| Repositório privado no GitHub (remoto do projeto) | advogado | habilita o CI e a entrega por digest |
| Inventário LGPD aprovado (`docs/lgpd/`) | advogado | **sem isso, nada de dado real** |
| Lista de e-mails autorizados a acessar | advogado | entram na política do Access |
| Destino da **segunda cópia** dos backups, fora da Oracle | advogado | ex.: outro provedor ou disco externo; entra no ROPA |

## 1. Oracle Cloud — rede e servidor

1. **Compartimento** próprio ("painel"). Tudo abaixo dentro dele.
2. **Rede (VCN)** com **sub-rede privada** (sem IP público) e **NAT Gateway** (o servidor só precisa *sair* para a
   internet: túnel Cloudflare, atualizações). **Lista de segurança: nenhuma regra de entrada**, exceto SSH vindo da
   sub-rede do Bastion.
3. **Bastion** (serviço Oracle): é por ele que a equipe técnica entra por SSH, com chave, sem expor a porta 22 à internet.
4. **Instância**: Ubuntu 24.04, 2 OCPU / 12 GB (Ampere A1) ou 1 OCPU / 8 GB (AMD), disco de 100 GB. Em
   "Opções avançadas → Cloud-init", cole `deploy/oracle/cloud-init.yaml` trocando `SUA_CHAVE_SSH_PUBLICA`.
   **Como saber que deu certo:** pelo Bastion, `cat /var/lib/painel/cloud-init.ok` mostra a data.
5. **Object Storage**: bucket **privado** `painel-backups`, com **regra de retenção** (ex.: 35 dias) e versionamento.
   Crie um usuário só para backup, com permissão apenas nesse bucket, e gere uma **chave de acesso compatível com S3**
   ("Customer Secret Key"). É isso que o `rclone` usa.

## 2. Cloudflare — túnel e controle de acesso

1. Zero Trust → Networks → **Tunnels** → *Create* → `cloudflared`. **Copie o token** (guarde: aparece uma vez).
2. No túnel, **Public hostname**: `painel.seuescritorio.com.br` → serviço **HTTP** `app:8080`.
3. Zero Trust → Access → **Applications** → *Self-hosted* → o mesmo endereço. Política **Allow** apenas para a
   **lista de e-mails** autorizados. Duração de sessão curta (ex.: 8 h).
4. Anote: o **domínio da equipe** (`algo.cloudflareaccess.com`) e o **Application Audience (AUD) tag**. O app usa os dois
   para validar o JWT: sem eles, em produção, **o app recusa iniciar**.

## 3. Chaves do backup (faça num computador seu, NÃO no servidor)

```bash
age-keygen -o chave-recuperacao.txt      # a privada fica SÓ com você; anote a linha "public key: age1..."
age-keygen -o chave-teste-restauracao.txt # esta privada irá ao servidor, só para o teste mensal automático
```

- **chave-recuperacao.txt**: guarde em **dois lugares offline** (cofre/pendrive). **Não** coloque em pasta sincronizada
  (Drive/OneDrive). Sem ela, backup nenhum se abre num desastre.
- No servidor ficam apenas chaves **públicas** (em `secrets/age_recipients`, uma por linha) e a privada **de teste**.
  Por quê: quem roubar só a cópia externa dos backups não consegue lê-la. (Quem tem acesso ao servidor já tem tudo — a
  chave de teste não piora isso.)

## 4. Primeira subida

Entre pelo Bastion como `painel-admin` e:

```bash
cd /opt/painel && git clone <URL-do-repositório-privado> . && cd deploy
./gerar-segredos.sh                                    # cria db_password, db_app_password, session_key, field_key
printf '%s' 'COLE_O_TOKEN_DO_TUNEL' > secrets/tunnel_token
printf '%s\n%s\n' 'age1...RECUPERACAO' 'age1...TESTE' > secrets/age_recipients
nano secrets/age_restore_identity                      # cole a chave PRIVADA DE TESTE
nano secrets/rclone.conf                               # veja o modelo abaixo
chmod 600 secrets/*; chmod 444 secrets/db_password secrets/db_app_password secrets/session_key secrets/field_key
```

Modelo do `secrets/rclone.conf` (Object Storage Oracle, API compatível com S3):

```ini
[oci]
type = s3
provider = Other
access_key_id = SUA_ACCESS_KEY
secret_access_key = SUA_SECRET_KEY
endpoint = https://NAMESPACE.compat.objectstorage.sa-saopaulo-1.oraclecloud.com
region = sa-saopaulo-1
```

Banco no mesmo servidor (padrão deste guia, serviço `db` do Docker): nada a configurar. Se um dia o banco for **fora** do
servidor, é obrigatório definir `PAINEL_DB_SSLMODE=verify-full` (o painel recusa iniciar com modo fraco).

Arquivo `oracle/ambiente` (variáveis, **sem segredos**):

```bash
PAINEL_ACCESS_TEAM_DOMAIN=algo.cloudflareaccess.com
PAINEL_ACCESS_AUD=COLE_O_AUD
PAINEL_PUBLIC_URL=https://painel.seuescritorio.com.br
RCLONE_DEST=oci:painel-backups/painel
# RCLONE_DEST_2=<segunda cópia, fora da Oracle>
```

Depois:

```bash
sudo ./oracle/bootstrap.sh 10.0.1.0/24       # use a sub-rede do seu Bastion: instala firewall e timers
docker build -q -t painel-backup:latest backup   # imagem de backup (só ferramentas; não compila o painel)
set -a; source oracle/ambiente; set +a
./oracle/deploy.sh ghcr.io/ORG/painel@sha256:DIGEST   # digest vem do resumo da release (GitHub Actions)
```

*Primeira subida sem CI (exceção, só até o repositório remoto existir):* num computador seu,
`docker build -f deploy/Dockerfile -t painel-app:AAAAMMDD .`, depois
`docker save painel-app:AAAAMMDD | gzip | ssh servidor 'gunzip | docker load'` e, no servidor,
`LOCAL=1 ./oracle/deploy.sh painel-app:AAAAMMDD`. Passe para a entrega por digest assim que possível.

Crie o primeiro administrador (senha lida da entrada; ele troca no primeiro acesso):

```bash
read -rs S; echo "$S" | PAINEL_IMAGE=$(cat .estado-deploy.atual) docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm -T migrate user create seu@email.com administrador; unset S
```

## 5. Aceite — confira tudo antes de usar dado real

| Teste | Como | Esperado |
|---|---|---|
| Nenhuma porta aberta | de **fora** (ex.: celular em 4G): `nmap -Pn IP_DA_VM` (a VM nem tem IP público) | nada responde |
| Acesso sem credencial | abrir `https://painel…` em janela anônima | página de login do Cloudflare Access |
| E-mail não autorizado | tentar com e-mail fora da lista | negado pelo Cloudflare |
| Login completo | e-mail autorizado → login do painel → 2º fator | entra; troca de senha provisória |
| Contêineres | `./oracle/status.sh` | app, db e cloudflared "healthy"/"Up" |
| Backup manual | `sudo systemctl start painel-backup` e `./oracle/status.sh` | "último com sucesso: há 0 dia(s)" e arquivo em `/var/backups/painel` |
| Cópia externa | listar o bucket (`rclone ls oci:painel-backups`) | o `.tar.age` e o `.sha256` estão lá |
| **Restauração** | `sudo systemctl start painel-restore-test` e `journalctl -u painel-restore-test -n 30` | "RESULTADO: cópia restaurada íntegra e utilizável." |
| Restauração com a chave offline | num computador seu: `AGE_IDENTITY=chave-recuperacao.txt` (ver §7) | mesmo resultado |
| Desligar o acesso | `docker compose … stop cloudflared` | painel fora do ar para todos; `start` volta |

## 6. Rotina

- **Todo dia (automático):** backup às 03:30 (Brasília). Retenção local: 7 diários + 4 semanais + 12 mensais.
- **Todo mês (automático):** teste de restauração (dia 1). **Se falhar, trate como incidente**: backup que não restaura
  não é backup. Veja `./oracle/status.sh` toda semana (deve mostrar backup recente e restauração sem "ATENÇÃO").
- **Atualizar o painel:** `./oracle/deploy.sh ghcr.io/ORG/painel@sha256:NOVO_DIGEST`. O script faz backup, sobe a nova
  versão, espera ficar saudável e **volta sozinho para a anterior se falhar**. Atenção: o retorno troca só o programa;
  se uma migração do banco já rodou e foi destrutiva, é preciso restaurar o backup feito pelo próprio deploy.
- **Aparelho do 2º fator perdido (usuário):** administrador → Usuários → "Redefinir 2º fator" (ou
  `docker compose ... run --rm migrate user reset-2fa EMAIL`); a pessoa cadastra de novo.

## 7. Desastre: restaurar de um backup

1. Servidor novo pelos passos 1–4 (o banco nasce vazio). **Pare o app**: `docker compose … stop app`.
2. Traga o backup (do bucket ou da segunda cópia) e a **chave de recuperação offline**.
3. **Mesmos segredos de antes** (`field_key` em especial: sem ela os documentos cifrados não abrem; guarde cópia dela,
   cifrada, junto da chave de recuperação — separada dos backups).
4. Restaure com o mesmo mecanismo do teste: `AGE_IDENTITY=chave-recuperacao.txt BACKUP_DIR=… ./restore-test.sh`
   mostra que o backup está íntegro; para **restaurar em produção**, use `backup/restore.sh` apontando para o banco e a
   pasta reais (`painel_docs`), com o app parado. Depois `docker compose … start app` e confira `painel audit verify`.

## 8. Incidente (suspeita de acesso indevido)

1. **Corte o acesso**: `docker compose … stop cloudflared` (o painel sai do ar para todos).
2. Preserve: `docker compose … logs > /var/log/painel/incidente-AAAAMMDD.log`; rode `painel audit verify`.
3. Revogue sessões e troque os segredos (`gerar-segredos.sh` após mover os antigos; **a `field_key` não pode ser
   trocada sem recifrar os dados** — decisão técnica a tomar com cuidado).
4. Siga o plano em `docs/lgpd/03-plano-incidente.md` (comunicação à ANPD e aos titulares, art. 48).
