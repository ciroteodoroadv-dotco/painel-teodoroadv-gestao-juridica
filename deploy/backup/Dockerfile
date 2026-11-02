# Imagem de backup/restauração: cliente PostgreSQL 17, age (cifra), rclone (cópia externa) e bash.
# Só ferramentas de linha de comando; sem serviço escutando porta.
FROM alpine:3.21
# uid 65532 = o mesmo usuário do app: só ele lê os documentos (0600).
RUN apk add --no-cache bash coreutils findutils postgresql17-client age rclone tar ca-certificates \
 && adduser -D -u 65532 backup \
 && mkdir -p /backups /work && chown backup /backups /work
COPY backup.sh prune.sh restore.sh lib.sh test-prune.sh /usr/local/bin/
RUN chmod 0755 /usr/local/bin/*.sh
USER backup
WORKDIR /work
ENTRYPOINT ["/bin/bash"]
