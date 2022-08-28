
#Requires -RunAsAdministrator


# Make sure wsl is installed and working
if (-not(wsl -l -v)) {
    throw "WSL2 does not seem to be installed yet; please install it with 'wsl --install'"
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
wsl -u root bash -c "apt update >/dev/null && sudo apt install -y ddev"

wsl bash -c 'echo $CAROOT'
wsl -u root mkcert -install
if (-not(wsl docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet"
}
