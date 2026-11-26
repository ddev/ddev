# Runs health checks on a Windows testbot and pushes each result to Icinga as a
# passive check result, like testbot_icinga_push.sh does on macOS. A Scheduled
# Task runs it every 5 minutes; the setup is in "Icinga2 Monitoring for macOS
# and Windows Runners" in buildkite-testmachine-setup.md.
#
# Reads ICINGA_URL, ICINGA_HOST, ICINGA_API_USER and ICINGA_API_PASSWORD from
# ~/.config/ddev-icinga/env, and the Icinga CA from ~/.config/ddev-icinga/ca.crt.
# With -DryRun it prints the results instead of sending them.

param([switch]$DryRun)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Get-Setting([string]$Name, $Default) {
  $value = [Environment]::GetEnvironmentVariable($Name)
  if ($value) { return $value } else { return $Default }
}

$ConfigDir = Get-Setting 'DDEV_ICINGA_CONFIG_DIR' (Join-Path $HOME '.config\ddev-icinga')
$DiskWarnPct = [int](Get-Setting 'DISK_WARN_PCT' 80)
$DiskCritPct = [int](Get-Setting 'DISK_CRIT_PCT' 90)
$JobWarnHours = [int](Get-Setting 'JOB_WARN_HOURS' 4)
$JobCritHours = [int](Get-Setting 'JOB_CRIT_HOURS' 8)
$ProbeTimeoutSeconds = [int](Get-Setting 'PROBE_TIMEOUT_SECONDS' 20)

$OK = 0; $WARNING = 1; $CRITICAL = 2; $UNKNOWN = 3

$Config = @{}
$EnvFile = Join-Path $ConfigDir 'env'
if (Test-Path $EnvFile) {
  foreach ($line in Get-Content $EnvFile) {
    if ($line -match '^\s*([A-Z_]+)=(.*)$') { $Config[$Matches[1]] = $Matches[2].Trim().Trim('"', "'") }
  }
}
$Source = $env:COMPUTERNAME.ToLower()
$IcingaHost = if ($Config['ICINGA_HOST']) { $Config['ICINGA_HOST'] } else { $Source }
$LogFile = Join-Path $ConfigDir 'push.log'

function Write-Log([string]$Message) {
  $line = '{0} {1}' -f (Get-Date -Format 's'), $Message
  if ($DryRun) { Write-Warning $line } else { Add-Content -Path $LogFile -Value $line }
}

