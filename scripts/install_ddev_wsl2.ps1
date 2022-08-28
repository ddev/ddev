# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
# It requires that Docker Desktop is installed and running, and that it has integration enabled with the Ubuntu
# distro, which is the default behavior.
# Run this in an administrative PowerShell window.

#Requires -RunAsAdministrator

# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
}
# Make sure it's an ubuntu release
if (-not( wsl -e grep ^NAME=.Ubuntu //etc/os-release)) {
    throw "Your installed WSL2 distro does not seem to be Ubuntu. You can certainly use DDEV with WSL2 in another distro, but this script is oriented to Ubuntu."
}
# Install Chocolatey if needed
if (-not(Get-Command choco 2>&1 >$null))
{
    "Chocolatey does not appear to be installed yet, installing"
    $ErrorActionPreference = "Stop"
    Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}
if (-not(Get-Command docker 2>&1 ) -Or -Not(docker ps ) ) {
    throw "docker does not seem to be installed yet, or Docker Desktop is not running. Please install it or start it. For example, choco install -y docker-desktop"
}

$ErrorActionPreference = "Stop"
# Install needed choco items
choco install -y gsudo mkcert

mkcert -install
setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }

wsl -u root bash -c "curl -sL https://apt.fury.io/drud/gpg.key | apt-key add -"
wsl -u root bash -c "echo 'deb https://apt.fury.io/drud/ * *' > /etc/apt/sources.list.d/ddev.list"
wsl -u root bash -c "apt-get update >/dev/null && sudo apt-get install -y ddev"

wsl bash -c 'echo $CAROOT'
wsl -u root mkcert -install
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. Check Resources->WSL Integration in Docker Desktop"
}
