
#Requires -RunAsAdministrator

$ErrorActionPreference = "Stop"
# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'" 
}
# Install Chocolatey if needed
if (-not(Get-Command choco))
{
    "Chocolatey does not appear to be installed yet, installing"
    Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
}
if (-not(Get-Command docker) -Or -Not(docker ps) ) {
    throw "docker does not seem to be installed yet, or Docker Desktop is not running. Please install it or start it. For example, choco install -y docker-desktop"
}
# Install needed choco items
choco install -y gsudo mkcert

mkcert -install
setx CAROOT "$(mkcert -CAROOT)"; If ($Env:WSLENV -notlike "*CAROOT/up:*") { setx WSLENV "CAROOT/up:$Env:WSLENV" }

wsl -u root bash -c "curl -sL https://apt.fury.io/drud/gpg.key | apt-key add -"
wsl -u root bash -c 'echo "deb https://apt.fury.io/drud/ * *" | tee /etc/apt/sources.list.d/ddev.list'
wsl -u root bash -c "apt update && sudo apt install -y ddev"

wsl bash -c 'echo $CAROOT'
wsl docker ps
