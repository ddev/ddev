# This PowerShell script tries to do almost all the things required to set up
# an Ubuntu WSL2 instance for use with DDEV and Docker Desktop.
# It requires that an Ubuntu wsl2 distro be installed already, preferably with `wsl --install`, but it can also be
# done manually.
# It requires that Docker Desktop is installed and running, and that it has integration enabled with the Ubuntu
# distro, which is the default behavior.
# You can download, inspect, and run this, or run it directly with
# Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
# iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/ddev/ddev/main/scripts/install_ddev_wsl2_docker_desktop.ps1'))

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
if (-not(Get-Command docker 2>&1 ) -Or -Not(docker ps ) ) {
    throw "\n\ndocker does not seem to be installed yet, or Docker Desktop is not running. Please install it or start it."
}

if (-not(wsl -e docker ps) ) {
    throw "Docker Desktop integration with the default distro does not seem to be enabled yet."
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

# Install DDEV on Windows to manipulate the host OS's hosts file.
$GitHubOwner = "ddev"
$RepoName    = "ddev"
# Get the latest release JSON from the GitHub API endpoint.

# Delete existing old installers
Get-ChildItem -Path $env:TEMP -Filter "ddev_windows_*_installer.*.exe" -ErrorAction SilentlyContinue | ForEach-Object {
    try {
        Remove-Item $_.FullName -Force -ErrorAction Stop
    } catch {
        Write-Warning "Could not delete old installer file $($_.FullName): $_"
    }
}

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

wsl -u root -e bash -c "apt-get update && apt-get install -y curl"
wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"
wsl -u root -e bash -c 'echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ \* \* > /etc/apt/sources.list.d/ddev.list'
wsl -u root -e bash -c "apt-get update && apt-get install -y wslu"
wsl -u root -e bash -c "apt-get install -y --no-install-recommends ddev"
wsl -u root -e bash -c "apt-get upgrade -y >/dev/null"


wsl bash -c 'echo $CAROOT'
wsl -u root mkcert -install
if (-not(wsl -e docker ps)) {
    throw "docker does not seem to be working inside the WSL2 distro yet. Check Resources->WSL Integration in Docker Desktop"
}
wsl -u root -e bash -c "touch /etc/wsl.conf && if ! fgrep '[boot]' /etc/wsl.conf >/dev/null; then printf '\n[boot]\nsystemd=true\n' >>/etc/wsl.conf; fi"

wsl ddev version
