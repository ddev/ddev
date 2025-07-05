!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"
!include "x64.nsh"
!include "WordFunc.nsh"

!insertmacro GetParameters
!insertmacro GetOptions

!insertmacro WordFind
; Remove the Trim macro since we're using our own TrimWhitespace function

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif

Name "DDEV"
OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_windows_${TARGET_ARCH}_installer.exe"

; Use proper Program Files directory for 64-bit applications
InstallDir "$PROGRAMFILES64\DDEV"
RequestExecutionLevel admin

!define PRODUCT_NAME "DDEV"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "DDEV Foundation"

; Variables
Var /GLOBAL INSTALL_OPTION
Var /GLOBAL SELECTED_DISTRO
Var /GLOBAL SILENT_INSTALL_TYPE
Var /GLOBAL SILENT_DISTRO
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

; Function declarations - must be before page definitions
Function DistroSelectionPage
    DetailPrint "Starting DistroSelectionPage..."
    ${If} $INSTALL_OPTION != "wsl2-docker-ce"
    ${AndIf} $INSTALL_OPTION != "wsl2-docker-desktop"
        DetailPrint "Skipping distro selection for non-WSL2 install"
        Abort
    ${EndIf}

    ; Skip this page if distro was specified via command line
    ${If} $SILENT_DISTRO != ""
        DetailPrint "Skipping distro selection - using command line distro: $SILENT_DISTRO"
        StrCpy $SELECTED_DISTRO $SILENT_DISTRO
        Abort
    ${EndIf}

    DetailPrint "Creating dialog..."
    nsDialogs::Create 1018
    Pop $0
    DetailPrint "Dialog create result: $0"
    ${If} $0 == error
        DetailPrint "Failed to create dialog"
        Abort
    ${EndIf}

    ; Get Ubuntu distros before creating any controls
    Call GetUbuntuDistros
    Pop $R0
    DetailPrint "Got distros: [$R0]"
    ${If} $R0 == ""
        DetailPrint "ERROR: No Ubuntu-based WSL2 distributions found"
        Push "No Ubuntu-based WSL2 distributions found. Please install Ubuntu for WSL2 first."
        Call ShowErrorAndAbort
    ${EndIf}

    DetailPrint "Creating label..."
    ${NSD_CreateLabel} 0 0 100% 24u "Select your Ubuntu-based WSL2 distribution:"
    Pop $1

    DetailPrint "Creating radio buttons..."

    ; Process the pipe-separated list and create radio buttons
    StrCpy $R1 $R0    ; Working copy of the list
    StrCpy $R2 0      ; Item count
    StrCpy $R3 0      ; Y position counter

    ${Do}
        ; Find position of next pipe or end
        StrCpy $R4 1   ; Length to extract
        StrCpy $R5 0   ; Position
        ${Do}
            StrCpy $R6 $R1 1 $R5  ; Get character at position
            ${If} $R6 == "|"
            ${OrIf} $R6 == ""
                ${Break}
            ${EndIf}
            IntOp $R5 $R5 + 1
        ${Loop}

        ; Extract the item
        ${If} $R5 > 0
            StrCpy $R7 $R1 $R5    ; Extract item
            DetailPrint "Adding radio button: [$R7]"
            
            ; Calculate Y position for radio button
            IntOp $R8 $R3 * 24
            IntOp $R8 $R8 + 30
            
            ; Create radio button
            ${NSD_CreateRadioButton} 10 $R8u 280u 16u "$R7"
            Pop $R9
            
            ; Store radio button handle based on item count
            ${If} $R2 == 0
                StrCpy $2 $R9  ; Store first radio button handle
                ${NSD_SetState} $R9 ${BST_CHECKED}  ; Select first item by default
            ${ElseIf} $R2 == 1
                StrCpy $3 $R9  ; Store second radio button handle
            ${ElseIf} $R2 == 2
                StrCpy $4 $R9  ; Store third radio button handle
            ${ElseIf} $R2 == 3
                StrCpy $5 $R9  ; Store fourth radio button handle
            ${ElseIf} $R2 == 4
                StrCpy $6 $R9  ; Store fifth radio button handle
            ${EndIf}
            
            IntOp $R2 $R2 + 1
            IntOp $R3 $R3 + 1
        ${EndIf}

        ; Move past the separator
        IntOp $R5 $R5 + 1
        StrCpy $R1 $R1 "" $R5

        ; Check if we're done
        ${If} $R1 == ""
            ${Break}
        ${EndIf}
    ${Loop}

    DetailPrint "Added $R2 radio buttons"

    DetailPrint "About to show dialog..."
    nsDialogs::Show
FunctionEnd

Function DistroSelectionPageLeave
    DetailPrint "Getting selected distro..."
    
    ; Check which radio button is selected and get its text
    ${NSD_GetState} $2 $R0
    ${If} $R0 == ${BST_CHECKED}
        ${NSD_GetText} $2 $SELECTED_DISTRO
        DetailPrint "Selected distro: $SELECTED_DISTRO"
        Return
    ${EndIf}
    
    ${NSD_GetState} $3 $R0
    ${If} $R0 == ${BST_CHECKED}
        ${NSD_GetText} $3 $SELECTED_DISTRO
        DetailPrint "Selected distro: $SELECTED_DISTRO"
        Return
    ${EndIf}
    
    ${NSD_GetState} $4 $R0
    ${If} $R0 == ${BST_CHECKED}
        ${NSD_GetText} $4 $SELECTED_DISTRO
        DetailPrint "Selected distro: $SELECTED_DISTRO"
        Return
    ${EndIf}
    
    ${NSD_GetState} $5 $R0
    ${If} $R0 == ${BST_CHECKED}
        ${NSD_GetText} $5 $SELECTED_DISTRO
        DetailPrint "Selected distro: $SELECTED_DISTRO"
        Return
    ${EndIf}
    
    ${NSD_GetState} $6 $R0
    ${If} $R0 == ${BST_CHECKED}
        ${NSD_GetText} $6 $SELECTED_DISTRO
        DetailPrint "Selected distro: $SELECTED_DISTRO"
        Return
    ${EndIf}
    
    ; Fallback - should not happen if we have proper radio button logic
    DetailPrint "No distro selected - using first available"
    ${NSD_GetText} $2 $SELECTED_DISTRO
