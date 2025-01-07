# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and Docker Desktop.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
# It requires that Docker Desktop is installed and running, and that it has integration enabled with the Ubuntu
# distro, which is the default behavior.
# Run this in an administrative PowerShell window.
# You can download, inspect, and run this, or run it directly with
# Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
# iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_desktop.ps1'))

#Requires -RunAsAdministrator

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
# Install Chocolatey if needed
if (-not (Get-Command "choco" -errorAction SilentlyContinue))
{
    "Chocolatey does not appear to be installed yet, installing"
    $ErrorActionPreference = "Stop"
    Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}
if (-not(Get-Command docker 2>&1 ) -Or -Not(docker ps ) ) {
    throw "\n\ndocker does not seem to be installed yet, or Docker Desktop is not running. Please install it or start it. For example, choco install -y docker-desktop"
}

if (-not(wsl -e docker ps) ) {
    throw "Docker Desktop integration with the default distro does not seem to be enabled yet."
}
$ErrorActionPreference = "Stop"
# Install needed choco items
choco upgrade -y ddev gsudo mkcert

mkcert -install
$env:CAROOT="$(mkcert -CAROOT)"
setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }

wsl -u root -e bash -c "apt-get update && apt-get install -y curl"
wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "apt-get update && apt-get install -y ddev wslu"
wsl -u root -e bash -c "apt-get upgrade -y >/dev/null"


wsl bash -c 'echo $CAROOT'
wsl -u root mkcert -install
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. Check Resources->WSL Integration in Docker Desktop"
}
wsl -u root -e bash -c "touch /etc/wsl.conf && if ! fgrep '[boot]' /etc/wsl.conf >/dev/null; then printf '\n[boot]\nsystemd=true\n' >>/etc/wsl.conf; fi"

refreshenv

wsl ddev version
