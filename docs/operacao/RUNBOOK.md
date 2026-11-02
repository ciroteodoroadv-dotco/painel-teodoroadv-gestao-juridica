# Guia de operação — Fase 1 (fundação)

Para quem não é de informática. Copie e cole os comandos, um de cada vez.
Nesta fase **não há dados reais**: o painel só liga, mostra que está vivo e fala com o banco.

## O que existe

- `app`: o programa do painel. Responde "estou vivo" e "o banco responde".
- `db`: o banco de dados PostgreSQL.
- Nenhuma porta fica aberta para a internet. No seu computador, a porta só abre para você mesmo (`127.0.0.1`).

## 1. Criar as senhas (uma vez só)

```bash
cd ~/projetos/painel-teodoro-v2/deploy
./gerar-segredos.sh
```

Isso cria 4 arquivos em `deploy/secrets/`. Eles **nunca** vão para o Git. **Faça cópia** dos três em local
seguro (fora do computador). Sem `field_key`, dados cifrados no futuro não poderão ser abertos.

## 2. Ligar no seu computador

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d
```

## 3. Conferir

```bash
curl http://127.0.0.1:8080/healthz   # deve responder: ok
curl http://127.0.0.1:8080/readyz    # deve responder: ok (banco funcionando)
docker compose ps                     # app e db devem estar "healthy"
```

## 4. Ver o que o painel está registrando

```bash
docker compose logs app
```

Cada linha é uma requisição: id, método, caminho, status e tempo. **Nunca** aparecem nomes, CPF, e-mail,
senha ou o que foi digitado em buscas.

## 5. Desligar

```bash
docker compose down        # desliga, mantém o banco
docker compose down -v     # desliga E APAGA o banco (só em teste!)
```

## Se algo der errado

| Sintoma | O que fazer |
|---|---|
| `app` não sobe e o log diz "segredo ausente" | Rode o passo 1. O painel recusa iniciar sem segredos, de propósito. |
| `/readyz` responde `indisponível` | O banco não está pronto. Espere 30 s; veja `docker compose logs db`. |
| Porta 8080 ocupada (`port is already allocated`) | Use outra porta: `export PAINEL_DEV_PORT=18080` antes do passo 2 e troque `8080` por `18080` nos `curl` do passo 3. |
| `stat ...ubuntu.sock: no such file` | Docker Desktop → Settings → Resources → WSL integration → ligar "Ubuntu". |

## Rodar os testes (para quem desenvolve)

```bash
cd ~/projetos/painel-teodoro-v2
gofmt -l .            # não deve listar nada
go vet ./...
go test ./...
./scripts/check-lines.sh
```

## Usuários e acesso (M2)

Não existe cadastro de usuário pela tela ainda: o administrador cria pelo terminal. A senha é lida da
entrada, nunca vai para o histórico nem para o log. Papéis: `administrador`, `advogado`, `assistente`.
Senha: 12+ caracteres, sem ser comum, sem conter o e-mail.

```bash
cd ~/projetos/painel-teodoro-v2/deploy
C="docker compose -f docker-compose.yml -f docker-compose.dev.yml"
read -rs SENHA; echo "$SENHA" | $C run --rm -T app user create fulano@exemplo.com advogado; unset SENHA
```

No primeiro acesso a pessoa entra com e-mail e senha e cadastra o aplicativo autenticador (QR code).
**Guarde os 8 códigos de recuperação** que aparecem uma única vez.

| Situação | Comando |
|---|---|
| Esqueceu a senha | `read -rs S; echo "$S" \| $C run --rm -T app user reset-password EMAIL; unset S` |
| Conta bloqueada por tentativas | `$C run --rm app user unlock EMAIL` |
| Conferir se a auditoria foi adulterada | `$C run --rm app audit verify` (deve dizer "Cadeia íntegra") |

Bloqueio por senha ou código errado: 5 erros = 1 min, 10 = 5 min, 15 = 30 min, 20 = só o administrador libera.
O administrador vê os últimos eventos em **Auditoria** (menu superior).

**Modo dev:** o `docker-compose.dev.yml` liga `PAINEL_ENV=dev` e dispensa o Cloudflare Access. Em servidor
(`PAINEL_ENV=prod`, o padrão) o app **recusa iniciar** sem `PAINEL_ACCESS_TEAM_DOMAIN` e `PAINEL_ACCESS_AUD`.

## Rodar os testes de integração (banco de teste descartável)

```bash
docker run -d --name painel-teste-db -e POSTGRES_PASSWORD=teste -e POSTGRES_DB=teste -p 127.0.0.1:15432:5432 postgres:17-alpine
export PAINEL_TEST_DB="postgres://postgres:teste@127.0.0.1:15432/teste?sslmode=disable"
go test -count=1 -p 1 ./...      # -p 1 é obrigatório: os testes recriam o banco
docker rm -f painel-teste-db
```

Sem `PAINEL_TEST_DB`, os testes de integração são pulados (aparecem como "skipped").

## Backup e restauração (testar no seu computador)

```bash
cd ~/projetos/painel-teodoro-v2/deploy
docker build -q -t painel-backup:dev backup
./teste-backup-local.sh        # backup + restauração + chave errada + backup adulterado + documento ausente
./teste-producao-local.sh      # modo produção (Cloudflare Access exigido) + deploy com rollback
```

Ambos usam só os dados fictícios da pilha de desenvolvimento e terminam com "todos os testes ... passaram".
O guia do servidor real está em `RUNBOOK-ORACLE.md`.

## Conhecer o sistema com dados de mentirinha (só no seu computador)

```bash
cd ~/projetos/painel-teodoro-v2/deploy
C="docker compose -f docker-compose.yml -f docker-compose.dev.yml"
$C run --rm -T app seed-demo EMAIL_DE_UM_USUARIO   # cria 4 pessoas, 2 processos e 4 tarefas "Exemplo"
```

O comando **recusa rodar em produção**. Só funciona em banco sem pessoas cadastradas.

**Chave de segurança (passkey) no seu computador:** o navegador só aceita chave física em endereço de domínio
ou `localhost` — por isso, para testar a chave, abra `http://localhost:18080` (e não `http://127.0.0.1:18080`).

## Documentos enviados

Ficam no volume `painel_docs` do Docker, **cifrados** (quem copiar a pasta não lê nada). A chave de cifra deriva
do arquivo `deploy/secrets/field_key`: **sem ele, os documentos não abrem**. O backup do volume e da `field_key`
(juntos, mas em locais separados) é parte obrigatória do M5. Aceitos: PDF, PNG, JPEG, DOCX até 25 MiB.

## Telas

Hoje (suas tarefas e a agenda da semana) · Pessoas · Processos · Tarefas · Agenda · Documentos · Minha conta (senha,
chaves, códigos) · Usuários e Auditoria (administrador). Pessoas e processos só são **ocultados** ao excluir; nada é apagado.

## O que ainda NÃO existe (marcos seguintes)

Túnel Cloudflare, backup e servidor Oracle (M5); jobs e
integrações (M6). Faltam também a importação reversível de cadastros e o expurgo definitivo (ver plano).
O backup ainda não existe: **não coloque dado real** até o M5.