FunctionEnd

; Define pages
!insertmacro MUI_PAGE_WELCOME

; License page for DDEV
!define MUI_PAGE_CUSTOMFUNCTION_PRE ddevLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE ddevLicLeave
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"

; Custom install type selection
Page custom InstallChoicePage InstallChoicePageLeave

; Add WSL2 distro selection page
Page custom DistroSelectionPage DistroSelectionPageLeave

; Directory page
!define MUI_DIRECTORYPAGE_TEXT_TOP "DDEV Windows-side components will be installed in this folder."
!define MUI_DIRECTORYPAGE_TEXT_DESTINATION "Windows install folder:"
!define MUI_DIRECTORYPAGE_HEADER_TEXT "Choose Windows install folder"
!insertmacro MUI_PAGE_DIRECTORY

; Start menu page
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "${PRODUCT_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT ${REG_UNINST_ROOT}
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINST_KEY}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "StartMenuGroup"
!define MUI_PAGE_CUSTOMFUNCTION_PRE StartMenuPagePre
!insertmacro MUI_PAGE_STARTMENU Application $StartMenuGroup

; Installation page
!insertmacro MUI_PAGE_INSTFILES

; Finish page
; TODO: is this useful? How about just linking to 'releases'
!define MUI_FINISHPAGE_SHOWREADME "https://github.com/ddev/ddev/releases/tag/${VERSION}"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Review the release notes"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_INSTFILES

; Language - must come after pages
!insertmacro MUI_LANGUAGE "English"

; Reserve plugin files for faster startup
ReserveFile /plugin EnVar.dll

Function InstallChoicePage
    ; Skip this page if install type was specified via command line
    ${If} $SILENT_INSTALL_TYPE != ""
        Abort
    ${EndIf}

    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 36u "Choose DDEV installation type:"
    Pop $1

    ${NSD_CreateRadioButton} 10 40u 98% 24u "WSL2 with Docker CE (Recommended)$\nInstalls Docker CE inside WSL2 for best performance"
    Pop $2

    ${NSD_CreateRadioButton} 10 70u 98% 24u "WSL2 with Docker Desktop or Rancher Desktop$\nRequires working Windows-installed Docker provider like Docker Desktop or Rancher Desktop with WSL2 backend"
    Pop $3

    ${NSD_CreateRadioButton} 10 100u 98% 24u "Traditional Windows$\nClassic Windows installation using Git Bash, PowerShell, or Cmd (Requires a Windows Docker provider like Docker Desktop or Rancher Desktop)"
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

Function StartMenuPagePre
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
    ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        Abort ; Skip the start menu page for WSL2 installations
    ${EndIf}
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

        ; Install ddev-hostname.exe & mkcert.exe for all installation types
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
        File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"
        File /oname=license.txt "..\LICENSE"

        ; Install Linux DDEV binaries to temp directory for WSL2 installations
        ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
            SetOutPath "C:\Windows\Temp\ddev_installer"
            File /oname=ddev_linux "..\.gotmp\bin\linux_${TARGET_ARCH}\ddev"
            File /oname=ddev-hostname_linux "..\.gotmp\bin\linux_${TARGET_ARCH}\ddev-hostname"
        ${EndIf}

        ; Install icons
        SetOutPath "$INSTDIR\Icons"
        SetOverwrite try
        File /oname=ddev.ico "graphics\ddev-install.ico"

        ; Run mkcert.exe -install early for all installation types (needed for WSL2 setup)
        Call RunMkcertInstall

        ; Add DDEV installation directory to PATH (EnVar::AddValue handles duplicates)
        DetailPrint "Adding DDEV installation directory to system PATH..."
        ReadRegStr $R0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
        DetailPrint "PATH before addition: $R0"
        
        EnVar::SetHKLM
        EnVar::AddValue "Path" "$INSTDIR"
        Pop $R1
        DetailPrint "EnVar::AddValue result: $R1"
        
        DetailPrint "PATH addition completed with result: $R1"

        ${If} $INSTALL_OPTION == "traditional"
            Call InstallTraditionalWindows
        ${Else}
            Call InstallWSL2Common
        ${EndIf}

        ; Create shortcuts only for traditional install
        ${If} $INSTALL_OPTION == "traditional"
            !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
            CreateDirectory "$SMPROGRAMS\$StartMenuGroup"
            CreateShortCut "$SMPROGRAMS\$StartMenuGroup\DDEV.lnk" "$INSTDIR\ddev.exe" "" "$INSTDIR\Icons\ddev.ico"
            !insertmacro MUI_STARTMENU_WRITE_END
        ${EndIf}
    SectionEnd

SectionGroupEnd

Section -Post
    WriteUninstaller "$INSTDIR\ddev_uninstall.exe"

    ; Remember install directory for updates
    WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "" "$INSTDIR\ddev.exe"
    WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "Path" "$INSTDIR"

    ; Write uninstaller keys with correct product name
    WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
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

