!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"
!include "x64.nsh"

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif

Name "DDEV Windows Installer"
OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_windows_${TARGET_ARCH}_installer.exe"

; Use proper Program Files directory for 64-bit applications
InstallDir "$PROGRAMFILES64\DDEV"
RequestExecutionLevel admin

!define PRODUCT_NAME "DDEV"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "DDEV Foundation"

; Variables
Var /GLOBAL INSTALL_OPTION
Var /GLOBAL DOCKER_OPTION
Var StartMenuGroup

!define REG_INSTDIR_ROOT "HKLM"
!define REG_INSTDIR_KEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe"
!define REG_UNINST_ROOT "HKLM"
!define REG_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Installer Types
InstType "Full"
InstType "Simple"
InstType "Minimal"

!define MUI_ICON "graphics\ddev-install.ico"
!define MUI_UNICON "graphics\ddev-uninstall.ico"

!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "graphics\ddev-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "graphics\ddev-wizard.bmp"

!define MUI_ABORTWARNING

; Define pages first
!insertmacro MUI_PAGE_WELCOME

; License page for DDEV
!define MUI_PAGE_CUSTOMFUNCTION_PRE ddevLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE ddevLicLeave
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"

; Custom install type selection
Page custom InstallChoicePage InstallChoicePageLeave

; Directory page
!define MUI_PAGE_CUSTOMFUNCTION_PRE DirectoryPre
!insertmacro MUI_PAGE_DIRECTORY

; Start menu page
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "${PRODUCT_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT ${REG_UNINST_ROOT}
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINST_KEY}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "StartMenuGroup"
!insertmacro MUI_PAGE_STARTMENU Application $StartMenuGroup

; Installation page
!insertmacro MUI_PAGE_INSTFILES

; Finish page
!define MUI_FINISHPAGE_SHOWREADME "https://github.com/ddev/ddev/releases/tag/${VERSION}"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Review the release notes"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_INSTFILES

; Language - must come after pages
!insertmacro MUI_LANGUAGE "English"

Function InstallChoicePage
    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 36u "Choose your preferred DDEV installation type:$\n(You can change Docker provider later using ddev config global)"
    Pop $1

    ${NSD_CreateRadioButton} 10 40u 98% 24u "WSL2 with Docker CE (Recommended)$\nInstalls Docker CE inside WSL2 for best performance"
    Pop $2

    ${NSD_CreateRadioButton} 10 70u 98% 24u "WSL2 with Docker Desktop$\nUse existing Docker Desktop with WSL2 backend"
    Pop $3

    ${NSD_CreateRadioButton} 10 100u 98% 24u "Traditional Windows$\nClassic Windows installation without WSL2"
    Pop $4

    ${NSD_SetState} $2 ${BST_CHECKED}
    nsDialogs::Show
FunctionEnd

Function InstallChoicePageLeave
  ${NSD_GetState} $2 $0
  StrCmp $0 ${BST_CHECKED} 0 +2
    StrCpy $INSTALL_OPTION "wsl2-docker-ce"

  ${NSD_GetState} $3 $0
  StrCmp $0 ${BST_CHECKED} 0 +2
    StrCpy $INSTALL_OPTION "wsl2-docker-desktop"

  ${NSD_GetState} $4 $0
  StrCmp $0 ${BST_CHECKED} 0 +2
    StrCpy $INSTALL_OPTION "traditional"
FunctionEnd

Section "-Initialize"
    ; Create the installation directory
    CreateDirectory "$INSTDIR"
SectionEnd

