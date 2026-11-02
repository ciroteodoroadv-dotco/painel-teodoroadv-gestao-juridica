# ADR 0007 — Backup cifrado com age, chave privada fora do servidor

**Status:** aceita (02/11/2026)

**Decisão:** o contêiner de backup gera `painel-AAAAMMDDTHHMMSSZ.tar.age` (banco `pg_dump -Fc` + documentos + manifesto
com SHA-256), cifrado com **age** para uma ou mais chaves **públicas**. O servidor guarda só as públicas. Cópia externa
por `rclone` (Object Storage Oracle e uma segunda cópia fora da Oracle). Retenção 7 diários, 4 semanais, 12 mensais.

**Duas chaves privadas:** (1) **recuperação de desastre**, fora do servidor, em dois lugares offline; (2) **teste de
restauração**, no servidor, usada só pelo teste mensal automático. Trade-off: com (2) no servidor, o teste é automático;
quem roubar só a cópia externa continua sem poder ler. Quem controla o servidor já tem o banco e a `field_key`.

**Backup só vale se restaura:** `restore-test.sh` restaura num banco e pasta temporários, **sem internet**, e roda
`painel restore-check` (cadeia de auditoria + decifrar e conferir o SHA-256 de **todos** os documentos).

**Limites:** `field_key` e `session_key` NÃO vão no backup (segredos); a `field_key` precisa de guarda própria. O
backup do volume de documentos é consistente com o banco porque o dump vem antes do empacotamento (arquivos só crescem).
Backup de banco em 1 instância só; sem replicação.
