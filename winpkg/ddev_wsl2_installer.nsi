!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif

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
  ${NSD_CreateRadioButton} 0 45u 100% 12u "Use Docker Desktop for Windows integration"
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
  DetailPrint "Checking WSL2 version..."
  nsExec::ExecToStack 'wsl.exe -l -v'
  Pop $1  ; error code
  Pop $0  ; output
  DetailPrint "WSL version check output: $0"
  DetailPrint "WSL version check exit code: $1"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
    Abort
  ${EndIf}

  ; Check for Ubuntu-based default distro
  DetailPrint "Checking for Ubuntu-based default distro..."
  nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME="'
  Pop $1  ; First pop is error code
  Pop $0  ; output
  DetailPrint "WSL Output: $0"
  DetailPrint "Exit Code: $1"

  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Could not check your default WSL2 distro. Please ensure WSL is working."
    Abort
  ${EndIf}

  ${If} $0 == ""
    MessageBox MB_ICONSTOP "Could not detect distro name. Please ensure WSL is working."
    Abort
  ${EndIf}

  ; Simple case-insensitive grep for Ubuntu in the output
  nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME= | grep -i ubuntu"'
  Pop $1
  Pop $0
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Your default WSL2 distro is not Ubuntu-based. Please set Ubuntu as your default WSL2 distro."
    Abort
  ${EndIf}

  DetailPrint "Ubuntu-based distro detected successfully."

  ; Check for WSL2 version (must be WSL2, not WSL1)
  DetailPrint "Checking for WSL2..."
  nsExec::ExecToStack 'wsl uname -v'
  Pop $1  ; error code
  Pop $0  ; output
  DetailPrint "WSL kernel version: $0"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Could not check WSL version. Please ensure WSL is working."
    Abort
  ${EndIf}
  ${If} $0 == ""
    MessageBox MB_ICONSTOP "Could not detect WSL version. Please ensure WSL is working."
    Abort
  ${EndIf}
  ${If} $0 == "WSL"
    MessageBox MB_ICONSTOP "Your default WSL distro is not WSL2. Please upgrade to WSL2."
    Abort
  ${EndIf}

  DetailPrint "WSL2 detected successfully."

  ; Check for non-root default user
  DetailPrint "Checking for non-root user..."
  nsExec::ExecToStack 'wsl whoami'
  Pop $1  ; error code
  Pop $0  ; output
  DetailPrint "Current user: $0"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Could not check WSL user. Please ensure WSL is working."
    Abort
  ${EndIf}
  ${If} $0 == "root"
    MessageBox MB_ICONSTOP "Default user in your WSL2 distro is root. Please configure an ordinary default user."
    Abort
  ${EndIf}

  DetailPrint "Non-root user detected successfully."

  ; Download DDEV for Windows
  DetailPrint "Downloading DDEV for Windows v1.24.6..."
  nsExec::ExecToLog 'powershell -Command "Invoke-WebRequest -Uri https://github.com/ddev/ddev/releases/download/v1.24.6/ddev_windows_amd64_installer.v1.24.6.exe -OutFile $TEMP\\ddev_installer.exe"'
  Pop $1  ; error code
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to download DDEV installer. Please check your internet connection."
    Abort
  ${EndIf}

  DetailPrint "Installing DDEV on Windows..."
  DetailPrint "Running: $TEMP\\ddev_installer.exe /S"
  ExecWait '"$TEMP\\ddev_installer.exe" /S' $R0
  DetailPrint "DDEV installer completed with exit code: $R0"
  ${If} $R0 != 0
    MessageBox MB_ICONSTOP "DDEV Windows installation failed with error code $R0"
    Abort
  ${EndIf}

  DetailPrint "DDEV installer completed successfully. Setting up environment..."

  ; Add DDEV to PATH
  ReadRegStr $0 HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "PATH"
  ${If} $0 == ""
    StrCpy $0 "$PROGRAMFILES\\DDEV"
  ${Else}
    StrCpy $0 "$0;$PROGRAMFILES\\DDEV"
  ${EndIf}
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "PATH" $0

  ; Notify Windows of the PATH change
  DetailPrint "Updating system PATH..."
  SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment" /TIMEOUT=5000

  DetailPrint "Looking for existing mkcert.exe..."
  ; Check Program Files first as that's where DDEV installs it
  ${If} ${FileExists} "C:\Program Files\DDEV\mkcert.exe"
    DetailPrint "Found mkcert.exe in C:\Program Files\DDEV"
    StrCpy $3 "C:\Program Files\DDEV\mkcert.exe"
    Goto mkcert_found
  ${EndIf}

  ; Try where command as backup
  nsExec::ExecToStack 'cmd /c where mkcert.exe'
  Pop $1  ; error code
  Pop $0  ; output
  DetailPrint "where mkcert.exe exit code: $1"
  DetailPrint "where mkcert.exe output: $0"
  ${If} $1 == 0
  ${AndIf} $0 != ""
    DetailPrint "Found mkcert.exe in PATH: $0"
    StrCpy $3 "$0"  ; Save path for later
    Goto mkcert_found
  ${EndIf}

  DetailPrint "Checking current user path..."
  ReadEnvStr $0 "PATH"
  DetailPrint "Current PATH: $0"

  MessageBox MB_ICONSTOP "mkcert.exe not found in C:\Program Files\DDEV or PATH. PATH: $0"
  Abort