Section Uninstall
    ; Uninstall mkcert if it was installed
    Call un.mkcertUninstall

    ; Clean up mkcert environment variables
    Call un.CleanupMkcertEnvironment

    ; Remove install directory from system PATH
    EnVar::SetHKLM
    EnVar::DeleteValue "Path" "$INSTDIR"

    ; Remove all installed files
    Delete "$INSTDIR\ddev.exe"
    Delete "$INSTDIR\ddev-hostname.exe"
    Delete "$INSTDIR\mkcert.exe"
    Delete "$INSTDIR\mkcert_license.txt"
    Delete "$INSTDIR\license.txt"
    Delete "$INSTDIR\mkcert install.lnk"
    Delete "$INSTDIR\mkcert uninstall.lnk"
    Delete "$INSTDIR\ddev_uninstall.exe"

    ; Remove icons and links directories
    RMDir /r "$INSTDIR\Icons"
    RMDir /r "$INSTDIR\Links"

    ; Remove all installed shortcuts if they exist
    !insertmacro MUI_STARTMENU_GETFOLDER "Application" $StartMenuGroup
    ${If} "$StartMenuGroup" != ""
        Delete "$SMPROGRAMS\$StartMenuGroup\DDEV.lnk"
        Delete "$SMPROGRAMS\$StartMenuGroup\DDEV Website.lnk"
        Delete "$SMPROGRAMS\$StartMenuGroup\DDEV Documentation.lnk"
        Delete "$SMPROGRAMS\$StartMenuGroup\Uninstall ${PRODUCT_NAME}.lnk"
        RMDir /r "$SMPROGRAMS\$StartMenuGroup\mkcert"
        RMDir "$SMPROGRAMS\$StartMenuGroup"
    ${EndIf}

    ; Remove registry keys
    DeleteRegKey ${REG_UNINST_ROOT} "${REG_UNINST_KEY}"
    DeleteRegKey ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}"

    ; Remove install directory if empty
    RMDir "$INSTDIR"

    ; Self-delete the uninstaller using ping approach
    SetAutoClose true
    ${If} ${FileExists} "$INSTDIR"
        ExecWait 'cmd.exe /C ping 127.0.0.1 -n 2 && del /F /Q "$EXEPATH"'
    ${EndIf}
SectionEnd

Function GetUbuntuDistros
    DetailPrint "Starting GetUbuntuDistros..."
    StrCpy $R0 ""  ; Result string

    DetailPrint "Checking registry key..."
    SetRegView 64
    ClearErrors
    EnumRegKey $R1 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss" 0
    ${If} ${Errors}
        DetailPrint "Error accessing Lxss registry key"
        Push ""
        Return
    ${EndIf}
    DetailPrint "Registry key exists and is accessible"

    ; Count total number of keys first
    StrCpy $R1 0   ; Index for enumeration
    StrCpy $R5 0   ; Total count
    count_loop:
        ClearErrors
        EnumRegKey $R2 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss" $R1
        ${If} ${Errors}
        ${OrIf} $R2 == ""
            Goto count_done
        ${EndIf}
        IntOp $R5 $R5 + 1
        IntOp $R1 $R1 + 1
        Goto count_loop
    count_done:
    DetailPrint "Found $R5 total WSL distributions"

    ; Now enumerate and check each key
    StrCpy $R1 0   ; Reset index
    ${While} $R1 < $R5
        ClearErrors
        EnumRegKey $R2 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss" $R1
        ${If} ${Errors}
            DetailPrint "Error enumerating key at index $R1"
            Goto next_key
        ${EndIf}

        ClearErrors
        ReadRegStr $R3 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss\$R2" "DistributionName"
        ${If} ${Errors}
            DetailPrint "Error reading DistributionName for key $R2"
            Goto next_key
        ${EndIf}

        ; Check if it starts with "Ubuntu"
        StrCpy $R4 $R3 6
        ${If} $R4 == "Ubuntu"
            DetailPrint "Found Ubuntu distribution: $R3"
            ${If} $R0 != ""
                StrCpy $R0 "$R0|"
            ${EndIf}
            StrCpy $R0 "$R0$R3"
        ${EndIf}

        next_key:
        IntOp $R1 $R1 + 1
    ${EndWhile}

    DetailPrint "Registry enumeration complete. Final list: [$R0]"
    Push $R0
FunctionEnd

