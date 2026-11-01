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

Isso cria 3 arquivos em `deploy/secrets/`. Eles **nunca** vão para o Git. **Faça cópia** dos três em local
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
| Porta 8080 ocupada | Feche o outro programa que usa a 8080. |

## Rodar os testes (para quem desenvolve)

```bash
cd ~/projetos/painel-teodoro-v2
gofmt -l .            # não deve listar nada
go vet ./...
go test ./...
./scripts/check-lines.sh
```

## O que ainda NÃO existe (marcos seguintes)

Login, 2FA, papéis, auditoria (M2); cadastros (M3); documentos (M4); túnel Cloudflare, backup e servidor
Oracle (M5). O backup ainda não existe: **não coloque dado real** até o M5.
