# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and docker-ce installed inside WSL2.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
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
    throw "Docker Desktop integration is enabled with the default distro and it must but turned off."
}
$ErrorActionPreference = "Stop"

# Determine the architecture we're running on to fetch the correct installer.
$realArchitecture = $env:PROCESSOR_ARCHITEW6432
if (-not $realArchitecture) {
    $realArchitecture = $env:PROCESSOR_ARCHITECTURE
}
switch ($realArchitecture) {
    "AMD64" {
        $architectureForInstaller = "amd64"
    }
    "ARM64" {
        $architectureForInstaller = "arm64"
    }
    "x86" {
        Write-Error "Error: x86 Windows detected, which is not supported."
        exit 1
    }
    Default {
        $architectureForInstaller = "amd64"
    }
}
Write-Host "Detected OS architecture: $realArchitecture; using DDEV installer: $architectureForInstaller"

# Cleanup old installers
Get-ChildItem -Path $env:TEMP -Filter "ddev_windows_*_installer.*.exe" -ErrorAction SilentlyContinue | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force -ErrorAction SilentlyContinue
    } catch {
        # Intentionally silent
    }
}

# Install DDEV on Windows to manipulate the host OS's hosts file.
$GitHubOwner = "ddev"
$RepoName    = "ddev"
# Get the latest release JSON from the GitHub API endpoint.
$apiUrl = "https://api.github.com/repos/$GitHubOwner/$RepoName/releases/latest"
try {
    $response = Invoke-WebRequest -Headers @{ Accept = 'application/json' } -Uri $apiUrl
} catch {
    Write-Error "Could not fetch latest release info from $apiUrl. Details: $_"
    exit 1
}
$json = $response.Content | ConvertFrom-Json
$tagName = $json.tag_name
Write-Host "The latest $GitHubOwner/$RepoName version is $tagName."
# Because the published artifact includes the version in its name, we have to insert $tagName into the filename.
$installerFilename = "ddev_windows_${architectureForInstaller}_installer.${tagName}.exe"
$downloadUrl = "https://github.com/$GitHubOwner/$RepoName/releases/download/$tagName/$installerFilename"
$TempDir = $env:TEMP
$DdevInstallerPath = Join-Path $TempDir ([guid]::NewGuid().ToString() + "_" + $installerFilename)

Write-Host "Downloading from $downloadUrl..."
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $DdevInstallerPath
} catch {
    Write-Error "Could not download the installer from $downloadUrl. Details: $_"
    exit 1
}
Start-Process $DdevInstallerPath -ArgumentList "/S", -Wait
$env:PATH += ";C:\Program Files\DDEV"

Write-Host "DDEV installation complete."

$mkcertPath = "C:\Program Files\DDEV\mkcert.exe"
$maxWait = 60
Write-Host "Waiting up to $maxWait seconds for $mkcertPath binary..."
$waited = 0
while (-not (Test-Path $mkcertPath) -and $waited -lt $maxWait) {
    Start-Sleep -Seconds 1
    $waited++
}
if (-not (Test-Path $mkcertPath)) {
    Write-Error "mkcert.exe did not appear at $mkcertPath after waiting $maxWait seconds"
    exit 1
}

& $mkcertPath -install
$env:CAROOT = & $mkcertPath -CAROOT
setx CAROOT $env:CAROOT; If ($Env:WSLENV -notlike "*CAROOT/up:*") { $env:WSLENV="CAROOT/up:$env:WSLENV"; setx WSLENV $Env:WSLENV }

wsl -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"
wsl -u root apt-get update
wsl -u root apt-get install -y ca-certificates curl gnupg lsb-release
wsl -u root install -m 0755 -d /etc/apt/keyrings
wsl -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"
wsl -u root -e bash -c 'echo deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu  $(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1'

wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io wslu"
wsl -u root -e bash -c "apt-get install -y --no-install-recommends ddev"
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
