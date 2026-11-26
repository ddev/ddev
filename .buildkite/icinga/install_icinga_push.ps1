# Installs testbot_icinga_push.ps1 on a Windows testbot as a Scheduled Task that
# runs every 5 minutes, like install_icinga_push.sh does on macOS. Safe to rerun:
# it re-registers the task only when the script or the task definition changed.
# testbot_maintenance.sh runs it on main builds once ~/.config/ddev-icinga/env
# exists, so runners pick up merged changes. Needs an elevated session.

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

$TaskName = 'DDEV Icinga push'
# Change this when the task definition below changes, so testbots re-register it.
$TaskVersion = 'ddev-icinga-push v1'
$ConfigDir = Join-Path $HOME '.config\ddev-icinga'
$Target = Join-Path $ConfigDir 'testbot_icinga_push.ps1'
$SourceScript = Join-Path $PSScriptRoot 'testbot_icinga_push.ps1'

if (-not (Test-Path (Join-Path $ConfigDir 'env')) -or -not (Test-Path (Join-Path $ConfigDir 'ca.crt'))) {
  Write-Error "$ConfigDir\env and $ConfigDir\ca.crt must exist first"
}

$changed = $false
if (-not (Test-Path $Target) -or
    (Get-FileHash $SourceScript).Hash -ne (Get-FileHash $Target).Hash) {
  Copy-Item $SourceScript $Target -Force
  $changed = $true
}

$task = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($changed -or -not $task -or $task.Description -ne $TaskVersion) {
  $action = New-ScheduledTaskAction -Execute 'powershell.exe' `
    -Argument "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File `"$Target`""
  $trigger = New-ScheduledTaskTrigger -Once -At (Get-Date) -RepetitionInterval (New-TimeSpan -Minutes 5)
  # S4U runs the task without a window, whether or not testbot is logged in.
  $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType S4U -RunLevel Limited
  $settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -MultipleInstances IgnoreNew `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 4) -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
  $null = Register-ScheduledTask -TaskName $TaskName -Description $TaskVersion -Action $action `
    -Trigger $trigger -Principal $principal -Settings $settings -Force
  Start-ScheduledTask -TaskName $TaskName
  Write-Output "Registered and started scheduled task '$TaskName'"
}
