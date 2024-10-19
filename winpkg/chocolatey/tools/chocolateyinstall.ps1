
$ErrorActionPreference = 'Stop';
$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$url64      = 'https://github.com/REPLACE_GITHUB_ORG/ddev/releases/download/REPLACE_DDEV_VERSION/ddev_windows_amd64_installer.REPLACE_DDEV_VERSION.exe'

$packageArgs = @{
  packageName   = $env:ChocolateyPackageName
  unzipLocation = $toolsDir
  fileType      = 'EXE'
  url64bit      = $url64

  softwareName  = 'ddev*'

  checksum64 = 'REPLACE_INSTALLER_CHECKSUM'
  checksumType64= 'sha256'

  validExitCodes= @(0, 3010, 1641)
  silentArgs   = '/S /C'
}

Install-ChocolateyPackage @packageArgs










    