SectionGroup /e "${PRODUCT_NAME}"
    Section "${PRODUCT_NAME}" SecDDEV
        SectionIn 1 2 3 RO

        SetOutPath "$INSTDIR"
        SetOverwrite on

        ; Install core files for all installation types
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev.exe"
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"
        File /oname=license.txt "..\LICENSE"

        ; Install icons
        SetOutPath "$INSTDIR\Icons"
        SetOverwrite try
        File /oname=ddev.ico "graphics\ddev-install.ico"

        ${If} $INSTALL_OPTION == "traditional"
            Call InstallTraditionalWindows
        ${Else}
            Call CheckWSL2Requirements
            ${If} $INSTALL_OPTION == "wsl2-docker-ce"
                Call InstallWSL2DockerCE
            ${Else}
                Call InstallWSL2DockerDesktop
            ${EndIf}
        ${EndIf}

        ; Create common shortcuts
        !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
        CreateDirectory "$SMPROGRAMS\$StartMenuGroup"
        CreateShortCut "$SMPROGRAMS\$StartMenuGroup\DDEV.lnk" "$INSTDIR\ddev.exe" "" "$INSTDIR\Icons\ddev.ico"
        !insertmacro MUI_STARTMENU_WRITE_END
    SectionEnd

    Section "Add to PATH" SecAddToPath
        SectionIn 1 2 3
        EnVar::SetHKLM
        EnVar::AddValue "Path" "$INSTDIR"
    SectionEnd
SectionGroupEnd

SectionGroup /e "mkcert"
    Section "mkcert" SecMkcert
        SectionIn 1 2
        SetOutPath "$INSTDIR"
        SetOverwrite try

        ; Copy mkcert files
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"

        ; Install icons
        SetOutPath "$INSTDIR\Icons"
        SetOverwrite try
        File /oname=ca-install.ico "graphics\ca-install.ico"
        File /oname=ca-uninstall.ico "graphics\ca-uninstall.ico"

        ; Create shortcuts
        CreateShortcut "$INSTDIR\mkcert install.lnk" "$INSTDIR\mkcert.exe" "-install" "$INSTDIR\Icons\ca-install.ico"
        CreateShortcut "$INSTDIR\mkcert uninstall.lnk" "$INSTDIR\mkcert.exe" "-uninstall" "$INSTDIR\Icons\ca-uninstall.ico"

        !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
        CreateDirectory "$SMPROGRAMS\$StartMenuGroup\mkcert"
        CreateShortCut "$SMPROGRAMS\$StartMenuGroup\mkcert\mkcert install trusted https.lnk" "$INSTDIR\mkcert.exe" "-install" "$INSTDIR\Icons\ca-install.ico"
        CreateShortCut "$SMPROGRAMS\$StartMenuGroup\mkcert\mkcert uninstall trusted https.lnk" "$INSTDIR\mkcert.exe" "-uninstall" "$INSTDIR\Icons\ca-uninstall.ico"
        !insertmacro MUI_STARTMENU_WRITE_END
    SectionEnd

    Section "Setup mkcert" SecMkcertSetup
        SectionIn 1 2
        MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to enable trusted https. Please accept the mkcert dialog box that may follow."
        nsExec::ExecToLog '"$INSTDIR\mkcert.exe" -install'
        Pop $R0
        ${If} $R0 = 0
            WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup" 1
        ${EndIf}
    SectionEnd
SectionGroupEnd

Section -Post
    WriteUninstaller "$INSTDIR\ddev_uninstall.exe"

    ; Remember install directory for updates
    WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "" "$INSTDIR\ddev.exe"
    WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "Path" "$INSTDIR"

    ; Write uninstaller keys
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayName" "$(^Name)"
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "UninstallString" "$INSTDIR\ddev_uninstall.exe"
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayIcon" "$INSTDIR\Icons\ddev.ico"
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
    WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NoModify" 1
    WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NoRepair" 1

    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
    CreateShortCut "$SMPROGRAMS\$StartMenuGroup\Uninstall ${PRODUCT_NAME}.lnk" "$INSTDIR\ddev_uninstall.exe"
    !insertmacro MUI_STARTMENU_WRITE_END
SectionEnd

Function CheckWSL2Requirements
    DetailPrint "Checking WSL2 requirements..."

    ; Check if WSL is installed
    nsExec::ExecToLog 'wsl --status'
    Pop $0
    ${If} $0 != 0
        MessageBox MB_ICONSTOP|MB_OK "WSL2 is not installed. Please install WSL2 first by running 'wsl --install' in an administrative PowerShell window."
        Abort "WSL2 not installed"
    ${EndIf}

    ; Check WSL2 version
    nsExec::ExecToLog 'wsl --set-default-version 2'
    Pop $0
    ${If} $0 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to set WSL2 as default. Please ensure WSL2 is properly installed."
        Abort "WSL2 setup failed"
    ${EndIf}

    DetailPrint "WSL2 requirements satisfied."
