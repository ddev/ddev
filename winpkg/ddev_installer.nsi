!include "MUI2.nsh"
!include "LogicLib.nsh"

Name "DDEV WSL2 Installer"
OutFile "ddev_installer.exe"
InstallDir "$PROGRAMFILES\DDEV"
RequestExecutionLevel admin

Var /GLOBAL DOCKER_OPTION

Page custom DockerChoicePage DockerChoicePageLeave
Page directory
Page instfiles

Function DockerChoicePage
  nsDialogs::Create 1018
  Pop $0
  ${If} $0 == error
    Abort
  ${EndIf}
  ${NSD_CreateLabel} 0 0 100% 24u "Choose Docker integration method for DDEV:"
  Pop $1
  ${NSD_CreateRadioButton} 0 30u 100% 12u "Install Docker CE inside WSL2 (recommended)"
  Pop $2
  ${NSD_CreateRadioButton} 0 45u 100% 12u "Use existing Docker Desktop for Windows"
  Pop $3
  ${NSD_SetState} $2 ${BST_CHECKED}
  nsDialogs::Show
FunctionEnd

Function DockerChoicePageLeave
  ${NSD_GetState} $2 $0
  StrCmp $0 ${BST_CHECKED} 0 +2
    StrCpy $DOCKER_OPTION "docker-ce"
  ${NSD_GetState} $3 $0
  StrCmp $0 ${BST_CHECKED} 0 +2
    StrCpy $DOCKER_OPTION "docker-desktop"
FunctionEnd

Section "Install DDEV and Docker integration"

  ; Check for WSL2
  ExecWait 'powershell -Command "wsl -l -v"' $0
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
    Abort
  ${EndIf}

  ; Check for Ubuntu-based default distro
  ExecWait 'powershell -Command "wsl -e grep ^NAME=.Ubuntu //etc/os-release"' $0
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "Your default WSL2 distro is not Ubuntu-based. Please set Ubuntu as your default WSL2 distro."
    Abort
  ${EndIf}

  ; Check for WSL2 version
  ExecWait 'powershell -Command "wsl -e bash -c \"env | grep WSL_INTEROP=\""' $0
  ${If} $0 != 0
    MessageBox MB_ICONSTOP "Your default WSL distro is not WSL2. Please upgrade to WSL2."
    Abort
  ${EndIf}

  ; Check for non-root default user
  ExecWait 'powershell -Command "wsl -e whoami"' $1
  StrCmp $1 "root" 0 +2
    MessageBox MB_ICONSTOP "Default user in your WSL2 distro is root. Please configure an ordinary default user." & Abort

  ; Download and install DDEV for Windows
  DetailPrint "Downloading latest DDEV for Windows..."
  nsExec::ExecToLog 'powershell -Command "$apiUrl = \"https://api.github.com/repos/ddev/ddev/releases/latest\"; $json = Invoke-WebRequest -Headers @{ Accept = \"application/json\" } -Uri $apiUrl | ConvertFrom-Json; $tagName = $json.tag_name; $arch = if ([System.Environment]::Is64BitOperatingSystem) { \"amd64\" } else { \"arm64\" }; $installer = \"ddev_windows_${arch}_installer.${tagName}.exe\"; $url = \"https://github.com/ddev/ddev/releases/download/$tagName/$installer\"; $out = \"$env:TEMP\\$installer\"; Invoke-WebRequest -Uri $url -OutFile $out; Start-Process $out -ArgumentList \"/S\" -Wait"'

  ; Add DDEV to PATH
  ${EnvVarUpdate} $0 "PATH" "A" "HKLM" "$PROGRAMFILES\DDEV"

  ; Wait for mkcert.exe
  DetailPrint "Waiting for mkcert.exe..."
  Sleep 5000

  ; Install mkcert root CA
  ExecWait '"$PROGRAMFILES\DDEV\mkcert.exe" -install'

  ; Set CAROOT environment variable
  nsExec::ExecToStack '"$PROGRAMFILES\DDEV\mkcert.exe" -CAROOT'
  Pop $0
  ${EnvVarUpdate} $0 "CAROOT" "A" "HKLM" $0

  ; Set WSLENV for CAROOT
  ReadEnvStr $1 "WSLENV"
  StrCpy $1 "$1;CAROOT/up"
  ${EnvVarUpdate} $0 "WSLENV" "A" "HKLM" $1

  StrCmp $DOCKER_OPTION "docker-ce" 0 docker_desktop

  ; --- Docker CE inside WSL2 ---
  DetailPrint "Installing Docker CE and DDEV inside WSL2..."
  ExecWait 'powershell -ExecutionPolicy Bypass -File "$INSTDIR\\scripts\\install_ddev_wsl2_docker_inside.ps1"'
  Goto done

docker_desktop:
  DetailPrint "Using Docker Desktop for Windows. Please ensure Docker Desktop is installed and WSL2 integration is enabled."
  ; Optionally, check for Docker Desktop and prompt if not found

done:
SectionEnd

