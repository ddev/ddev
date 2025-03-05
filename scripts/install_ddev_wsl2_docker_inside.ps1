# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and docker-ce installed inside WSL2.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
# Run this in an administrative PowerShell window.
# You can download, inspect, and run this, or run it directly with
# Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
# iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_inside.ps1'))

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

if (wsl bash -c "test -d /mnt/wsl/docker-desktop >/dev/null 2>&1" ) {
    throw "Docker Desktop integration is enabled with the default distro and it must but turned off."
}
$ErrorActionPreference = "Stop"

# Install binaries required for updating the hosts file on windows.

# Set the executable path for DDEV.
$ExecutablesDirectoryPath = Join-Path $env:ProgramFiles "DDEV"
if (!(Test-Path $ExecutablesDirectoryPath)) {
    New-Item -ItemType Directory -Path $ExecutablesDirectoryPath | Out-Null
}
$existingPath = (Get-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" -Name PATH).Path
if ($existingPath -notlike "*$ExecutablesDirectoryPath*") {
    $newPath = $existingPath + ";" + $ExecutablesDirectoryPath
    Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" -Name PATH -Value $newPath
    # Tell the running session to pick up the updated environment (though new processes will see it automatically)
    $env:Path = $env:Path + ";" + $ExecutablesDirectoryPath
}

# Install DDEV.
$DdevBinaryPath = Join-Path $ExecutablesDirectoryPath "ddev.exe"
# TODO: To use https://github.com/<OWNER>/<REPO>/releases/latest/download/myprogram.exe,
# there can't be version numbers in the file name; it needs to be the same for every release.
# For now, just use the latest one available.
Invoke-WebRequest `
    -Uri "https://github.com/ddev/ddev/releases/download/v1.24.3/ddev_windows_amd64_installer.v1.24.3.exe" `
    -OutFile $DdevBinaryPath

# Install Sudo for Windows.
Set-ExecutionPolicy RemoteSigned -scope Process
[Net.ServicePointManager]::SecurityProtocol = 'Tls12'
# Don't use `master` here because that means we trust that project's branch forever,
# which is a security risk. Stick to known trustworthy releases.
iwr -UseBasicParsing https://raw.githubusercontent.com/gerardog/gsudo/v2.6.0/installgsudo.ps1 | iex

# TODO: install mkcert

mkcert -install
$env:CAROOT="$(mkcert -CAROOT)"
setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }

wsl -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"
wsl -u root apt-get update
wsl -u root apt-get install -y ca-certificates curl gnupg lsb-release
wsl -u root install -m 0755 -d /etc/apt/keyrings
wsl -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"
wsl -u root -e bash -c 'echo deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu  $(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1'

wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "apt-get update && apt-get install -y ddev docker-ce docker-ce-cli containerd.io wslu"
wsl -u root -e bash -c "apt-get upgrade -y >/dev/null"
wsl bash -c 'sudo usermod -aG docker $USER'

wsl bash -c 'echo CAROOT=$CAROOT'
wsl -u root mkcert -install
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. "
}
# If docker desktop was previously set up, the .docker can break normal use of docker client.
wsl rm -rf ~/.docker

wsl -u root -e bash -c "touch /etc/wsl.conf && if ! fgrep '[boot]' /etc/wsl.conf >/dev/null; then printf '\n[boot]\nsystemd=true\n' >>/etc/wsl.conf; fi"

wsl ddev version