FunctionEnd

Function InstallWSL2DockerCE
    DetailPrint "DEBUG: Starting InstallWSL2DockerCE"

    ; Check for WSL2
    DetailPrint "Checking WSL2 version..."
    nsExec::ExecToStack 'wsl.exe -l -v'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL version check output: $0"
    DetailPrint "WSL version check exit code: $1"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
        Abort
    ${EndIf}

    ; Check for Ubuntu-based default distro
    DetailPrint "Checking for Ubuntu-based default distro..."
    nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME="'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL Output: $0"
    DetailPrint "Exit Code: $1"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Could not check your default WSL2 distro. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == ""
        MessageBox MB_ICONSTOP|MB_OK "Could not detect distro name. Please ensure WSL is working."
        Abort
    ${EndIf}
    nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME= | grep -i ubuntu"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Your default WSL2 distro is not Ubuntu-based. Please set Ubuntu as your default WSL2 distro."
        Abort
    ${EndIf}
    DetailPrint "Ubuntu-based distro detected successfully."

    ; Check for WSL2 kernel
    DetailPrint "Checking for WSL2..."
    nsExec::ExecToStack 'wsl uname -v'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL kernel version: $0"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Could not check WSL version. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == ""
        MessageBox MB_ICONSTOP|MB_OK "Could not detect WSL version. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == "WSL"
        MessageBox MB_ICONSTOP|MB_OK "Your default WSL distro is not WSL2. Please upgrade to WSL2."
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
        MessageBox MB_ICONSTOP|MB_OK "Could not check WSL user. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == "root"
        MessageBox MB_ICONSTOP|MB_OK "Default user in your WSL2 distro is root. Please configure an ordinary default user."
        Abort
    ${EndIf}
    DetailPrint "Non-root user detected successfully."

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
        MessageBox MB_ICONSTOP|MB_OK "Failed to install dependencies. Please check the logs."
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
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add Docker repository key. Please check your internet connection. Exit code: $1, Output: $0"
        Abort
    ${EndIf}

    ; Add Docker repository
    DetailPrint "Adding Docker repository..."
    nsExec::ExecToStack 'wsl -u root -e bash -c "echo deb [arch=$$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $$(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add Docker repository. Exit code: $1, Output: $0"
        Abort
    ${EndIf}

    ; --- Common setup (DDEV repo, apt update, install) ---
    ; Add DDEV GPG key
    DetailPrint "Adding DDEV repository key..."
    nsExec::ExecToStack 'wsl -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add DDEV repository key. Error: $0"
        Abort
    ${EndIf}

    ; Add DDEV repository
    DetailPrint "Adding DDEV repository..."
    nsExec::ExecToStack 'wsl -u root -e bash -c "echo \"deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *\" > /etc/apt/sources.list.d/ddev.list"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add DDEV repository. Please check the logs."
        Abort
    ${EndIf}

    ; Update package lists
    DetailPrint "Updating package lists..."
    nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to update package lists. Error: $0"
        Abort
    ${EndIf}

    ; Install packages for Docker CE
    DetailPrint "Installing packages..."
    StrCpy $0 "ddev docker-ce docker-ce-cli containerd.io wslu"
    nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y $0 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to install packages. Error: $2"
        Abort
    ${EndIf}

    ; Detect default user in WSL2 (use wsl whoami)
    DetailPrint "Detecting default user in WSL2..."
    nsExec::ExecToStack 'wsl whoami'
    Pop $1
    Pop $0
    DetailPrint "whoami output: $0"
    ; Remove any trailing newline or carriage return
    Push $0
    Call TrimNewline
    Pop $9
    DetailPrint "Default user detected: $9"

    ; Add user to docker group using root (no sudo)
    DetailPrint "Adding user $9 to docker group with root..."
    nsExec::ExecToStack 'wsl -u root bash -c "usermod -aG docker $9"'
    Pop $1
    Pop $0
    DetailPrint "usermod output: $0"

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
        MessageBox MB_ICONSTOP|MB_OK "DDEV verification failed. Please check the logs."
        Abort
    ${EndIf}

    DetailPrint "All done! Installation completed successfully."
    MessageBox MB_ICONINFORMATION|MB_OK "DDEV WSL2 Docker CE installation completed successfully."