# An empty service name sends the host check result.
function Send-Result([string]$Service, [int]$Status, [string]$Output, [string[]]$PerfData = @()) {
  $perf = @($PerfData | Where-Object { $_ })
  if ($DryRun) {
    $name = if ($Service) { $Service } else { '<host>' }
    Write-Output ('{0} [{1}] {2} | {3}' -f $name, $Status, $Output, ($perf -join ' '))
    return
  }
  $body = [ordered]@{
    exit_status      = $Status
    plugin_output    = $Output
    performance_data = $perf
    check_source     = $Source
  }
  if ($Service) {
    $body.type = 'Service'
    $body.filter = 'host.name == h && service.name == n'
    $body.filter_vars = @{ h = $IcingaHost; n = $Service }
  } else {
    $body.type = 'Host'
    $body.filter = 'host.name == h'
    $body.filter_vars = @{ h = $IcingaHost }
  }
  # Windows PowerShell mangles quotes in native arguments, so the body goes in a file.
  $bodyFile = [IO.Path]::GetTempFileName()
  try {
    [IO.File]::WriteAllText($bodyFile, ($body | ConvertTo-Json -Compress -Depth 4))
    # The Icinga CA publishes no revocation list, which Schannel checks by default.
    $out = & curl.exe -sS --fail-with-body --max-time 30 --ssl-no-revoke `
      --cacert (Join-Path $ConfigDir 'ca.crt') `
      -u ('{0}:{1}' -f $Config['ICINGA_API_USER'], $Config['ICINGA_API_PASSWORD']) `
      -H 'Accept: application/json' -X POST `
      ('{0}/v1/actions/process-check-result' -f $Config['ICINGA_URL']) `
      --data-binary "@$bodyFile" 2>&1
    if ($LASTEXITCODE -ne 0) {
      Write-Log ('Failed to push {0} result to {1}: {2}' -f $(if ($Service) { $Service } else { 'host' }), $Config['ICINGA_URL'], ($out -join ' '))
    }
  } finally {
    Remove-Item $bodyFile -ErrorAction SilentlyContinue
  }
}

# Invoke-Docker runs docker.exe and kills it after the timeout, returning exit
# code 124 like timeout(1), so a hung daemon can't hang this script.
function Invoke-Docker([string]$Arguments) {
  $psi = New-Object Diagnostics.ProcessStartInfo $script:Docker, $Arguments
  $psi.UseShellExecute = $false
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.CreateNoWindow = $true
  $process = [Diagnostics.Process]::Start($psi)
  $stdout = $process.StandardOutput.ReadToEndAsync()
  $null = $process.StandardError.ReadToEndAsync()
  if (-not $process.WaitForExit($ProbeTimeoutSeconds * 1000)) {
    try { $process.Kill() } catch { }
    return @{ Code = 124; Out = '' }
  }
  return @{ Code = $process.ExitCode; Out = $stdout.Result }
}

function Get-JobProcesses {
  @(Get-CimInstance Win32_Process -Filter "Name = 'buildkite-agent.exe'" |
    Where-Object { $_.CommandLine -match '\sbootstrap(\s|$)' })
}

function Test-Host {
  $uptime = (Get-Date) - (Get-CimInstance Win32_OperatingSystem).LastBootUpTime
  Send-Result '' $OK ('UP: up {0} days, {1:hh\:mm}' -f $uptime.Days, $uptime)
}

function Test-Disk {
  $drive = Get-PSDrive C
  $usedPct = [math]::Floor($drive.Used * 100 / ($drive.Used + $drive.Free))
  $freeGb = [math]::Floor($drive.Free / 1GB)
  $status = $OK
  if ($usedPct -ge $DiskCritPct) { $status = $CRITICAL } elseif ($usedPct -ge $DiskWarnPct) { $status = $WARNING }
  Send-Result 'disk' $status ('C: {0}% used, {1}GB free' -f $usedPct, $freeGb) `
    @("used_pct=$usedPct%;$DiskWarnPct;$DiskCritPct;0;100", "free=${freeGb}GB")
}

# Test-BuildkiteAgent wants the agent running and no job older than the limits.
function Test-BuildkiteAgent {
  if (-not (Get-Process buildkite-agent -ErrorAction SilentlyContinue)) {
    Send-Result 'buildkite-agent' $CRITICAL 'buildkite-agent is not running'
    return
  }
  $longest = 0
  foreach ($job in Get-JobProcesses) {
    $seconds = [int]((Get-Date) - $job.CreationDate).TotalSeconds
    if ($seconds -gt $longest) { $longest = $seconds }
  }
  $hours = [math]::Floor($longest / 3600)
  $msg = if ($longest -eq 0) { 'buildkite-agent running, idle' } else {
    'buildkite-agent running, current job {0}h {1}m old' -f $hours, [math]::Floor(($longest % 3600) / 60)
  }
  $status = $OK
  if ($hours -ge $JobCritHours) { $status = $CRITICAL } elseif ($hours -ge $JobWarnHours) { $status = $WARNING }
  Send-Result 'buildkite-agent' $status $msg @("job_seconds=${longest}s;$($JobWarnHours * 3600);$($JobCritHours * 3600)")
}

# Test-DockerProviders probes each Docker context whose named pipe exists. A
# missing pipe means the provider is stopped; a probe that times out means a
# hung daemon. Contexts that are up are kept for Test-LeftoverContainers.
$UpContexts = @()
function Test-DockerProviders {
  if (-not $script:Docker) {
    Send-Result 'docker-providers' $UNKNOWN 'docker.exe not found'
    return
  }
  $list = Invoke-Docker 'context ls --format {{.Name}}'
  if ($list.Code -ne 0) {
    Send-Result 'docker-providers' $UNKNOWN ('docker context ls failed with exit code {0}' -f $list.Code)
    return
  }
  $up = @(); $stopped = @(); $hung = @(); $status = $OK
  foreach ($ctx in ($list.Out -split "`r?`n" | Where-Object { $_ -and $_ -ne 'default' })) {
    $endpoint = (Invoke-Docker "context inspect -f {{.Endpoints.docker.Host}} $ctx").Out.Trim()
    if ($endpoint -match '^npipe:////\./pipe/(.+)$' -and -not (Test-Path "\\.\pipe\$($Matches[1])")) {
      $stopped += $ctx
      continue
    }
    $probe = Invoke-Docker "--context $ctx version --format {{.Server.Version}}"
    if ($probe.Code -eq 0) { $up += $ctx }
    elseif ($probe.Code -eq 124) { $hung += $ctx; $status = $CRITICAL }
    else { $stopped += $ctx }
  }
  $script:UpContexts = $up
  $fmt = { param($a) if ($a.Count) { $a -join ' ' } else { 'none' } }
  Send-Result 'docker-providers' $status ('hung: {0}; up: {1}; stopped: {2}' -f (& $fmt $hung), (& $fmt $up), (& $fmt $stopped)) `
    @("up=$($up.Count)", "hung=$($hung.Count);;1")
}

# Test-LeftoverContainers reports containers left in any running provider while
# no job runs; they can hold ports a later job needs.
function Test-LeftoverContainers {
  if (@(Get-JobProcesses).Count -gt 0) {
    Send-Result 'leftover-containers' $OK 'Job running, not checked'
    return
  }
  $total = 0; $detail = ''
  foreach ($ctx in $script:UpContexts) {
    $count = @((Invoke-Docker "--context $ctx ps -aq").Out -split "`r?`n" | Where-Object { $_ }).Count
    if ($count -gt 0) { $total += $count; $detail += " $ctx=$count" }
  }
  if ($total -gt 0) {
    Send-Result 'leftover-containers' $WARNING ('{0} container(s) with no job running:{1}' -f $total, $detail) @("containers=$total")
  } else {
    Send-Result 'leftover-containers' $OK 'No containers with no job running' @('containers=0')
  }
}

if (-not $DryRun) {
  foreach ($var in 'ICINGA_URL', 'ICINGA_API_USER', 'ICINGA_API_PASSWORD') {
    if (-not $Config[$var]) {
      Write-Log "$var is not set in $EnvFile"
      exit 1
    }
  }
}

# A Scheduled Task may not have Docker Desktop's directory on PATH.
$Docker = (Get-Command docker.exe -ErrorAction SilentlyContinue).Source
if (-not $Docker) {
  $Docker = @(
    (Join-Path $env:LOCALAPPDATA 'Programs\DockerDesktop\resources\bin\docker.exe'),
    (Join-Path $env:ProgramFiles 'Docker\Docker\resources\bin\docker.exe')
  ) | Where-Object { Test-Path $_ } | Select-Object -First 1
}

foreach ($check in 'Test-Host', 'Test-Disk', 'Test-BuildkiteAgent', 'Test-DockerProviders', 'Test-LeftoverContainers') {
  try {
    & $check
  } catch {
    Write-Log "$check failed: $_"
  }
}
