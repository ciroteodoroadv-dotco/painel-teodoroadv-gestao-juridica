# Começar a usar com dados reais (no seu computador)

Linguagem simples. Siga na ordem. Tempo total: cerca de 20 minutos.

> **Honestidade primeiro.** Este modo roda **só no seu computador** (`localhost`): se ele estiver desligado, o Docker
> parado ou o disco falhar, o painel some. Por isso o backup é a parte mais importante. O inventário LGPD (M0) e o
> servidor protegido (Oracle + Cloudflare) **ainda não existem**: você está assumindo esse risco ao usar dado real.
> **O painel NÃO monitora DJEN/DataJud ainda** — os prazos são digitados por pessoas. Continue conferindo no tribunal.

## 1. Zerar a demonstração e criar o seu usuário (uma vez)

No terminal do WSL (Ubuntu):

```bash
cd ~/projetos/painel-teodoro-v2/deploy && ./comecar-uso-real.sh
```

Ele pede confirmação antes de apagar (só apaga os dados de **exemplo**), cria o seu administrador (você digita a senha) e
faz o primeiro backup. Abra `http://localhost:18080` (**localhost**, não 127.0.0.1). No primeiro acesso: troque a senha,
cadastre o segundo fator e **guarde os 8 códigos de recuperação** (aparecem uma vez).

## 2. Agendar o backup diário no Windows (uma vez)

No PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File \\wsl.localhost\Ubuntu\home\user\projetos\painel-teodoro-v2\deploy\agendar-backup-windows.ps1
```

Backups cifrados vão para `D:\OneDrive\Backups-Painel` (o OneDrive os leva para fora do computador; só texto cifrado).
Conferir: arquivo `ultimo-backup.ok` com a data de hoje. Se existir `ultimo-backup.falha`, **o backup falhou**.

## 3. Guardar a chave de recuperação (o passo mais importante)

O arquivo `C:\Users\User\.painel-chaves\recuperacao.txt` é a **única** forma de abrir os backups num computador novo.
- Copie para **dois lugares fora do computador** (pendrive guardado em local seguro, cofre/gerenciador de senhas).
- **Não** coloque em pasta do OneDrive/Drive. Sem ela, os backups são lixo; com ela na mão de outra pessoa, ela lê tudo.
- A `field_key` (em `deploy/secrets/`) também está guardada cifrada ao lado dos backups (`chaves-painel.age`).

## 4. Proteger o computador

- Ative o **BitLocker** (disco cifrado) e use senha forte no Windows; bloqueie a tela ao se afastar.
- Deixe o **Docker Desktop iniciar com o Windows** (Configurações → General → "Start Docker Desktop when you sign in").
- O painel responde só neste computador (`127.0.0.1`); não há acesso de fora.

## 5. Rotina

- **Todo dia:** abra **Hoje** — os **prazos** vencidos ou a vencer em 5 dias aparecem no topo. Prazo digitado por assistente
  fica **A CONFERIR** até um advogado **confirmar**; só então vale.
- **Toda semana:** confira se `ultimo-backup.ok` está recente e teste uma restauração uma vez ao mês:
  `cd deploy && BACKUP_DIR=/mnt/d/OneDrive/Backups-Painel AGE_IDENTITY=/mnt/c/Users/User/.painel-chaves/recuperacao.txt IMAGE_BACKUP=painel-backup:dev ./restore-test.sh`
  (deve terminar com "cópia restaurada íntegra e utilizável").
- **PC novo ou desastre:** reinstale Docker/WSL, copie este projeto, use a chave de recuperação e `backup/restore.sh`
  (ver `RUNBOOK-ORACLE.md` §7); a `field_key` volta de `chaves-painel.age`.

## 6. O que fazer se algo parecer errado

Painel não abre → Docker Desktop está rodando? (`docker ps`). Esqueceu a senha ou perdeu o aparelho do 2º fator → outro
administrador usa "Usuários"; se for o único, rode no WSL `painel user reset-password EMAIL` / `reset-2fa EMAIL`
(ver `RUNBOOK.md`).

## 7. Ligar as integrações (depois que o painel estiver em uso)

Nada que fala com a internet vem ligado. Ligue **uma por vez**, na ordem que preferir, seguindo o `docs/plano-fusao/ACEITE-FINAL.md`: índices do Banco Central e Buscar CEP
(`docs/operacao/INDICES-E-CEP.md`), intimações do DJEN (`COLETA-DJEN.md`) e monitor do DataJud (`MONITOR-DATAJUD.md`). Depois de cada uma, confira o resultado na fonte oficial.