FunctionEnd

Function InstallWSL2DockerDesktop
    DetailPrint "DEBUG: Starting InstallWSL2DockerDesktop"

    ; Check for WSL2
    DetailPrint "Checking WSL2 version..."
    nsExec::ExecToStack 'wsl.exe -l -v'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL version check output: $0"
    DetailPrint "WSL version check exit code: $1"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
        Abort
    ${EndIf}

    ; Check for Ubuntu-based default distro
    DetailPrint "Checking for Ubuntu-based default distro..."
    nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME="'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL Output: $0"
    DetailPrint "Exit Code: $1"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Could not check your default WSL2 distro. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == ""
        MessageBox MB_ICONSTOP|MB_OK "Could not detect distro name. Please ensure WSL is working."
        Abort
    ${EndIf}
    nsExec::ExecToStack 'wsl bash -c "cat /etc/os-release | grep -i ^NAME= | grep -i ubuntu"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Your default WSL2 distro is not Ubuntu-based. Please set Ubuntu as your default WSL2 distro."
        Abort
    ${EndIf}
    DetailPrint "Ubuntu-based distro detected successfully."

    ; Check for WSL2 kernel
    DetailPrint "Checking for WSL2..."
    nsExec::ExecToStack 'wsl uname -v'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "WSL kernel version: $0"
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Could not check WSL version. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == ""
        MessageBox MB_ICONSTOP|MB_OK "Could not detect WSL version. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == "WSL"
        MessageBox MB_ICONSTOP|MB_OK "Your default WSL distro is not WSL2. Please upgrade to WSL2."
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
        MessageBox MB_ICONSTOP|MB_OK "Could not check WSL user. Please ensure WSL is working."
        Abort
    ${EndIf}
    ${If} $0 == "root"
        MessageBox MB_ICONSTOP|MB_OK "Default user in your WSL2 distro is root. Please configure an ordinary default user."
        Abort
    ${EndIf}
    DetailPrint "Non-root user detected successfully."

    ; Add DDEV GPG key
    DetailPrint "Adding DDEV repository key..."
    nsExec::ExecToStack 'wsl -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add DDEV repository key. Error: $0"
        Abort
    ${EndIf}

    ; Add DDEV repository
    DetailPrint "Adding DDEV repository..."
    nsExec::ExecToStack 'wsl -u root -e bash -c "echo \"deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *\" > /etc/apt/sources.list.d/ddev.list"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to add DDEV repository. Please check the logs."
        Abort
    ${EndIf}

    ; Update package lists
    DetailPrint "Updating package lists..."
    nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to update package lists. Error: $0"
        Abort
    ${EndIf}

    ; Install packages for Docker Desktop (no docker-ce, only docker-ce-cli and wslu)
    DetailPrint "Installing packages..."
    StrCpy $0 "ddev docker-ce-cli wslu"
    nsExec::ExecToStack 'wsl -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y $0 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to install packages. Error: $2"
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
        MessageBox MB_ICONSTOP|MB_OK "DDEV verification failed. Please check the logs."
        Abort
    ${EndIf}

    DetailPrint "All done! Installation completed successfully."
    MessageBox MB_ICONINFORMATION|MB_OK "DDEV WSL2 Docker Desktop installation completed successfully."
FunctionEnd

