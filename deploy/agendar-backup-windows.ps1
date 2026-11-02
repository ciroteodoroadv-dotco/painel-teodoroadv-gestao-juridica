# Agenda o backup diário do Painel Teodoro no Windows. Rode UMA vez, por você, no PowerShell:
#     powershell -ExecutionPolicy Bypass -File \\wsl.localhost\Ubuntu\home\user\projetos\painel-teodoro-v2\deploy\agendar-backup-windows.ps1
# Cria a tarefa "Painel Teodoro - Backup diario": todo dia às 03:30; se o computador estiver desligado nessa hora,
# roda assim que ele voltar. O script de backup pula se já houve backup nas últimas 12 horas.
# Para remover: Unregister-ScheduledTask -TaskName "Painel Teodoro - Backup diario" -Confirm:$false
$nome = "Painel Teodoro - Backup diario"
$acao = New-ScheduledTaskAction -Execute "wsl.exe" -Argument '-e bash -lc "~/projetos/painel-teodoro-v2/deploy/backup-local.sh"'
$diario = New-ScheduledTaskTrigger -Daily -At 3:30am
$cfg = New-ScheduledTaskSettingsSet -StartWhenAvailable -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Hours 1)
$desc = "Backup cifrado diario do Painel Teodoro (banco + documentos) para D:\OneDrive\Backups-Painel."

# Tenta também rodar ao entrar no Windows (precisa de administrador em alguns computadores); se negar, fica só o diário.
try {
    $logon = New-ScheduledTaskTrigger -AtLogOn; $logon.Delay = "PT3M"
    Register-ScheduledTask -TaskName $nome -Action $acao -Trigger @($diario, $logon) -Settings $cfg -Description $desc -Force -ErrorAction Stop | Out-Null
    Write-Host "OK: tarefa criada (diária 03:30 + ao entrar no Windows)."
} catch {
    try {
        Register-ScheduledTask -TaskName $nome -Action $acao -Trigger $diario -Settings $cfg -Description $desc -Force -ErrorAction Stop | Out-Null
        Write-Host "OK: tarefa criada (diária 03:30; sem o gatilho 'ao entrar', que exige administrador)."
    } catch {
        Write-Host "FALHOU: $($_.Exception.Message)"
        Write-Host "Tente abrir o PowerShell como Administrador e rodar de novo."
        exit 1
    }
}
Get-ScheduledTask -TaskName $nome | Select-Object TaskName, State | Format-List
Write-Host "Para testar agora:  Start-ScheduledTask -TaskName '$nome'   e depois veja D:\OneDrive\Backups-Painel\backup.log"
