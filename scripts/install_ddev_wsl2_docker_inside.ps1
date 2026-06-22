# This PowerShell script tries to do almost all the things required to set up
# a Debian-based WSL2 instance for use with DDEV and docker-ce installed inside WSL2.
#
# **DDEV now ships with a GUI installer for Windows/WSL2 which is usually easier.**
# See https://ddev.com/download
#
# Prerequisites:
# - A Debian-based WSL2 distro installed, e.g. Ubuntu or Debian (preferably with `wsl --install`)
# - Docker Desktop must NOT be installed, or WSL2 integration must be disabled for this distro
#
# The -Distro argument is required. To find available distros: wsl -l -v
#
# Quick install - run this one-liner in PowerShell (replace "Ubuntu" with your distro name):
#   & ([scriptblock]::Create((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_inside.ps1'))) -Distro "Ubuntu"
#
# Alternatively, download the script first to inspect it, then run with:
#   Set-ExecutionPolicy Bypass -Scope Process -Force
#   .\install_ddev_wsl2_docker_inside.ps1 -Distro "Ubuntu"

param(
    [Parameter(Mandatory=$true, HelpMessage="Name of the WSL2 distro to install into (e.g. 'Ubuntu'). Find yours with: wsl -l -v")]
    [string]$Distro
)

# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
}
# Make sure named distro is a Debian-based release
$osRelease = wsl -d $Distro -e cat /etc/os-release
if (-not ($osRelease -match 'ID(_LIKE)?=.*ubuntu' -or $osRelease -match 'ID(_LIKE)?=.*debian')) {
    throw "Distro '$Distro' does not appear to be Debian-based (Ubuntu, Debian, etc.). You can certainly use DDEV with WSL2 in another distro, but this script is oriented to Debian-based distros."
}
# Make sure using WSL2
if (-not (wsl -d $Distro -e bash -c "env | grep WSL_INTEROP=")) {
    throw "Distro '$Distro' is not WSL version 2, please delete it and start over again"
}
if (-not(Compare-Object "root" (wsl -d $Distro -e whoami)) ) {
    throw "The default user in '$Distro' seems to be root. Please configure an ordinary default user"
}

if (wsl -d $Distro bash -c "test -d /mnt/wsl/docker-desktop >/dev/null 2>&1" ) {
    throw "Docker Desktop integration is enabled with '$Distro' and it must be turned off."
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

wsl -d $Distro -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"
wsl -d $Distro -u root apt-get update
wsl -d $Distro -u root apt-get install -y ca-certificates curl gnupg lsb-release
wsl -d $Distro -u root install -m 0755 -d /etc/apt/keyrings
# Configure the Docker apt repo. Use a PS double-quoted string with backtick-
# escaped $ so PowerShell does not interpolate bash variables, and use bash
# single quotes for the printf format string. This avoids the Windows
# CreateProcess argument-splitting bug that occurs when a PS single-quoted
# string containing " chars is passed to wsl.exe — CreateProcess treats those
# " as argument delimiters and splits the bash command at the first ".
wsl -d $Distro -u root -e bash -c "rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list; . /etc/os-release; if echo `$ID `$ID_LIKE | grep -qi ubuntu; then FAMILY=ubuntu; else FAMILY=debian; fi; curl -fsSL https://download.docker.com/linux/`$FAMILY/gpg -o /etc/apt/keyrings/docker.asc && chmod a+r /etc/apt/keyrings/docker.asc && printf 'Types: deb\nURIs: https://download.docker.com/linux/%s\nSuites: %s\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n' `$FAMILY `${UBUNTU_CODENAME:-`$VERSION_CODENAME} | tee /etc/apt/sources.list.d/docker.sources > /dev/null"

wsl -d $Distro -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg /etc/apt/sources.list.d/ddev.list && curl -fsSL https://pkg.ddev.com/apt/gpg.key | tee /etc/apt/keyrings/ddev.asc > /dev/null && chmod a+r /etc/apt/keyrings/ddev.asc"
wsl -d $Distro -u root -e bash -c "printf 'Types: deb\nURIs: https://pkg.ddev.com/apt/\nSuites: *\nComponents: *\nSigned-By: /etc/apt/keyrings/ddev.asc\n' > /etc/apt/sources.list.d/ddev.sources"
wsl -d $Distro -u root -e bash -c "apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io"
wsl -d $Distro -u root -e bash -c "apt-get install -y --no-install-recommends ddev ddev-wsl2"
$wslUser = (wsl -d $Distro whoami).Trim()
wsl -d $Distro -u root usermod -aG docker $wslUser

wsl -d $Distro mkcert.exe -install
$env:CAROOT = & wsl -d $Distro mkcert.exe -CAROOT
setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }

# Convert the Windows CAROOT path to a Linux path and pass it directly to mkcert,
# avoiding a wsl --terminate which breaks Docker Desktop integration.
$linuxCaRoot = (& wsl -d $Distro wslpath -u ($env:CAROOT -replace '\\', '/')).Trim()
Write-Host "Linux CAROOT: $linuxCaRoot"
try {
    wsl -d $Distro -u root -e bash -c "echo 'ALL ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/temp-mkcert-install && chmod 440 /etc/sudoers.d/temp-mkcert-install"
    if ($LASTEXITCODE -ne 0) { throw "Failed to create temporary sudoers entry (exit $LASTEXITCODE)" }
    wsl -d $Distro bash -c "CAROOT='$linuxCaRoot' mkcert -install"
} finally {
    wsl -d $Distro -u root rm -f /etc/sudoers.d/temp-mkcert-install
}
if (-not(wsl -d $Distro -e docker ps)) {
    throw "docker does not seem to be working inside '$Distro' yet."
}
# If docker desktop was previously set up, the .docker can break normal use of docker client.
wsl -d $Distro rm -rf ~/.docker

wsl -d $Distro ddev version