Function InstallTraditionalWindows
    DetailPrint "DEBUG: Starting InstallTraditionalWindows"

    SetOutPath $INSTDIR
    SetOverwrite on

    ; Copy core files
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev.exe"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"
    File /oname=license.txt "..\LICENSE"

    ; Install icons
    SetOutPath "$INSTDIR\Icons"
    SetOverwrite try
    File /oname=ddev.ico "graphics\ddev-install.ico"

    ; Create shortcuts
    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
    CreateDirectory "$INSTDIR\Links"
    CreateDirectory "$SMPROGRAMS\$StartMenuGroup"

    WriteIniStr "$INSTDIR\Links\${PRODUCT_WEB_SITE}.url" "InternetShortcut" "URL" "${PRODUCT_WEB_SITE_URL}"
    CreateShortCut "$SMPROGRAMS\$StartMenuGroup\${PRODUCT_WEB_SITE}.lnk" "$INSTDIR\Links\${PRODUCT_WEB_SITE}.url" "" "$INSTDIR\Icons\ddev.ico"

    WriteIniStr "$INSTDIR\Links\${PRODUCT_DOCUMENTATION}.url" "InternetShortcut" "URL" "${PRODUCT_DOCUMENTATION_URL}"
    CreateShortCut "$SMPROGRAMS\$StartMenuGroup\${PRODUCT_DOCUMENTATION}.lnk" "$INSTDIR\Links\${PRODUCT_DOCUMENTATION}.url" "" "$INSTDIR\Icons\ddev.ico"

    !insertmacro MUI_STARTMENU_WRITE_END

    DetailPrint "Traditional Windows installation completed."
    MessageBox MB_ICONINFORMATION|MB_OK "DDEV Traditional Windows installation completed successfully."
FunctionEnd

Function un.onInit
  MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Are you sure you want to completely remove $(^Name) and all of its components?" IDYES DoUninstall
  Abort

DoUninstall:
  ; Switch to 64 bit view and disable FS redirection
  SetRegView 64
  ${DisableX64FSRedirection}
FunctionEnd

Function DirectoryPre
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
    ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Skip directory selection for WSL2 installs
        Abort
    ${EndIf}
FunctionEnd

Function ddevLicPre
    ReadRegDWORD $R0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:ddevLicenseAccepted"
    ${If} $R0 = 1
        Abort
    ${EndIf}
FunctionEnd

Function ddevLicLeave
    WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:ddevLicenseAccepted" 0x00000001
FunctionEnd

Function .onInit
    ; Set proper 64-bit handling
    SetRegView 64
    ${DisableX64FSRedirection}

    ; Initialize directory to proper Program Files location
    ${If} ${RunningX64}
        StrCpy $INSTDIR "$PROGRAMFILES64\${PRODUCT_NAME}"
    ${Else}
        MessageBox MB_ICONSTOP|MB_OK "This installer is for 64-bit Windows only."
        Abort
    ${EndIf}
FunctionEnd

; Helper: returns "1" if $R0 contains $R1, else ""
Function StrContains
    Exch $R1 ; substring
    Exch
    Exch $R0 ; string
    Push $R2
    StrCpy $R2 ""
    ${DoWhile} $R0 != ""
        StrCpy $R2 $R0 6
        StrCmp $R2 $R1 0 found
            Push "1"
            Goto done
        found:
        StrCpy $R0 $R0 "" 1
    ${Loop}
    Push ""
done:
    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd

; Helper: Trim leading spaces from a string
Function TrimLeft
    Exch $R0
    Push $R1
    StrCpy $R1 0
    loop_trimleft:
        StrCpy $R2 $R0 1 $R1
        StrCmp $R2 " " trimmed
        StrCmp $R2 "" done_trimleft
        IntOp $R1 $R1 + 1
        Goto loop_trimleft
    trimmed:
        StrCpy $R0 $R0 "" $R1
    done_trimleft:
        Pop $R1
        Exch $R0
FunctionEnd

; Helper: Trim trailing newline and carriage return from a string
Function TrimNewline
    Exch $R0
    Push $R1
    StrCpy $R1 $R0 -1
    loop_trimnl:
        StrCpy $R1 $R0 -1
        StrCpy $R2 $R1 1 -1
        ${If} $R2 == "$\n"
            StrCpy $R0 $R0 -1
            Goto loop_trimnl
        ${EndIf}
        ${If} $R2 == "$\r"
            StrCpy $R0 $R0 -1
            Goto loop_trimnl
        ${EndIf}
    Pop $R1
    Exch $R0
FunctionEnd
