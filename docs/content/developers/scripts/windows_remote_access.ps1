# Sets up SSH and RDP on a Windows test runner for access over Tailscale.
# Run in admin PowerShell. -PublicKey adds maintainers' SSH public keys:
#   .\windows_remote_access.ps1 -PublicKey 'ssh-ed25519 AAAA... user'
param([string[]]$PublicKey = @())

$ErrorActionPreference = "Stop"

$os = (Get-CimInstance Win32_OperatingSystem).Caption
if ($os -match "Home") {
    throw "$os can't host RDP; Pro, Enterprise, or Education is required"
}

Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0 | Out-Null
Set-Service sshd -StartupType Automatic
Start-Service sshd

Set-ItemProperty -Path "HKLM:\System\CurrentControlSet\Control\Terminal Server" -Name fDenyTSConnections -Value 0
Enable-NetFirewallRule -DisplayGroup "Remote Desktop"

# sshd reads administrators' keys from here instead of ~\.ssh\authorized_keys,
# and silently ignores the file if it is UTF-16 or has looser permissions.
$keyFile = "C:\ProgramData\ssh\administrators_authorized_keys"
foreach ($key in $PublicKey) {
    if (-not ((Test-Path $keyFile) -and (Select-String -Path $keyFile -SimpleMatch $key -Quiet))) {
        Add-Content -Path $keyFile -Value $key -Encoding ascii
    }
}
if (Test-Path $keyFile) {
    icacls.exe $keyFile /inheritance:r /grant "Administrators:F" /grant "SYSTEM:F" | Out-Null
}

Get-Service sshd, TermService | Format-Table Name, Status
Get-NetTCPConnection -State Listen -LocalPort 22, 3389 -ErrorAction SilentlyContinue |
    Select-Object -Unique LocalPort | Format-Table @{ Label = "Listening"; Expression = { $_.LocalPort } }
