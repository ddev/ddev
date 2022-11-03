# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and docker-ce installed inside WSL2.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
# Run this in an administrative PowerShell window.
# You can download, inspect, and run this, or run it directly with
# Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
# iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/drud/ddev/master/scripts/install_ddev_wsl2.ps1'))

#Requires -RunAsAdministrator

# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
}
# Make sure default distro an ubuntu release
if (-not( wsl -e grep ^NAME=.Ubuntu //etc/os-release)) {
    throw "Your installed WSL2 distro does not seem to be Ubuntu. You can certainly use DDEV with WSL2 in another distro, but this script is oriented to Ubuntu."
}
if (-not(Compare-Object "root" (wsl -e whoami)) ) {
    throw "The default user in your distro seems to be root. Please configure an ordinary default user"
}
# Install Chocolatey if needed
if (-not(choco 2>&1 >$null)) {
    "Chocolatey does not appear to be installed yet, installing"
    $ErrorActionPreference = "Stop"
    Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}

if (wsl bash "if [ -d /mnt/wsl/docker-desktop ]; then echo yup; fi" ) {
    throw "Docker Desktop integration is enabled with the default distro and it must but turned off."
}
$ErrorActionPreference = "Stop"
# Install needed choco items
choco install -y mkcert

mkcert -install
setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }

wsl -u root apt-get remove -y docker docker-engine docker.io containerd runc
wsl -u root apt-get update
wsl -u root apt-get install -y ca-certificates curl gnupg lsb-release
wsl -u root mkdir -p /etc/apt/keyrings
wsl -u root bash -c "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg"
wsl -u root -e bash -c 'echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu  $(lsb_release -cs) stable\" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null'

wsl -u root -e bash -c "curl -fsSL https://apt.fury.io/drud/gpg.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo \"deb [signed-by=/etc/apt/trusted.gpg.d/ddev.gpg] https://apt.fury.io/drud/ * *\" > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "sudo apt update && sudo apt install -y ddev docker-ce docker-ce-cli containerd.io"
wsl -u root -e bash -c "apt-get upgrade -y >/dev/null"
wsl bash -c 'sudo usermod -aG docker $USER'

wsl bash -c 'echo CAROOT=$CAROOT'
wsl -u root mkcert -install
wsl -u root service docker start
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. "
}

wsl ddev version
