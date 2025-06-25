!include "MUI2.nsh"
!include "LogicLib.nsh"

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif
Var TARGET_ARCH
Var INSTALL_ARCH /* Architecture where installation is happening */

Name "DDEV WSL2 Installer"
OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_wsl2_installer_${TARGET_ARCH}.exe"

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
    MessageBox MB_ICONSTOP "Default user in your WSL2 distro is root. Please configure an ordinary default user."
    Abort

  ; Download and install DDEV for Windows (static version, update as needed)
  DetailPrint "Downloading DDEV for Windows v1.23.4..."
  ExecWait 'powershell -Command "Invoke-WebRequest -Uri https://github.com/ddev/ddev/releases/download/v1.23.4/ddev_windows_amd64_installer.v1.23.4.exe -OutFile $env:TEMP\\ddev_installer.exe; Start-Process $env:TEMP\\ddev_installer.exe -ArgumentList \"/S\" -Wait"'

  ; Add DDEV to PATH
  ReadRegStr $0 HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "PATH"
  StrCpy $0 "$0;$PROGRAMFILES\\DDEV"
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "PATH" $0

  ; Wait for mkcert.exe
  DetailPrint "Waiting for mkcert.exe..."
  Sleep 5000

  ; Install mkcert root CA
  ExecWait '"$PROGRAMFILES\\DDEV\\mkcert.exe" -install'

  ; Set CAROOT environment variable
  nsExec::ExecToStack '"$PROGRAMFILES\\DDEV\\mkcert.exe" -CAROOT'
  Pop $0
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "CAROOT" $0

  ; Set WSLENV for CAROOT
  ReadRegStr $1 HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "WSLENV"
  StrCpy $1 "$1;CAROOT/up"
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "WSLENV" $1

  StrCmp $DOCKER_OPTION "docker-ce" 0 docker_desktop

  ; --- Docker CE inside WSL2 ---
  DetailPrint "Installing Docker CE and DDEV inside WSL2..."
  ; Remove old Docker versions
  ExecWait 'wsl -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"'
  ; Update package lists
  ExecWait 'wsl -u root apt-get update'
  ; Install dependencies
  ExecWait 'wsl -u root apt-get install -y ca-certificates curl gnupg lsb-release'
  ; Create keyrings directory
  ExecWait 'wsl -u root install -m 0755 -d /etc/apt/keyrings'
  ; Add Docker GPG key
  ExecWait 'wsl -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"'
  ; Add Docker repository
  ExecWait 'wsl -u root -e bash -c "echo deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu  $(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1"'
  ; Add DDEV GPG key
  ExecWait 'wsl -u root -e bash -c "rm -f /etc/apt/keyrings/ddev.gpg && curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
  ; Add DDEV repository
  ExecWait 'wsl -u root -e bash -c "echo deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * * > /etc/apt/sources.list.d/ddev.list"'
  ; Install DDEV, Docker CE, and dependencies
  ExecWait 'wsl -u root -e bash -c "apt-get update && apt-get install -y ddev docker-ce docker-ce-cli containerd.io wslu"'
  ; Upgrade packages
  ExecWait 'wsl -u root -e bash -c "apt-get upgrade -y >/dev/null"'
  ; Add user to docker group
  ExecWait 'wsl bash -c "sudo usermod -aG docker $USER"'
  ; Install mkcert root CA in WSL
  ExecWait 'wsl -u root mkcert -install'
  ; Test Docker
  ExecWait 'wsl -e docker ps'
  ; Remove old .docker config if present
  ExecWait 'wsl rm -rf ~/.docker'
  ; Enable systemd in WSL
  ExecWait 'wsl -u root -e bash -c "touch /etc/wsl.conf && if ! fgrep \"[boot]\" /etc/wsl.conf >/dev/null; then printf \"\n[boot]\nsystemd=true\n\" >>/etc/wsl.conf; fi"'
  ; Show DDEV version
  ExecWait 'wsl ddev version'
  Goto done

docker_desktop:
  DetailPrint "Using Docker Desktop for Windows. Please ensure Docker Desktop is installed and WSL2 integration is enabled."
  ; Optionally, check for Docker Desktop and prompt if not found

done:
SectionEnd
