# This PowerShell script tries to do almost all the things required to set up
# a Debian-based WSL2 instance for use with DDEV and Docker Desktop.
#
# **DDEV now ships with a GUI installer for Windows/WSL2 which is usually easier.**
# See https://ddev.com/download
#
# Prerequisites:
# - A Debian-based WSL2 distro installed, e.g. Ubuntu or Debian (preferably with `wsl --install`)
# - The distro you want must be set as the default WSL2 distro
# - Docker Desktop installed, running, and with WSL2 integration enabled for your distro
#
# Quick install - run this one-liner in PowerShell:
#   iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_desktop.ps1'))
#
# Alternatively, download the script first to inspect it, then run with:
#   Set-ExecutionPolicy Bypass -Scope Process -Force
#   .\install_ddev_wsl2_docker_desktop.ps1

# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
}
# Make sure default distro is a Debian-based release
$osRelease = wsl -e cat //etc/os-release
if (-not ($osRelease -match 'ID(_LIKE)?=.*ubuntu' -or $osRelease -match 'ID(_LIKE)?=.*debian')) {
    throw "Your installed WSL2 distro does not appear to be Debian-based (Ubuntu, Debian, etc.). You can certainly use DDEV with WSL2 in another distro, but this script is oriented to Debian-based distros."
}
# Make sure using WSL2
if (-not (wsl -e bash -c "env | grep WSL_INTEROP=")) {
    throw "Your default distro is not WSL version 2, please delete it and start over again"
}
if (-not(Compare-Object "root" (wsl -e whoami)) ) {
    throw "The default user in your distro seems to be root. Please configure an ordinary default user"
}
if (-not(Get-Command docker 2>&1 ) -Or -Not(docker ps ) ) {
    throw "\n\ndocker does not seem to be installed yet, or Docker Desktop is not running. Please install it or start it."
}

if (-not(wsl -e docker ps) ) {
    throw "Docker Desktop integration with the default distro does not seem to be enabled yet."
}
$ErrorActionPreference = "Stop"
# On PowerShell 7.4+, $PSNativeCommandUseErrorActionPreference defaults to $true,
# which would make a non-zero exit from a native command (wsl/curl/docker) a
# terminating error and bypass the explicit "if (-not(...)) { throw ... }" guards
# below. Set it to $false so behavior matches Windows PowerShell 5.1 (on 5.1 this
# is just an unused variable). Native-command results are checked explicitly.
$PSNativeCommandUseErrorActionPreference = $false

# Remove old Windows ddev.exe if it exists using uninstaller
# Check both old system-wide location and new per-user location
if (Test-Path "$env:PROGRAMFILES\DDEV\ddev_uninstall.exe") {
    Write-Host "Removing old Windows ddev.exe installation (system-wide)"
    $proc = Start-Process "$env:PROGRAMFILES\DDEV\ddev_uninstall.exe" -ArgumentList "/S" -PassThru
    if (-not $proc.WaitForExit(120000)) {
        Write-Warning "DDEV uninstaller did not complete within 2 minutes; killing it"
        $proc.Kill()
    }
}
if (Test-Path "$env:LOCALAPPDATA\Programs\DDEV\ddev_uninstall.exe") {
    Write-Host "Removing old Windows ddev.exe installation (per-user)"
    $proc = Start-Process "$env:LOCALAPPDATA\Programs\DDEV\ddev_uninstall.exe" -ArgumentList "/S" -PassThru
    if (-not $proc.WaitForExit(120000)) {
        Write-Warning "DDEV uninstaller did not complete within 2 minutes; killing it"
        $proc.Kill()
    }
}

wsl -u root -e bash -c "apt-get update && apt-get install -y curl"
wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg /etc/apt/sources.list.d/ddev.list && curl -fsSL https://pkg.ddev.com/apt/gpg.key | tee /etc/apt/keyrings/ddev.asc > /dev/null && chmod a+r /etc/apt/keyrings/ddev.asc"
wsl -u root -e bash -c "printf 'Types: deb\nURIs: https://pkg.ddev.com/apt/\nSuites: *\nComponents: *\nSigned-By: /etc/apt/keyrings/ddev.asc\n' > /etc/apt/sources.list.d/ddev.sources"

wsl -u root -e bash -c "apt-get update && apt-get install -y --no-install-recommends ddev ddev-wsl2"

wsl mkcert.exe -install
$env:CAROOT = & wsl mkcert.exe -CAROOT
setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }

$defaultDistro = (wsl --list --quiet | Select-Object -First 1) -replace '[\r\n\x00-\x1F\x7F-\x9F]', '' -replace '^\s+|\s+$', ''
Write-Host "Terminating default WSL2 distro: $defaultDistro"
wsl --terminate $defaultDistro

wsl bash -c 'echo CAROOT=$CAROOT'
try {
    wsl -u root -e bash -c "echo 'ALL ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/temp-mkcert-install && chmod 440 /etc/sudoers.d/temp-mkcert-install"
    if ($LASTEXITCODE -ne 0) { throw "Failed to create temporary sudoers entry (exit $LASTEXITCODE)" }
    wsl mkcert -install
} finally {
    wsl -u root rm -f /etc/sudoers.d/temp-mkcert-install
}
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. Check Resources->WSL Integration in Docker Desktop"
}

wsl ddev version
