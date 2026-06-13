# Icinga/Nagios check for Docker Desktop WSL2 integration on DDEV test machines.
# Returns 0 (OK) or 2 (CRITICAL) with a one-line status on stdout.
#
# Deploy to each Windows test machine and register as an Icinga check.
# The listed distros are the desktop-provider test instances that require
# Docker Desktop WSL2 integration to be enabled.

$distros = @(
    "ddev-test-ubuntu-desktop",
    "ddev-test-ubuntu2404-desktop",
    "ddev-test-debian-desktop"
)

$failed = @()
$ok = @()

foreach ($distro in $distros) {
    $out = wsl.exe -d $distro -- docker ps 2>&1
    if ($LASTEXITCODE -ne 0) {
        $failed += $distro
    } else {
        $ok += $distro
    }
}

if ($failed.Count -gt 0) {
    Write-Output "CRITICAL: Docker Desktop WSL2 integration lost for: $($failed -join ', '). Re-enable: Docker Desktop -> Settings -> Resources -> WSL Integration -> enable each distro -> Apply & Restart."
    exit 2
}

Write-Output "OK: Docker Desktop WSL2 integration active for all distros ($($ok -join ', '))"
exit 0