; TODO: there seem to be missing error checks here.
Function InstallWSL2CommonSetup
    ; Check for WSL2
    DetailPrint "Checking WSL2 version..."
    nsExec::ExecToStack 'wsl.exe -l -v'
    Pop $1
    Pop $0
    DetailPrint "WSL version check output: $0"
    DetailPrint "WSL version check exit code: $1"
    ${If} $1 != 0
        DetailPrint "ERROR: WSL2 not detected - exit code: $1, output: $0"
        Push "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
        Call ShowErrorAndAbort
    ${EndIf}

    ; Check for Ubuntu in selected distro
    DetailPrint "Checking selected distro $SELECTED_DISTRO..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "cat /etc/os-release | grep -i ^NAME="'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Cannot access distro $SELECTED_DISTRO - exit code: $1, output: $0"
        Push "Could not access the selected distro. Please ensure it's working properly."
        Call ShowErrorAndAbort
    ${EndIf}

    ; Check for WSL2 kernel
    DetailPrint "Checking WSL2 kernel..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO uname -v'
    Pop $1
    Pop $0
    DetailPrint "WSL kernel version: $0"
    ${If} $1 != 0
        DetailPrint "ERROR: WSL version check failed - exit code: $1, output: $0"
        Push "Could not check WSL version. Please ensure WSL is working."
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $0 == ""
        DetailPrint "ERROR: Empty WSL version output"
        Push "Could not detect WSL version. Please ensure WSL is working."
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $0 == "WSL"
        DetailPrint "ERROR: WSL1 detected instead of WSL2 - version output: $0"
        Push "The selected distro ($SELECTED_DISTRO) is not WSL2. Please use a WSL2 distro."
        Call ShowErrorAndAbort
    ${EndIf}
    DetailPrint "WSL2 detected successfully."

    ; Check for non-root default user
    DetailPrint "Checking for non-root user..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO whoami'
    Pop $1  ; error code
    Pop $0  ; output
    DetailPrint "Current user: $0"
    ${If} $1 != 0
        DetailPrint "ERROR: WSL user check failed - exit code: $1, output: $0"
        Push "Could not check WSL user. Please ensure WSL is working."
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $0 == "root"
        DetailPrint "ERROR: Default WSL user is root - this is not supported"
        Push "The default user in your WSL2 distro is root. Please configure an ordinary default user."
        Call ShowErrorAndAbort
    ${EndIf}
    DetailPrint "Non-root user detected successfully."

    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Check if Docker is already working in WSL2
        DetailPrint "Checking Docker Desktop connectivity..."
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO docker ps'
        Pop $1
        Pop $0
        ${If} $1 != 0
            DetailPrint "ERROR: Docker Desktop/Rancher Desktop not accessible in WSL2 - exit code: $1, output: $0"
            Push "Docker provider is not working in WSL2. Please ensure Docker Desktop, Rancher Desktop, or another provider is running and integration with the $SELECTED_DISTRO distro is enabled."
            Call ShowErrorAndAbort
        ${EndIf}

        ; Make sure we're not running docker-ce or docker.io daemon
        DetailPrint "Verifying Docker installation type..."
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO pgrep dockerd'
        Pop $1
        Pop $0
        ${If} $1 == 0
            DetailPrint "ERROR: Local Docker daemon detected in WSL2 - conflicts with Docker Desktop. Process list: $0"
            Push "A local Docker daemon (from docker-ce or docker.io) is running in WSL2. This conflicts with Docker Desktop. Please remove Docker first ('sudo apt-get remove docker-ce' or 'sudo apt-get remove docker.io')."
            Call ShowErrorAndAbort
        ${EndIf}
    ${EndIf}

    ; Remove old Docker versions first
    DetailPrint "WSL($SELECTED_DISTRO): Removing old Docker packages if present..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"'
    Pop $1
    Pop $0
    ; Note: This command is allowed to fail if packages aren't installed

    ; apt-get upgrade
    DetailPrint "WSL($SELECTED_DISTRO): Doing apt-get upgrade..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "apt-get update && apt-get upgrade -y >/dev/null 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: apt-get update/upgrade failed - exit code: $1, output: $0"
        Push "Failed to update/upgrade packages. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Install linux packages
    DetailPrint "WSL($SELECTED_DISTRO): Installing required linux packages..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root apt-get install -y ca-certificates curl gnupg gnupg2 libsecret-1-0 lsb-release pass'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to install dependencies - exit code: $1, output: $0"
        Push "Failed to install dependencies. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Create keyrings directory if it doesn't exist
    DetailPrint "WSL($SELECTED_DISTRO): Setting up keyrings directory..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root install -m 0755 -d /etc/apt/keyrings'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to create keyrings directory - exit code: $1, output: $0"
        Push "Failed to create keyrings directory. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add Docker GPG key
    DetailPrint "WSL($SELECTED_DISTRO): Adding Docker repository key..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to add Docker repository key - exit code: $1, output: $0"
        Push "Failed to add Docker apt repository key. Please check your internet connection. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add Docker repository
    DetailPrint "WSL($SELECTED_DISTRO): Adding Docker apt repository..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root -e bash -c "echo deb [arch=$$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $$(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to add Docker repository - exit code: $1, output: $0"
        Push "Failed to add Docker repository. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add DDEV GPG key
    DetailPrint "WSL($SELECTED_DISTRO): Adding DDEV apt repository key..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to add DDEV repository key - exit code: $1, output: $0"
        Push "Failed to add DDEV repository key. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add DDEV repository
    DetailPrint "WSL($SELECTED_DISTRO): Adding DDEV apt repository..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root -e bash -c "echo \"deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *\" > /etc/apt/sources.list.d/ddev.list"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to add DDEV repository - exit code: $1, output: $0"
        Push "Failed to add DDEV repository. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Update package lists
    DetailPrint "WSL($SELECTED_DISTRO): apt-get update..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: apt-get update failed - exit code: $1, output: $0"
        Push "Failed to apt-get update. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}
FunctionEnd

Function InstallWSL2Common
    DetailPrint "Starting WSL2 Docker installation for $SELECTED_DISTRO"
    Call InstallWSL2CommonSetup

    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Install packages needed for Docker Desktop (including ddev)
        StrCpy $0 "docker-ce-cli wslu ddev"
    ${Else}
        ; Install full Docker CE packages (including ddev)
        StrCpy $0 "docker-ce docker-ce-cli containerd.io wslu ddev"
    ${EndIf}

    ; Install the selected packages
    DetailPrint "WSL($SELECTED_DISTRO): apt-get install $0."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y $0 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        DetailPrint "ERROR: Package installation failed - exit code: $1, output: $2"
        Push "Failed to install packages. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Overwrite the installed DDEV binary with the bundled version
    DetailPrint "WSL($SELECTED_DISTRO): Overwriting DDEV binary with bundled version..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "/mnt/c/Windows/Temp/ddev_installer/ddev_linux" /usr/bin/ddev'
    Pop $1
    Pop $2
    ${If} $1 != 0
        DetailPrint "ERROR: DDEV binary overwrite failed - exit code: $1, output: $2"
        Push "Failed to overwrite DDEV binary. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Make it executable
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root chmod +x /usr/bin/ddev'
    Pop $1
    Pop $2
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to make DDEV binary executable - exit code: $1, output: $2"
        Push "Failed to make DDEV binary executable. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Overwrite the installed ddev-hostname binary with the bundled version
    DetailPrint "WSL($SELECTED_DISTRO): Overwriting ddev-hostname binary with bundled version..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "/mnt/c/Windows/Temp/ddev_installer/ddev-hostname_linux" /usr/bin/ddev-hostname'
    Pop $1
    Pop $2
    ${If} $1 != 0
        DetailPrint "ERROR: ddev-hostname binary overwrite failed - exit code: $1, output: $2"
        Push "Failed to overwrite ddev-hostname binary. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Make ddev-hostname executable
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root chmod +x /usr/bin/ddev-hostname'
    Pop $1
    Pop $2
    ${If} $1 != 0
        DetailPrint "ERROR: Failed to make ddev-hostname binary executable - exit code: $1, output: $2"
        Push "Failed to make ddev-hostname binary executable. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    

    ; Add the unprivileged user to the docker group for docker-ce installation
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        DetailPrint "WSL($SELECTED_DISTRO): Getting username of unprivileged user..."
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO whoami'
        Pop $1
        Pop $2
        ${If} $1 != 0
            DetailPrint "ERROR: Failed to get WSL2 username - exit code: $1, output: $2"
            Push "Failed to get WSL2 username. Error: $2"
            Call ShowErrorAndAbort
        ${EndIf}
        
        ; Trim whitespace from username
        Push $2
        Call TrimNewline
        Pop $2
        
        DetailPrint "WSL($SELECTED_DISTRO): Adding user '$2' to docker group..."
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root usermod -aG docker $2'
        Pop $1
        Pop $3
        ${If} $1 != 0
            DetailPrint "Warning: Failed to add user to docker group. Error: $3"
            MessageBox MB_ICONEXCLAMATION|MB_OK "Warning: Failed to add user '$2' to docker group. You may need to run 'sudo usermod -aG docker $2' manually in WSL2."
        ${Else}
            DetailPrint "Successfully added user '$2' to docker group."
        ${EndIf}
    ${EndIf}

    ; Show DDEV version
    DetailPrint "Verifying DDEV installation with 'ddev version'..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO ddev version'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: DDEV version check failed - exit code: $1, output: $0"
        Push "WSL($SELECTED_DISTRO) doesn't seem to have working 'ddev version'. Please execute it manually in $SELECTED_DISTRO to debug the problem. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Set up mkcert in WSL2 if we're doing WSL2 installation
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
    ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        Call SetupMkcertInWSL2
    ${EndIf}

    ; Final validation - ensure DDEV is actually working
    DetailPrint "Performing final validation of DDEV installation..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO ddev version'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Final DDEV validation failed - exit code: $1, output: $0"
        Push "Installation validation failed. DDEV may not be working properly. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Clean up temp directory
    DetailPrint "Cleaning up temporary files..."
    Delete "C:\Windows\Temp\ddev_installer\ddev_linux"
    Delete "C:\Windows\Temp\ddev_installer\ddev-hostname_linux"
    RMDir "C:\Windows\Temp\ddev_installer"
    
    DetailPrint "All done! Installation completed successfully and validated."
    ${IfNot} ${Silent}
        MessageBox MB_ICONINFORMATION|MB_OK "DDEV WSL2 installation completed successfully."
    ${EndIf}
FunctionEnd

Function InstallTraditionalWindows
    DetailPrint "Starting InstallTraditionalWindows"

    ; Check for Docker provider on Windows
    DetailPrint "Checking for Docker provider on Windows..."
    nsExec::ExecToStack 'docker ps'
    Pop $1
    Pop $0
    ${If} $1 != 0
        DetailPrint "ERROR: Docker provider check failed - exit code: $1, output: $0"
        Push "Docker provider not found or not working on Windows.$\n$\nTraditional Windows installation requires a working Docker provider like Docker Desktop or Rancher Desktop.$\n$\nPlease install Docker Desktop (https://www.docker.com/products/docker-desktop/) or Rancher Desktop (https://rancherdesktop.io/) and make sure it's running before installing DDEV."
        Call ShowErrorAndAbort
    ${EndIf}
    DetailPrint "Docker provider check successful."

    ; Remove CAROOT environment variable for traditional Windows (WSL2-specific)
    DetailPrint "Removing CAROOT environment variable (not needed for traditional Windows)"
    DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"

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

    ; Use literal names for website and documentation
    WriteIniStr "$INSTDIR\Links\DDEV Website.url" "InternetShortcut" "URL" "https://ddev.com"
    CreateShortCut "$SMPROGRAMS\$StartMenuGroup\DDEV Website.lnk" "$INSTDIR\Links\DDEV Website.url" "" "$INSTDIR\Icons\ddev.ico"

    WriteIniStr "$INSTDIR\Links\DDEV Documentation.url" "InternetShortcut" "URL" "https://ddev.readthedocs.io"
    CreateShortCut "$SMPROGRAMS\$StartMenuGroup\DDEV Documentation.lnk" "$INSTDIR\Links\DDEV Documentation.url" "" "$INSTDIR\Icons\ddev.ico"

    !insertmacro MUI_STARTMENU_WRITE_END

    DetailPrint "Traditional Windows installation completed."
    ${IfNot} ${Silent}
        MessageBox MB_ICONINFORMATION|MB_OK "DDEV Traditional Windows installation completed successfully."
    ${EndIf}
FunctionEnd

Function RunMkcertInstall
    DetailPrint "Setting up mkcert.exe (Windows) for trusted HTTPS certificates..."
    ${IfNot} ${Silent}
        MessageBox MB_ICONINFORMATION|MB_OK "Now setting up mkcert.exe to enable trusted https. Please accept the mkcert dialog box that may follow."
    ${EndIf}
    
    ; Unset CAROOT environment variable in current process
    System::Call 'kernel32::SetEnvironmentVariable(t "CAROOT", i 0)'
    Pop $0

    ; Run mkcert.exe -install to create fresh certificate authority
    DetailPrint "Running mkcert.exe -install to create certificate authority..."
    nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -install'
    Pop $R0
    Pop $R1 ; Output
    ${If} $R0 = 0
        DetailPrint "mkcert.exe -install completed successfully"
        WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup" 1
        
        ; Set up CAROOT environment variable for WSL2 sharing (only for WSL2 installs)
        ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
            Call SetupMkcertForWSL2
        ${EndIf}
    ${Else}
        DetailPrint "mkcert.exe -install failed with exit code: $R0"
        MessageBox MB_ICONEXCLAMATION|MB_OK "mkcert -install failed with exit code: $R0. Output: $R1. You may need to run 'mkcert.exe -install' manually on Windows."
    ${EndIf}
FunctionEnd

Function SetupMkcertForWSL2
    DetailPrint "Setting up mkcert certificate sharing with WSL2..."
    
    ; Get the CAROOT directory from mkcert (mkcert -install already completed)
    nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -CAROOT'
    Pop $R0 ; error code
    Pop $R1 ; output (CAROOT path)
    
    ${If} $R0 = 0
        ; Trim whitespace from CAROOT path
        Push $R1
        Call TrimNewline
        Pop $R1
        
        DetailPrint "CAROOT directory: $R1"
        
        ; Set CAROOT environment variable using EnVar plugin
        EnVar::SetHKLM
        EnVar::Delete "CAROOT"  ; Remove entire variable first
        Pop $0  ; Get error code from Delete
        DetailPrint "EnVar::Delete CAROOT result: $0"
        
        EnVar::AddValue "CAROOT" "$R1"
        Pop $0  ; Get error code from AddValue
        DetailPrint "EnVar::AddValue CAROOT result: $0"
        
        ; Get current WSLENV value from registry
        ReadRegStr $R2 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
        ${If} ${Errors}
            StrCpy $R2 ""
        ${EndIf}

        ; Store original value for debugging
        StrCpy $R4 $R2
        
        ${If} $R2 != ""
            StrCpy $R2 "CAROOT/up:$R2"
        ${Else}
            StrCpy $R2 "CAROOT/up"
        ${EndIf}
        
        EnVar::SetHKLM
        EnVar::Delete "WSLENV"  ; Remove existing WSLENV entirely
        Pop $0  ; Get error code from Delete
        ; DetailPrint "EnVar::Delete WSLENV result: $0"
        
        EnVar::AddValue "WSLENV" "$R2"
        Pop $0  ; Get error code from AddValue
        ; DetailPrint "EnVar::AddValue WSLENV result: $0"
        ; DetailPrint "Original WSLENV was: [$R4]"
        ; DetailPrint "New WSLENV set to: [$R2]"
        
        ; Verify by reading from registry
        ReadRegStr $R5 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
        ; DetailPrint "WSLENV read back from registry: [$R5]"
        
        DetailPrint "mkcert certificate sharing with WSL2 configured successfully."
        
        ; Read current value from registry for verification
        ; ReadRegStr $R6 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
        ; DetailPrint "WSLENV verification - Original: [$R4], Set to: [$R2], Actual: [$R6]"
    ${Else}
        DetailPrint "Failed to get CAROOT directory from mkcert"
        MessageBox MB_ICONEXCLAMATION|MB_OK "Failed to get CAROOT directory from mkcert. WSL2 certificate sharing may not work properly."
    ${EndIf}
FunctionEnd

Function SetupMkcertInWSL2
    DetailPrint "Setting up mkcert inside WSL2 distro: $SELECTED_DISTRO"
    
    ; Check current Windows CAROOT environment variable from registry
    ReadRegStr $R2 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
    ; DetailPrint "Windows CAROOT environment variable: $R2"
    
    ; Check current Windows WSLENV environment variable from registry
    ReadRegStr $R3 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
    ; DetailPrint "Windows WSLENV environment variable: $R3"
    
    ; Verify CAROOT is accessible in WSL2
    DetailPrint "Verifying CAROOT is accessible in WSL2..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo CAROOT=$$CAROOT"'
    Pop $R0
    Pop $R1
    DetailPrint "WSL2 CAROOT check result: $R1"
    
    ; Validate that WSL2 CAROOT is accessible
    ${If} $R0 = 0
        ; Check that CAROOT isn't empty and starts with /mnt (indicating Windows path)
        Push $R1
        Push "CAROOT="
        Call StrContains
        Pop $R4
        ${If} $R4 != ""
        ${AndIf} $R1 != "CAROOT="
            Push $R1
            Push "/mnt"
            Call StrContains
            Pop $R5
            ${If} $R5 != ""
                DetailPrint "CAROOT appears valid, testing accessibility..."
                ; Test if CAROOT directory is accessible
                nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "ls $$CAROOT >/dev/null 2>&1"'
                Pop $R6
                Pop $R7
                
                ${If} $R6 != 0
                    MessageBox MB_ICONEXCLAMATION|MB_OK "CAROOT directory is not accessible from WSL2. Certificate sharing may not work properly."
                ${Else}
                    DetailPrint "WSL2 CAROOT validation successful"
                ${EndIf}
            ${Else}
                MessageBox MB_ICONEXCLAMATION|MB_OK "CAROOT does not appear to be a Windows path accessible from WSL2. Certificate sharing may not work properly."
            ${EndIf}
        ${Else}
            MessageBox MB_ICONEXCLAMATION|MB_OK "CAROOT environment variable is empty in WSL2. Certificate sharing may not work properly."
        ${EndIf}
    ${Else}
        DetailPrint "Failed to check CAROOT in WSL2"
        MessageBox MB_ICONEXCLAMATION|MB_OK "Failed to access CAROOT environment variable in WSL2. Certificate sharing may not work properly."
    ${EndIf}
    
    ; Run mkcert -install in WSL2
    DetailPrint "Running mkcert -install in WSL2..."
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root mkcert -install'
    Pop $R0
    Pop $R1
    
    ${If} $R0 = 0
        DetailPrint "mkcert -install completed successfully in WSL2."
        DetailPrint "WSL2 mkcert output: $R1"
    ${Else}
        DetailPrint "mkcert -install failed in WSL2 with exit code: $R0"
        DetailPrint "WSL2 mkcert error: $R1"
        MessageBox MB_ICONEXCLAMATION|MB_OK "mkcert -install failed in WSL2 with exit code: $R0. Error: $R1. You may need to run 'mkcert -install' manually in WSL2 later."
    ${EndIf}

FunctionEnd

Function un.onInit
  MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Are you sure you want to completely remove $(^Name) and all of its components?" IDYES DoUninstall
  Abort

DoUninstall:
  ; Switch to 64 bit view and disable FS redirection
  SetRegView 64
  ${DisableX64FSRedirection}
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

Function ParseCommandLine
    ; Initialize variables
    StrCpy $SILENT_INSTALL_TYPE ""
    StrCpy $SILENT_DISTRO ""
    
    ; Get command line
    ${GetParameters} $R0
    DetailPrint "Command line parameters: $R0"
    
    ; Check for /help argument
    ${GetOptions} $R0 "/help" $R1
    ${IfNot} ${Errors}
        MessageBox MB_ICONINFORMATION|MB_OK "DDEV Windows Installer Usage:$\n$\n\
            /docker-ce /distro=<name>     - WSL2 with Docker CE (Recommended)$\n\
            /docker-desktop /distro=<name> - WSL2 with Docker Desktop$\n\
            /rancher-desktop /distro=<name> - WSL2 with Rancher Desktop$\n\
            /traditional                  - Traditional Windows install$\n\
            /S                           - Silent install$\n\
            /help or /?                  - Show this help message$\n$\n\
            Examples:$\n\
            installer.exe /docker-ce /distro=Ubuntu-22.04$\n\
            installer.exe /docker-desktop /distro=Ubuntu-20.04$\n\
            installer.exe /traditional$\n\
            installer.exe /traditional /S"
        Abort
    ${EndIf}
    
    ; Check for /? argument  
    ${GetOptions} $R0 "/?" $R1
    ${IfNot} ${Errors}
        MessageBox MB_ICONINFORMATION|MB_OK "DDEV Windows Installer Usage:$\n$\n\
            /docker-ce /distro=<name>     - WSL2 with Docker CE (Recommended)$\n\
            /docker-desktop /distro=<name> - WSL2 with Docker Desktop$\n\
            /rancher-desktop /distro=<name> - WSL2 with Rancher Desktop$\n\
            /traditional                  - Traditional Windows install$\n\
            /S                           - Silent install$\n\
            /help or /?                  - Show this help message$\n$\n\
            Examples:$\n\
            installer.exe /docker-ce /distro=Ubuntu-22.04$\n\
            installer.exe /docker-desktop /distro=Ubuntu-20.04$\n\
            installer.exe /traditional$\n\
            installer.exe /traditional /S"
        Abort
    ${EndIf}
    
    ; Check for /docker-ce argument
    ${GetOptions} $R0 "/docker-ce" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "wsl2-docker-ce"
        DetailPrint "Found /docker-ce argument"
    ${EndIf}
    
    ; Check for /docker-desktop argument
    ${GetOptions} $R0 "/docker-desktop" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "wsl2-docker-desktop"
        DetailPrint "Found /docker-desktop argument"
    ${EndIf}
    
    ; Check for /rancher-desktop argument
    ${GetOptions} $R0 "/rancher-desktop" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "wsl2-docker-desktop"
        DetailPrint "Found /rancher-desktop argument"
    ${EndIf}
    
    ; Check for /traditional argument
    ${GetOptions} $R0 "/traditional" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "traditional"
        DetailPrint "Found /traditional argument"
    ${EndIf}
    
    ; Check for /distro argument
    ${GetOptions} $R0 "/distro=" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_DISTRO $R1
        DetailPrint "Found /distro argument: $SILENT_DISTRO"
    ${EndIf}
    
    ; Validate that distro is specified for WSL2 installation types
    ${If} $SILENT_INSTALL_TYPE == "wsl2-docker-ce"
    ${OrIf} $SILENT_INSTALL_TYPE == "wsl2-docker-desktop"
        ${If} $SILENT_DISTRO == ""
            DetailPrint "ERROR: Missing required /distro argument for WSL2 installation type: $SILENT_INSTALL_TYPE"
            MessageBox MB_ICONSTOP|MB_OK "The /distro=<distro_name> argument is required when using /docker-ce, /docker-desktop, or /rancher-desktop.$\n$\nExample: installer.exe /docker-ce /distro=Ubuntu-22.04"
            Abort
        ${EndIf}
    ${EndIf}
FunctionEnd

Function .onInit
    ; Set proper 64-bit handling
    SetRegView 64
    ${DisableX64FSRedirection}

    ; Initialize directory to proper Program Files location
    ${If} ${RunningX64}
        StrCpy $INSTDIR "$PROGRAMFILES64\DDEV"
    ${Else}
        MessageBox MB_ICONSTOP|MB_OK "This installer is for 64-bit Windows only."
        Abort
    ${EndIf}
    
    ; Parse command line arguments
    Call ParseCommandLine
    
    ; Handle installation type selection
    ${If} $SILENT_INSTALL_TYPE != ""
        ; Command line argument specified - use it
        StrCpy $INSTALL_OPTION $SILENT_INSTALL_TYPE
        ${If} $SILENT_DISTRO != ""
            StrCpy $SELECTED_DISTRO $SILENT_DISTRO
        ${EndIf}
        DetailPrint "Command line install with type: $INSTALL_OPTION"
    ${ElseIf} ${Silent}
        ; Legacy silent install (Chocolatey) - default to traditional
        StrCpy $INSTALL_OPTION "traditional"
        DetailPrint "Silent install detected, defaulting to traditional Windows installation"
    ${EndIf}
FunctionEnd

; Helper: Show error message with standard guidance and abort
; Call with error message on stack
Function ShowErrorAndAbort
    Exch $R0  ; Get error message from stack
    DetailPrint "INSTALLATION ERROR: $R0"
    MessageBox MB_ICONSTOP|MB_OK "$R0$\n$\nUse 'Show details' for more information. Then click 'Cancel', fix the issue, and retry the installer."
    Abort
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


; Helper: Trim trailing newline and carriage return from a string
Function TrimNewline
    Exch $R0
    Push $R1
    StrCpy $R1 $R0 -1
    loop_trimnl:
        StrCpy $R1 $R0 -1
        StrCpy $R2 $R0 1 -1
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

Function un.mkcertUninstall
    ${If} ${FileExists} "$INSTDIR\mkcert.exe"
        Push $0
        
        ; Read setup status from registry
        ReadRegDWORD $0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup"
        
        ; Check if setup was done
        ${If} $0 == 1
            ; Get user confirmation
            MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "mkcert was found in this installation. Do you like to remove the mkcert configuration?" /SD IDNO IDYES +2
            Goto Skip
            
            MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to disable trusted https. Please accept the mkcert dialog box that may follow."
            
            nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -uninstall'
            Pop $0 ; get return value
            Pop $1 ; get output
            
        Skip:
        ${EndIf}
        
        Pop $0
    ${EndIf}
FunctionEnd

Function un.CleanupMkcertEnvironment
    DetailPrint "Cleaning up mkcert environment variables..."
    
    ; Get CAROOT directory before cleanup
    ReadRegStr $R0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
    ${IfNot} ${Errors}
        DetailPrint "CAROOT directory: $R0"
        
        ; Run mkcert -uninstall first to properly clean up certificates
        ${If} ${FileExists} "$INSTDIR\mkcert.exe"
            DetailPrint "Running mkcert -uninstall to clean up certificates..."
            nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -uninstall'
            Pop $R1
            Pop $R2 ; get output
            ${If} $R1 = 0
                DetailPrint "mkcert -uninstall completed successfully"
            ${Else}
                DetailPrint "mkcert -uninstall failed with exit code: $R1"
            ${EndIf}
        ${EndIf}
        
        ; Remove any remaining CAROOT directory
        ${If} ${FileExists} "$R0"
            DetailPrint "Removing remaining CAROOT directory: $R0"
            RMDir /r "$R0"
        ${EndIf}
    ${EndIf}
    
    ; Remove CAROOT environment variable
    DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
    
    ; Clean up WSLENV by removing CAROOT/up
    ReadRegStr $R0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
    ${If} ${Errors}
        DetailPrint "WSLENV not found, nothing to clean up"
        Return
    ${EndIf}
    
    DetailPrint "Current WSLENV: $R0"
    
    ; Remove CAROOT/up: from the beginning
    ${WordFind} "$R0" "CAROOT/up:" "E+1{" $R1
    ${If} $R1 != $R0
        StrCpy $R0 $R1
    ${Else}
        ; Remove :CAROOT/up from anywhere else
        ${WordFind} "$R0" ":CAROOT/up" "E+1{" $R1
        ${If} $R1 != $R0
            StrCpy $R0 $R1
        ${Else}
            ; Check if it's just CAROOT/up by itself
            ${If} $R0 == "CAROOT/up"
                StrCpy $R0 ""
            ${EndIf}
        ${EndIf}
    ${EndIf}
    
    ; Update or delete WSLENV
    ${If} $R0 == ""
        DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
        DetailPrint "Removed empty WSLENV"
    ${Else}
        WriteRegStr HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV" "$R0"
        DetailPrint "Updated WSLENV to: $R0"
    ${EndIf}
    
    DetailPrint "mkcert environment variables cleanup completed"
FunctionEnd

; Installation completion callbacks for proper exit code handling
Function .onInstSuccess
    DetailPrint "Installation completed successfully"
    SetErrorLevel 0
FunctionEnd

Function .onInstFailed
    DetailPrint "Installation failed"
    SetErrorLevel 1
FunctionEnd