mkcert_found:
  DetailPrint "Using mkcert.exe from: $3"

  ; Install mkcert root CA
  DetailPrint "Installing mkcert root CA using: $3"
  nsExec::ExecToLog '"$3" -install'
  Pop $1  ; error code
  Pop $0  ; output
  DetailPrint "mkcert -install exit code: $1"
  DetailPrint "mkcert -install output: $0"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "mkcert installation failed. Error code: $1, Output: $0"
    Abort
  ${EndIf}

  DetailPrint "mkcert root CA installed successfully."

  ; Set CAROOT environment variable
  nsExec::ExecToStack '"$3" -CAROOT'
  Pop $1  ; error code
  Pop $0  ; output
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Could not get CAROOT. Please check the logs."
    Abort
  ${EndIf}
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "CAROOT" $0

  ; Set WSLENV for CAROOT
  ReadRegStr $1 HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "WSLENV"
  StrCpy $1 "$1;CAROOT/up"
  WriteRegStr HKLM "SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment" "WSLENV" $1

  StrCmp $DOCKER_OPTION "docker-ce" pre_docker_setup pre_docker_setup

pre_docker_setup:
  ; Remove old Docker versions first
  DetailPrint "Removing old Docker versions if present..."
  nsExec::ExecToStack 'wsl -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"'
  Pop $1
  Pop $0

  ; apt-get upgrade
  DetailPrint "Doing apt-get upgrade..."
  nsExec::ExecToStack 'wsl -u root bash -c "apt-get update && apt-get upgrade -y >/dev/null 2>&1"'
  Pop $1
  Pop $0

  ; Install linux packages
  DetailPrint "Installing linux packages..."
  nsExec::ExecToStack 'wsl -u root apt-get install -y ca-certificates curl gnupg gnupg2 libsecret-1-0 lsb-release pass'
  Pop $1
  Pop $0
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to install dependencies. Please check the logs."
    Abort
  ${EndIf}

  ; Create keyrings directory if it doesn't exist
  DetailPrint "Setting up keyrings directory..."
  nsExec::ExecToStack 'wsl -u root install -m 0755 -d /etc/apt/keyrings'
  Pop $1
  Pop $0

  ; Add Docker GPG key
  DetailPrint "Adding Docker repository key..."
  nsExec::ExecToStack 'wsl -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"'
  Pop $1
  Pop $0
  DetailPrint "Key installation output: $0"
  DetailPrint "Key installation exit code: $1"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to add Docker repository key. Please check your internet connection. Exit code: $1, Output: $0"
    Abort
  ${EndIf}

  ; Add Docker repository
  DetailPrint "Adding Docker repository..."
  nsExec::ExecToStack 'wsl -u root -e bash -c "echo deb [arch=$$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $$(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1"'
  Pop $1
  Pop $0
  DetailPrint "Repository addition output: $0"
  DetailPrint "Repository addition exit code: $1"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to add Docker repository. Exit code: $1, Output: $0"
    Abort
  ${EndIf}

  StrCmp $DOCKER_OPTION "docker-ce" docker_ce docker_desktop

docker_ce:
  DetailPrint "Installing Docker CE and DDEV inside WSL2..."
  Goto common_setup

docker_desktop:
  DetailPrint "Setting up $DOCKER_OPTION..."
  Goto common_setup

common_setup:
  ; Add DDEV GPG key
  DetailPrint "Adding DDEV repository key..."
  nsExec::ExecToStack 'wsl -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
  Pop $1
  Pop $0
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to add DDEV repository key. Error: $0"
    Abort
  ${EndIf}

  ; Add DDEV repository
  DetailPrint "Adding DDEV repository..."
  nsExec::ExecToStack 'wsl -u root -e bash -c "echo \"deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *\" > /etc/apt/sources.list.d/ddev.list"'
  Pop $1
  Pop $0
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to add DDEV repository. Please check the logs."
    Abort
  ${EndIf}

  ; Update package lists
  DetailPrint "Updating package lists..."
  nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1"'
  Pop $1
  Pop $0
  DetailPrint "apt-get update output: $0"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to update package lists. Error: $0"
    Abort
  ${EndIf}

  ; Install packages based on Docker option
  DetailPrint "Installing packages..."
  ${If} $DOCKER_OPTION == "docker-ce"
    StrCpy $0 "ddev docker-ce docker-ce-cli containerd.io wslu"
  ${Else}
    StrCpy $0 "ddev docker-ce-cli wslu"
  ${EndIf}
  nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y $0 2>&1"'
  Pop $1
  Pop $2
  DetailPrint "Installation output: $2"
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "Failed to install packages. Error: $2"
    Abort
  ${EndIf}

  ; Install mkcert root CA in WSL
  nsExec::ExecToStack 'wsl -u root mkcert -install'
  Pop $1
  Pop $0

  ; Remove old .docker config if present
  nsExec::ExecToStack 'wsl rm -rf ~/.docker'
  Pop $1
  Pop $0

  ; Show DDEV version
  DetailPrint "Verifying DDEV installation..."
  nsExec::ExecToStack 'wsl ddev version'
  Pop $1
  Pop $0
  ${If} $1 != 0
    MessageBox MB_ICONSTOP "DDEV verification failed. Please check the logs."
    Abort
  ${EndIf}

  ${If} $DOCKER_OPTION == "docker-desktop"
    DetailPrint "All done! Please ensure Docker Desktop is running with WSL2 integration enabled."
  ${Else}
    DetailPrint "All done! Installation completed successfully."
  ${EndIf}
  Goto done

done:
  DetailPrint "Installation completed successfully."
SectionEnd
