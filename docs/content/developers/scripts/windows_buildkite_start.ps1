net localgroup docker-users /add
net localgroup docker-users testbot /add

Set-Timezone -Id "Mountain Standard Time"

# Enable developer mode feature
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock" /t REG_DWORD /f /v "AllowDevelopmentWithoutDevLicense" /d "1"

cmd /c "setx /M PATH ""C:\Program Files\Git\bin;%PATH%"""
