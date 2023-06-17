
# Chocolatey
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))

# Install required items using chocolatey
choco install -y composer ddev docker-desktop git jq  mysql-cli golang GoogleChrome make mkcert netcat nodejs nssm zip

net localgroup docker-users /add
net localgroup docker-users testbot /add

Set-Timezone -Id "Mountain Standard Time"

# Enable developer mode feature
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock" /t REG_DWORD /f /v "AllowDevelopmentWithoutDevLicense" /d "1"

cmd /c "setx /M PATH ""C:\Program Files\Git\bin;%PATH%"""


