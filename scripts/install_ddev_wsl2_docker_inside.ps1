# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and docker-ce installed inside WSL2.
# These days it's very unusual to do this because DDEV v1.24.7+ ships with a GUI installer for Windows/WSL2
# So use that instead.
# This requires that an Ubuntu-based WSL2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually. The distro you want to act on must be set to the default WSL2 distro.
#
# You can download, inspect, and run this, or run it directly with
# Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
# iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_inside.ps1'))

# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
}
# Make sure default distro an ubuntu release
if (-not( wsl -e grep ^NAME=.Ubuntu //etc/os-release)) {
    throw "Your installed WSL2 distro does not seem to be Ubuntu. You can certainly use DDEV with WSL2 in another distro, but this script is oriented to Ubuntu."
}
# Make sure using WSL2
if (-not (wsl -e bash -c "env | grep WSL_INTEROP=")) {
    throw "Your default distro is not WSL version 2, please delete it and start over again"
}
if (-not(Compare-Object "root" (wsl -e whoami)) ) {
    throw "The default user in your distro seems to be root. Please configure an ordinary default user"
}

if (wsl bash -c "test -d /mnt/wsl/docker-desktop >/dev/null 2>&1" ) {
    throw "Docker Desktop integration is enabled with the default distro and it must be turned off."
}
$ErrorActionPreference = "Stop"

# Remove old Windows ddev.exe if it exists using uninstaller
if (Test-Path "$env:PROGRAMFILES\DDEV\ddev_uninstall.exe") {
    Write-Host "Removing old Windows ddev.exe installation"
    Start-Process "$env:PROGRAMFILES\DDEV\ddev_uninstall.exe" -ArgumentList "/SILENT" -Wait
}

wsl -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"
wsl -u root apt-get update
wsl -u root apt-get install -y ca-certificates curl gnupg lsb-release
wsl -u root install -m 0755 -d /etc/apt/keyrings
wsl -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"
wsl -u root -e bash -c 'echo deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu  $(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1'

wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io wslu"
wsl -u root -e bash -c "apt-get install -y --no-install-recommends ddev ddev-wsl2"
wsl bash -c 'sudo usermod -aG docker $USER'

setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }
$defaultDistro = (wsl --list --quiet | Select-Object -First 1) -replace '[\r\n\x00-\x1F\x7F-\x9F]', '' -replace '^\s+|\s+$', ''
Write-Host "Terminating default WSL2 distro: $defaultDistro"
wsl --terminate $defaultDistro

wsl bash -c 'echo CAROOT=$CAROOT'
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. "
}
# If docker desktop was previously set up, the .docker can break normal use of docker client.
wsl rm -rf ~/.docker

wsl ddev version
