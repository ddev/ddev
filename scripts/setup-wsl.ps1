# Copyright (c) Gilbertsoft LLC. All rights reserved.
# Licensed under the MIT License.
<#
.Synopsis
    Setup WSL for the usage with DDEV Local.
.DESCRIPTION
    By default, Chocolatey and mkcert will be installed and setted up. Run this script from a elevated PowerShell console.
.EXAMPLE
    Setup WSL for DDEV Local
    Set-ExecutionPolicy Bypass -Scope Process -Force; .\setup-wsl.ps1
.EXAMPLE
    Invoke this script directly from GitHub
    Set-ExecutionPolicy Bypass -Scope Process -Force; Invoke-Expression "& { $(Invoke-RestMethod 'https://github.com/drud/ddev/raw/master/scripts/setup-wsl.ps1') }"
#>
[CmdletBinding()]
param()

#Requires -Version 5.0
#Requires -RunAsAdministrator

Set-StrictMode -Version 3.0
$ErrorActionPreference = "Stop"

$IsLinuxEnv = (Get-Variable -Name "IsLinux" -ErrorAction Ignore) -and $IsLinux
$IsMacOSEnv = (Get-Variable -Name "IsMacOS" -ErrorAction Ignore) -and $IsMacOS
$IsWinEnv = !$IsLinuxEnv -and !$IsMacOSEnv

if (-not $IsWinEnv) {
    throw "This script is only supported on Windows"
}


<#
.Synopsis
    Sets or appends a value to a environment variable
.DESCRIPTION
    Sets or appends a value to a environment variable.
.Parameter Name
    The name of the variable to change
.Parameter Value
    The value to set.
.Parameter Append
    Keeps the current value and appends the new one to the end separated by the defined separator.
.Parameter Separator
    The separator to use for the append mode.
.Parameter Global
    Write the variable to the global scope instead of the user.
#>
Function Set-EnvironmentVariable {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory, ValueFromPipeline, ValueFromPipelineByPropertyName)]
        [ValidateNotNullOrEmpty()]
        [string] $Name,

        [Parameter(Mandatory, ValueFromPipeline, ValueFromPipelineByPropertyName)]
        [string] $Value,

        [Parameter(ValueFromPipeline, ValueFromPipelineByPropertyName)]
        [switch] $Append,

        [Parameter(ValueFromPipeline, ValueFromPipelineByPropertyName)]
        [string] $Separator = ';',

        [Parameter(ValueFromPipeline, ValueFromPipelineByPropertyName)]
        [switch] $Global
    )

    if (-not $IsWinEnv) {
        return
    }

    if (-not $Global) {
        [string] $Environment = 'Environment'
        [Microsoft.Win32.RegistryKey] $Key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($Environment, [Microsoft.Win32.RegistryKeyPermissionCheck]::ReadWriteSubTree)
    } else {
        [string] $Environment = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
        [Microsoft.Win32.RegistryKey] $Key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey($Environment, [Microsoft.Win32.RegistryKeyPermissionCheck]::ReadWriteSubTree)
    }

    # $key is null here if it the user was unable to get ReadWriteSubTree access.
    if ($null -eq $Key) {
        throw (new-object -typeName 'System.Security.SecurityException' -ArgumentList "Unable to access the target registry")
    }

    # Keep current ValueKind if possible/appropriate
    try {
        [Microsoft.Win32.RegistryValueKind] $ValueKind = $Key.GetValueKind($Name)
    } catch {
        [Microsoft.Win32.RegistryValueKind] $ValueKind = [Microsoft.Win32.RegistryValueKind]::String
    }

    if ($Append) {
        # Get current unexpanded value
        [string] $CurrentUnexpandedValue = $Key.GetValue($Name, '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)

        # Evaluate new value
        $NewValue = [string]::Concat($CurrentUnexpandedValue.TrimEnd($Separator), $Separator, $Value)
    } else {
        $NewValue = $Value
    }

    # Upgrade ValueKind to [Microsoft.Win32.RegistryValueKind]::ExpandString if appropriate
    if ($NewValue.Contains('%')) {
        $ValueKind = [Microsoft.Win32.RegistryValueKind]::ExpandString
    }

    $Key.SetValue($Name, $NewValue, $ValueKind)
}



# Setting Tls to 12 to prevent the Invoke-WebRequest : The request was
# aborted: Could not create SSL/TLS secure channel. error.
$originalValue = [Net.ServicePointManager]::SecurityProtocol
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor 3072 # = [Net.SecurityProtocolType]::Tls12

try {
    # Check if mkcert is installed
    try {
        mkcert -version
    } catch [System.Management.Automation.CommandNotFoundException] {
        # Check if Chocolatey is installed
        try {
            choco
        } catch {
            # Chocolatey was not found, install it now
            #Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
            Invoke-Expression "& { $(Invoke-RestMethod 'https://chocolatey.org/install.ps1') }"
        }

        # Install mkcert
        choco install mkcert -y
    }

    # Setup mkcert
    mkcert -install

    # Set default WSL version
    wsl --set-default-version 2

    # Setup CAROOT variable
    If ($null -eq $Env:CAROOT) {
        Set-EnvironmentVariable -Name "CAROOT" -Value $(mkcert -CAROOT)
        #setx CAROOT "$(mkcert -CAROOT)"
    }

    # Export CAROOT to WSL
    If ($Env:WSLENV -notlike "*CAROOT*") {
        Set-EnvironmentVariable -Name "WSLENV" -Value "CAROOT/up" -Append -Separator ":"
        #setx WSLENV "CAROOT/up:$Env:WSLENV"
    }
} finally {
    # Restore original value
    [Net.ServicePointManager]::SecurityProtocol = $originalValue
}
