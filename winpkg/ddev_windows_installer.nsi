!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"
!include "x64.nsh"
!include "WordFunc.nsh"
!include "StrFunc.nsh"

!insertmacro GetParameters
!insertmacro GetOptions
!insertmacro GetTime

!insertmacro WordFind
${StrStr}
${StrRep}
${UnStrRep}
${StrTrimNewLines}
${StrCase}
; Remove the Trim macro since we're using our own TrimWhitespace function

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif

Name "DDEV"
OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_windows_${TARGET_ARCH}_installer.exe"

; Use per-user installation directory to avoid requiring admin privileges
InstallDir "$LOCALAPPDATA\Programs\DDEV"
RequestExecutionLevel user

!define PRODUCT_NAME "DDEV"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "DDEV Foundation"
!define RELEASES_URL "https://github.com/ddev/ddev/releases"

; Variables
Var /GLOBAL INSTALL_OPTION
Var /GLOBAL SELECTED_DISTRO
Var /GLOBAL SILENT_INSTALL_TYPE
Var /GLOBAL SILENT_DISTRO
Var /GLOBAL WINDOWS_CAROOT
Var /GLOBAL DEBUG_LOG_HANDLE
Var /GLOBAL DEBUG_LOG_PATH
Var /GLOBAL WSL_WINDOWS_TEMP
Var /GLOBAL WINDOWS_TEMP
Var /GLOBAL MKCERT_UNINSTALL_APPROVED  ; Track if user approved mkcert removal during uninstall
Var /GLOBAL DOCKER_DISTRO_FAMILY       ; "ubuntu" or "debian" for Docker CE repo selection
Var /GLOBAL DOCKER_SUITE               ; Debian/Ubuntu codename for Docker CE repo (e.g. bookworm, trixie, noble)
Var /GLOBAL DISTRO_LISTBOX_HANDLE      ; Handle for distro selection ListBox
Var /GLOBAL DISTRO_LIST                ; Pipe-separated list of detected distros

!define REG_INSTDIR_ROOT "HKCU"
!define REG_INSTDIR_KEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe"
!define REG_UNINST_ROOT "HKCU"
!define REG_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define REG_SETTINGS_ROOT "HKCU"
!define REG_SETTINGS_KEY "Software\DDEV\Settings"

; Installer Types
InstType "Full"
InstType "Simple"
InstType "Minimal"

!define MUI_ICON "graphics\ddev-install.ico"
!define MUI_UNICON "graphics\ddev-uninstall.ico"

!define MUI_INSTFILESPAGE_ABORTHEADER_TEXT "Installation Aborted"
!define MUI_INSTFILESPAGE_ABORTHEADER_SUBTEXT "Setup was not completed successfully."

!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "graphics\ddev-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "graphics\ddev-wizard.bmp"

!define MUI_ABORTWARNING

; Function declarations - must be before page definitions

; InitializeDebugLog - Open debug log file for writing
Function InitializeDebugLog
    ; Get current timestamp for filename and log header
    ; GetTime returns: $0=day, $1=month, $2=year, $3=dayofweek, $4=hour, $5=minute, $6=second
    ${GetTime} "" "L" $0 $1 $2 $3 $4 $5 $6
    
    ; Simple padding function for 2-digit format
    StrLen $R1 $1
    ${If} $R1 == 1
        StrCpy $1 "0$1"
    ${EndIf}
    StrLen $R1 $0
    ${If} $R1 == 1
        StrCpy $0 "0$0"
    ${EndIf}
    StrLen $R1 $4
    ${If} $R1 == 1
        StrCpy $4 "0$4"
    ${EndIf}
    StrLen $R1 $5
    ${If} $R1 == 1
        StrCpy $5 "0$5"
    ${EndIf}
    StrLen $R1 $6
    ${If} $R1 == 1
        StrCpy $6 "0$6"
    ${EndIf}
    
    ; Format: YYYYMMDD.HHMMSS
    StrCpy $R0 "$2$1$0.$4$5$6"
    
    StrCpy $DEBUG_LOG_PATH "$TEMP\ddev_installer_debug_$R0.log"
    FileOpen $DEBUG_LOG_HANDLE "$DEBUG_LOG_PATH" w
    ${If} $DEBUG_LOG_HANDLE != ""
        FileWrite $DEBUG_LOG_HANDLE "$R0$\r$\n"
        FileWrite $DEBUG_LOG_HANDLE "=== DDEV Installer Debug Log ===$\r$\n"
        FileWrite $DEBUG_LOG_HANDLE "Log location: $DEBUG_LOG_PATH$\r$\n"
        FileWrite $DEBUG_LOG_HANDLE "Installer started at: $2-$1-$0 $4:$5:$6$\r$\n"
    ${EndIf}
FunctionEnd

; LogPrint - DetailPrint wrapper that also writes to debug log with timestamp
; Usage: Push "message" ; Call LogPrint
Function LogPrint
    Exch $R0  ; Get message from stack
    Push $R1
    Push $R2  ; for formatted timestamp
    Push $0   ; save GetTime output registers
    Push $1
    Push $2
    Push $3
    Push $4
    Push $5
    Push $6

    ; Get current local time: $0=day $1=month $2=year $3=dayofweek $4=hour $5=min $6=sec
    ${GetTime} "" "L" $0 $1 $2 $3 $4 $5 $6

    ; Zero-pad hour, minute, second
    StrLen $R1 $4
    ${If} $R1 == 1
        StrCpy $4 "0$4"
    ${EndIf}
    StrLen $R1 $5
    ${If} $R1 == 1
        StrCpy $5 "0$5"
    ${EndIf}
    StrLen $R1 $6
    ${If} $R1 == 1
        StrCpy $6 "0$6"
    ${EndIf}

    StrCpy $R2 "$4:$5:$6"

    Pop $6
    Pop $5
    Pop $4
    Pop $3
    Pop $2
    Pop $1
    Pop $0

    ; Write to installer window and log file with timestamp prefix
    DetailPrint "$R2 $R0"

    ${If} $DEBUG_LOG_HANDLE != ""
        FileWrite $DEBUG_LOG_HANDLE "$R2 $R0$\r$\n"
    ${EndIf}

    Pop $R2
    Pop $R1
    Pop $R0
FunctionEnd

; InstallScriptToDistro - Copy a script from Windows temp to WSL2 distro and make it executable
; Usage: 
;   Push "distro_name"     ; WSL2 distro name
;   Push "script_name.sh"  ; Script name (without path)
;   Call InstallScriptToDistro
;   Pop $result            ; 0 = success, non-zero = error
Function InstallScriptToDistro
    Pop $R0  ; Get script name from stack
    Pop $R1  ; Get distro name from stack
    
    ; Validate script name is not empty for security
    ${If} $R0 == ""
        Push "ERROR: Script name cannot be empty"
        Call LogPrint
        Push 1
        Return
    ${EndIf}

    Push "Installing script $R0 to WSL2 distro $R1..."
    Call LogPrint
    
    ; Scripts should already be copied to temp directory by this point
    ${If} ${FileExists} "$WINDOWS_TEMP\ddev_installer\$R0"
        Push "Using script $R0 from temp directory"
        Call LogPrint
    ${Else}
        Push "ERROR: Script $R0 not found in temp directory"
        Call LogPrint
        Push 1
        Return
    ${EndIf}
    
    ; Remove any existing script first, then copy from Windows temp to WSL2 /tmp
    nsExec::ExecToStack 'wsl -d $R1 -u root rm -f /tmp/$R0'
    nsExec::ExecToStack 'wsl -d $R1 cp "$WSL_WINDOWS_TEMP/ddev_installer/$R0" /tmp/'
    Pop $R2  ; Exit code
    Pop $R3  ; Output
    
    ${If} $R2 != 0
        Push "Failed to copy script $R0 to distro $R1: exit code $R2, output: $R3"
        Call LogPrint
        Push $R2
        Return
    ${EndIf}
    
    ; Make script executable
    nsExec::ExecToStack 'wsl -d $R1 chmod +x /tmp/$R0'
    Pop $R2  ; Exit code  
    Pop $R3  ; Output
    
    ${If} $R2 != 0
        Push "Failed to make script $R0 executable in distro $R1: $R3"
        Call LogPrint
        Push $R2
        Return
    ${EndIf}
    
    Push "Successfully installed script $R0 to /tmp/$R0 in distro $R1"
    Call LogPrint
    Push 0  ; Success
FunctionEnd

Function DistroSelectionPage
    Push "Starting DistroSelectionPage..."
    Call LogPrint
    ${If} $INSTALL_OPTION != "wsl2-docker-ce"
    ${AndIf} $INSTALL_OPTION != "wsl2-docker-desktop"
        Push "Skipping distro selection for non-WSL2 install"
        Call LogPrint
        Abort
    ${EndIf}

    ; Skip this page if distro was specified via command line
    ${If} $SILENT_DISTRO != ""
        Push "Skipping distro selection - using command line distro: $SILENT_DISTRO"
        Call LogPrint
        StrCpy $SELECTED_DISTRO $SILENT_DISTRO
        Abort
    ${EndIf}

    Push "Creating dialog..."
    Call LogPrint
    nsDialogs::Create 1018
    Pop $0
    Push "Dialog create result: $0"
    Call LogPrint
    ${If} $0 == error
        Push "Failed to create dialog"
        Call LogPrint
        Abort
    ${EndIf}

    ; Get Debian-based distros before creating any controls
    Call GetDebianBasedDistros
    Pop $DISTRO_LIST
    Push "Got distros: [$DISTRO_LIST]"
    Call LogPrint
    ${If} $DISTRO_LIST == ""
        Push "ERROR: No Debian-based WSL2 distributions found"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "No Debian-based WSL2 distributions found. Please install Ubuntu or Debian for WSL2 first.$\n$\nDebug information has been written to: $DEBUG_LOG_PATH (please include with any error report)$\n$\nYou can check this file to see what distributions were detected."
        Push "No Debian-based WSL2 distributions found. Please install Ubuntu or Debian for WSL2 first."
        Call ShowErrorAndAbort
    ${EndIf}

    Push "Creating label..."
    Call LogPrint
    ${NSD_CreateLabel} 0 0 100% 24u "Select your Debian-based WSL2 distribution:"
    Pop $1

    Push "Creating listbox..."
    Call LogPrint
    ${NSD_CreateListBox} 10 30u 280u 130u ""
    Pop $DISTRO_LISTBOX_HANDLE

    ; Get previously selected distro
    ReadRegStr $R8 ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "SelectedDistro"
    Push "Previously selected distro: $R8"
    Call LogPrint

    ; Populate the listbox and determine default selection index
    StrCpy $R1 $DISTRO_LIST   ; Working copy
    StrCpy $R2 0              ; Current item index
    StrCpy $R3 0              ; Default selection index

    ${Do}
        StrCpy $R5 0
        ${Do}
            StrCpy $R6 $R1 1 $R5
            ${If} $R6 == "|"
            ${OrIf} $R6 == ""
                ${Break}
            ${EndIf}
            IntOp $R5 $R5 + 1
        ${Loop}

        ${If} $R5 > 0
            StrCpy $R7 $R1 $R5
            Push "Adding listbox item: [$R7]"
            Call LogPrint
            ${NSD_LB_AddString} $DISTRO_LISTBOX_HANDLE "$R7"

            ${If} $R7 == $R8
                StrCpy $R3 $R2
                Push "Will select: $R7 (index $R2, previous choice)"
                Call LogPrint
            ${EndIf}

            IntOp $R2 $R2 + 1
        ${EndIf}

        IntOp $R5 $R5 + 1
        StrCpy $R1 $R1 "" $R5

        ${If} $R1 == ""
            ${Break}
        ${EndIf}
    ${Loop}

    Push "Added $R2 items to listbox, selecting index $R3"
    Call LogPrint
    SendMessage $DISTRO_LISTBOX_HANDLE ${LB_SETCURSEL} $R3 0

    Push "About to show dialog..."
    Call LogPrint
    nsDialogs::Show
FunctionEnd

Function DistroSelectionPageLeave
    Push "Getting selected distro..."
    Call LogPrint

    ; NSD_LB_GetSelection returns the text of the selected item directly
    ${NSD_LB_GetSelection} $DISTRO_LISTBOX_HANDLE $SELECTED_DISTRO
    Push "Selected distro: $SELECTED_DISTRO"
    Call LogPrint

    ; Fallback if nothing selected
    ${If} $SELECTED_DISTRO == ""
        Push "No distro selected - using first available"
        Call LogPrint
        ${WordFind} "$DISTRO_LIST" "|" "+1{" $SELECTED_DISTRO
    ${EndIf}
    
    ; Store the selected distro for next time
    WriteRegStr ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "SelectedDistro" $SELECTED_DISTRO
    Push "Stored selected distro: $SELECTED_DISTRO"
    Call LogPrint

    ; Copy all scripts to temp directory for later use
    Push "Copying all scripts to temp directory..."
    Call LogPrint
    CreateDirectory "$WINDOWS_TEMP\ddev_installer"
    SetOutPath "$WINDOWS_TEMP\ddev_installer"
    File /oname=check_root_user.sh "scripts\check_root_user.sh"
    File /oname=mkcert_install.sh "scripts\mkcert_install.sh"
    File /oname=install_temp_sudoers.sh "scripts\install_temp_sudoers.sh"
    File /oname=detect_docker_suite.sh "scripts\detect_docker_suite.sh"
    File /oname=detect_docker_family.sh "scripts\detect_docker_family.sh"
    File /oname=apt_install_with_log.sh "scripts\apt_install_with_log.sh"
    File /oname=ensure_systemd_enabled.sh "scripts\ensure_systemd_enabled.sh"
    File /oname=wait_for_systemd.sh "scripts\wait_for_systemd.sh"
    Push "All scripts copied to temp directory"
    Call LogPrint
    
FunctionEnd


; CheckRootUser - Verify the default user is not root
Function CheckRootUser
    Push "=== Checking for root user in distro: $SELECTED_DISTRO ==="
    Call LogPrint
    
    Push $SELECTED_DISTRO
    Push "check_root_user.sh"
    Call InstallScriptToDistro
    Pop $R4
    ${If} $R4 != 0
        Push "Failed to install check_root_user.sh script"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "Failed to install check_root_user.sh to WSL2 distro"
        Abort
    ${EndIf}

    Push "Running check_root_user.sh..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash /tmp/check_root_user.sh'
    Pop $R4
    Pop $R5
    ${If} $R4 != 0
        Push "Root user detected in distro $SELECTED_DISTRO - exit code: $R4, output: $R5"
        Call LogPrint
        Push "Exiting installer due to root user detection"
        Call LogPrint
        Push "The default user in distro $SELECTED_DISTRO is still 'root', but it needs to be a normal user.$\n$\nPlease configure your WSL2 distro to use a normal user account instead of root.$\n$\nRefer to WSL documentation for instructions on changing the default user."
        Call ShowErrorAndAbort
    ${Else}
        Push "Root user check passed for distro $SELECTED_DISTRO - output: $R5"
        Call LogPrint
    ${EndIf}

FunctionEnd

; Define pages
!insertmacro MUI_PAGE_WELCOME

; License page for DDEV
!define MUI_PAGE_CUSTOMFUNCTION_PRE ddevLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE ddevLicLeave
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"

; Custom install scope page
Page custom InstallScopePage InstallScopePageLeave

; Custom install type selection
Page custom InstallChoicePage InstallChoicePageLeave

; Add WSL2 distro selection page
Page custom DistroSelectionPage DistroSelectionPageLeave

; Git for Windows check page for traditional installation
Page custom GitCheckPage GitCheckPageLeave

; Docker provider check page for all installations
Page custom DockerCheckPage DockerCheckPageLeave

; Directory page
!define MUI_DIRECTORYPAGE_TEXT_TOP "DDEV Windows-side components will be installed in this folder."
!define MUI_DIRECTORYPAGE_TEXT_DESTINATION "Windows install folder:"
!define MUI_DIRECTORYPAGE_HEADER_TEXT "Choose Windows install folder"
!insertmacro MUI_PAGE_DIRECTORY

; Start menu page - Disabled for per-user installation simplification
; !define MUI_STARTMENUPAGE_DEFAULTFOLDER "${PRODUCT_NAME}"
; !define MUI_STARTMENUPAGE_REGISTRY_ROOT ${REG_UNINST_ROOT}
; !define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINST_KEY}"
; !define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "StartMenuGroup"
; !define MUI_PAGE_CUSTOMFUNCTION_PRE StartMenuPagePre
; !insertmacro MUI_PAGE_STARTMENU Application $StartMenuGroup

; Installation page
!insertmacro MUI_PAGE_INSTFILES

; Standard MUI finish page with custom run action
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Support DDEV - Open GitHub Sponsors page"
!define MUI_FINISHPAGE_RUN_FUNCTION "LaunchSponsors"
!define MUI_FINISHPAGE_TITLE "DDEV Installation Complete"
!define MUI_FINISHPAGE_TEXT "Thank you for installing DDEV!$\r$\n$\r$\nPlease consider supporting DDEV so we can continue supporting you."
; MUI_FINISHPAGE_RUN_CHECKED is intentionally omitted: in silent (/S) mode NSIS
; automatically calls the run function when this is defined, which opens a browser
; window and can delay installer exit, causing CI timeout failures.
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_INSTFILES

; Language - must come after pages
!insertmacro MUI_LANGUAGE "English"

; Reserve plugin files for faster startup
ReserveFile /plugin EnVar.dll

Function InstallScopePage
    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 12u "Choose installation scope:"
    Pop $1

    ${NSD_CreateRadioButton} 10 20u 100% 12u "Install for current user only"
    Pop $2
    ${NSD_SetState} $2 ${BST_CHECKED}

    ${NSD_CreateRadioButton} 10 40u 100% 12u "Install for all users (not available)"
    Pop $3
    EnableWindow $3 0  ; Disable this option

    ${NSD_CreateLabel} 10 60u 100% 40u "DDEV must be installed per-user because WSL2 distros are per-user and cannot be shared between Windows users.$\r$\n$\r$\nEach Windows user who needs DDEV should run this installer under their own account.$\r$\n$\r$\nInstallation location: %LOCALAPPDATA%\Programs\DDEV"
    Pop $4

    nsDialogs::Show
FunctionEnd

Function InstallScopePageLeave
    ; Nothing to do - only one option available
FunctionEnd

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

    ; Read previous installation choice and set default
    ReadRegStr $R0 ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "InstallType"
    ${If} $R0 == "wsl2-docker-ce"
        ${NSD_SetState} $2 ${BST_CHECKED}
        Push "Set default to WSL2 with Docker CE based on previous installation"
        Call LogPrint
    ${ElseIf} $R0 == "wsl2-docker-desktop"
        ${NSD_SetState} $3 ${BST_CHECKED}
        Push "Set default to WSL2 with Docker Desktop based on previous installation"
        Call LogPrint
    ${ElseIf} $R0 == "traditional"
        ${NSD_SetState} $4 ${BST_CHECKED}
        Push "Set default to Traditional Windows based on previous installation"
        Call LogPrint
    ${Else}
        ; Default to Docker CE if no previous installation found
        ${NSD_SetState} $2 ${BST_CHECKED}
        Push "No previous installation found, defaulting to WSL2 with Docker CE"
        Call LogPrint
    ${EndIf}

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

  ; Store the selected installation type for next time
  WriteRegStr ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "InstallType" $INSTALL_OPTION
  Push "Stored installation type: $INSTALL_OPTION"
  Call LogPrint
FunctionEnd

Function GitCheckPage
    Push "Starting GitCheckPage..."
    Call LogPrint
    ; Skip this page if not traditional Windows installation
    ${If} $INSTALL_OPTION != "traditional"
        Push "Skipping Git check for non-traditional install: $INSTALL_OPTION"
        Call LogPrint
        Abort
    ${EndIf}

    ; Skip this page if install type was specified via command line
    ${If} $SILENT_INSTALL_TYPE != ""
        Push "Skipping Git check page for command line install"
        Call LogPrint
        Abort
    ${EndIf}

    ; Check for Git for Windows
    Push "Checking for Git for Windows before proceeding..."
    Call LogPrint
    Call CheckGitForWindows
    Pop $R0
    ${If} $R0 == "1"
        Push "Git for Windows found, proceeding with installation"
        Call LogPrint
        Abort ; Skip this page since Git is already installed
    ${EndIf}

    ; Git not found - show page to inform user
    Push "Git for Windows not found, showing information page"
    Call LogPrint
    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Push "Failed to create Git check dialog"
        Call LogPrint
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 48u "Git for Windows is required for traditional Windows installation but was not found.$\r$\n$\r$\nGit for Windows provides both Git and a Bash shell that DDEV needs to function properly.$\r$\n$\r$\nYou can install it now or cancel this installation."
    Pop $1

    ${NSD_CreateButton} 10 60u 120u 24u "Install Git for Windows"
    Pop $2
    ${NSD_OnClick} $2 GitInstallButtonClick

    ${NSD_CreateButton} 140u 60u 80u 24u "Cancel Installation"
    Pop $3
    ${NSD_OnClick} $3 GitCancelButtonClick

    nsDialogs::Show
FunctionEnd

Function GitCheckPageLeave
    ; This function is called when leaving the Git check page normally
    ; Check if Git was installed while on this page
    Call CheckGitForWindows
    Pop $R0
    ${If} $R0 == "1"
        Push "Git for Windows now detected, continuing installation"
        Call LogPrint
        Return
    ${EndIf}
    
    ; If we get here, Git is still not found but user somehow left the page
    ; This shouldn't normally happen with our button handlers
    Push "Leaving Git check page without Git installed"
    Call LogPrint
FunctionEnd

Function GitInstallButtonClick
    Push "User clicked Install Git for Windows button"
    Call LogPrint
    ExecShell "open" "https://gitforwindows.org/"
    MessageBox MB_ICONINFORMATION|MB_OK "Git for Windows download page opened in your browser.$\n$\nPlease download and install Git for Windows, then restart this installer.$\n$\nThe installer will now exit."
    Push "Exiting installer so user can install Git for Windows"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
FunctionEnd

Function GitCancelButtonClick
    Push "User clicked Cancel Installation button"
    Call LogPrint
    MessageBox MB_ICONINFORMATION|MB_OK "Installation cancelled.$\n$\nGit for Windows is required for traditional Windows installation.$\n$\nThe installer will now exit."
    Push "Exiting installer - user cancelled Git installation"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
FunctionEnd

Function CheckDockerProvider
    Push "Checking for Docker provider..."
    Call LogPrint
    
    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Check if Docker is accessible in WSL2
        Push "Checking Docker Desktop connectivity in WSL2..."
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO docker ps'
        Pop $R0
        Pop $R1
        ${If} $R0 == 0
            Push "Docker provider found in WSL2: $R1"
            Call LogPrint
            Push "1"
            Return
        ${Else}
            Push "Docker provider not accessible in WSL2: $R1"
            Call LogPrint
            Push "0"
            Return
        ${EndIf}
    ${Else}
        ; Check if Docker is accessible on Windows (traditional or WSL2 Docker CE setup)
        Push "Checking Docker provider on Windows..."
        Call LogPrint
        nsExec::ExecToStack 'docker ps'
        Pop $R0
        Pop $R1
        ${If} $R0 == 0
            Push "Docker provider found on Windows: $R1"
            Call LogPrint
            Push "1"
            Return
        ${Else}
            Push "Docker provider not accessible on Windows: $R1"
            Call LogPrint
            Push "0"
            Return
        ${EndIf}
    ${EndIf}
FunctionEnd

Function DockerCheckPage
    Push "Starting DockerCheckPage..."
    Call LogPrint
    
    ; Skip this page for wsl2-docker-ce since Docker CE will be installed during the process
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        Push "Skipping Docker check for wsl2-docker-ce install (Docker CE will be installed)"
        Call LogPrint
        Abort
    ${EndIf}
    
    ; Skip this page if install type was specified via command line
    ${If} $SILENT_INSTALL_TYPE != ""
        Push "Skipping Docker check page for command line install"
        Call LogPrint
        Abort
    ${EndIf}

    ; Check for Docker provider
    Push "Checking for Docker provider before proceeding..."
    Call LogPrint
    Call CheckDockerProvider
    Pop $R0
    ${If} $R0 == "1"
        Push "Docker provider found, proceeding with installation"
        Call LogPrint
        Abort ; Skip this page since Docker is already working
    ${EndIf}

    ; Docker not found - show page to inform user
    Push "Docker provider not found, showing information page"
    Call LogPrint
    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Push "Failed to create Docker check dialog"
        Call LogPrint
        Abort
    ${EndIf}

    ; Create different messages based on installation type
    ${If} $INSTALL_OPTION == "traditional"
        ${NSD_CreateLabel} 0 0 100% 60u "Docker provider is required for traditional Windows installation but was not found or is not running.$\r$\n$\r$\nPlease install and start Docker Desktop (https://www.docker.com/products/docker-desktop/) or Rancher Desktop (https://rancherdesktop.io/) before installing DDEV.$\r$\n$\r$\nYou can exit now to install Docker, or cancel this installation."
    ${ElseIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        ${NSD_CreateLabel} 0 0 100% 60u "Docker Desktop or Rancher Desktop is required for this installation but is not accessible in WSL2.$\r$\n$\r$\nEnsure that Docker Desktop or Rancher Desktop is installed, running, and has WSL2 integration enabled for the '$SELECTED_DISTRO' distro.$\r$\n$\r$\nThe 'docker ps' command must succeed inside WSL2 before launching this installer.$\r$\n$\r$\nYou can exit now to configure Docker, or cancel this installation."
    ${Else}
        ${NSD_CreateLabel} 0 0 100% 60u "Docker provider is required but was not found or is not running.$\r$\n$\r$\nPlease install and start a Docker provider before installing DDEV.$\r$\n$\r$\nYou can exit now to install Docker, or cancel this installation."
    ${EndIf}
    Pop $1

    ${NSD_CreateButton} 10 75u 120u 24u "Exit to Install Docker"
    Pop $2
    ${NSD_OnClick} $2 DockerExitButtonClick

    ${NSD_CreateButton} 140u 75u 80u 24u "Cancel Installation"
    Pop $3
    ${NSD_OnClick} $3 DockerCancelButtonClick

    nsDialogs::Show
FunctionEnd

Function DockerCheckPageLeave
    ; This function is called when leaving the Docker check page normally
    ; Check if Docker was configured while on this page
    Call CheckDockerProvider
    Pop $R0
    ${If} $R0 == "1"
        Push "Docker provider now detected, continuing installation"
        Call LogPrint
        Return
    ${EndIf}
    
    ; If we get here, Docker is still not found but user somehow left the page
    Push "Leaving Docker check page without Docker provider"
    Call LogPrint
FunctionEnd

Function DockerExitButtonClick
    Push "User clicked Exit to Install Docker button"
    Call LogPrint
    ${If} $INSTALL_OPTION == "traditional"
        MessageBox MB_ICONINFORMATION|MB_OK "Please install Docker Desktop or Rancher Desktop, ensure it's running, then restart this installer.$\n$\nThe installer will now exit."
    ${ElseIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        MessageBox MB_ICONINFORMATION|MB_OK "Please ensure Docker Desktop or Rancher Desktop is running and has WSL2 integration enabled for '$SELECTED_DISTRO', then restart this installer.$\n$\nThe installer will now exit."
    ${Else}
        MessageBox MB_ICONINFORMATION|MB_OK "Please install and configure a Docker provider, then restart this installer.$\n$\nThe installer will now exit."
    ${EndIf}
    Push "Exiting installer so user can install/configure Docker"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
    Quit
FunctionEnd

Function DockerCancelButtonClick
    Push "User clicked Cancel Installation button"
    Call LogPrint
    MessageBox MB_ICONINFORMATION|MB_OK "Installation cancelled.$\n$\nA Docker provider is required for DDEV installation.$\n$\nThe installer will now exit."
    Push "Exiting installer - user cancelled Docker installation"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
    Quit
FunctionEnd

Section "-Initialize"
    ; Create the installation directory
    CreateDirectory "$INSTDIR"
SectionEnd

SectionGroup /e "${PRODUCT_NAME}"
    Section "${PRODUCT_NAME}" SecDDEV
        SectionIn 1 2 3 RO

        ; Ensure 64-bit file system redirection is disabled
        ; This is critical for accessing wsl.exe in System32
        ${DisableX64FSRedirection}
        SetRegView 64

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
            SetOutPath "$WINDOWS_TEMP\ddev_installer"
            File /oname=ddev_linux "..\.gotmp\bin\linux_${TARGET_ARCH}\ddev"
            File /oname=ddev-hostname_linux "..\.gotmp\bin\linux_${TARGET_ARCH}\ddev-hostname"
            File /oname=mkcert_linux "..\.gotmp\bin\linux_${TARGET_ARCH}\mkcert"
            File /oname=ddev-hostname.exe "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"
            File /oname=mkcert.exe "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
            File /oname=mkcert_install.sh "scripts\mkcert_install.sh"
            File /oname=install_temp_sudoers.sh "scripts\install_temp_sudoers.sh"
            File /oname=check_root_user.sh "scripts\check_root_user.sh"
            File /oname=detect_docker_suite.sh "scripts\detect_docker_suite.sh"
            File /oname=detect_docker_family.sh "scripts\detect_docker_family.sh"
            File /oname=apt_install_with_log.sh "scripts\apt_install_with_log.sh"
            File /oname=ensure_systemd_enabled.sh "scripts\ensure_systemd_enabled.sh"
            File /oname=wait_for_systemd.sh "scripts\wait_for_systemd.sh"
        ${EndIf}

        ; Install icons
        SetOutPath "$INSTDIR\Icons"
        SetOverwrite try
        File /oname=ddev.ico "graphics\ddev-install.ico"

        ; Run mkcert.exe -install early for all installation types (needed for WSL2 setup)
        Call RunMkcertInstall

        ; Add DDEV installation directory to PATH (EnVar::AddValue handles duplicates)
        Push "Adding DDEV installation directory to user PATH..."
        Call LogPrint
        ReadRegStr $R0 HKCU "Environment" "Path"
        Push "PATH before addition: $R0"
        Call LogPrint

        EnVar::SetHKCU
        EnVar::AddValue "Path" "$INSTDIR"
        Pop $R1
        Push "EnVar::AddValue result: $R1"
        Call LogPrint
        
        Push "PATH addition completed with result: $R1"
        Call LogPrint

        ; Verify wsl.exe is accessible (critical for WSL operations)
        ${If} ${FileExists} "$WINDIR\System32\wsl.exe"
            Push "WSL executable found in System32"
            Call LogPrint
        ${Else}
            Push "WARNING: wsl.exe not found in System32 - file system redirection may be enabled"
            Call LogPrint
        ${EndIf}

        ${If} $INSTALL_OPTION == "traditional"
            Call InstallTraditionalWindows
        ${Else}
            Call InstallWSL2Common
        ${EndIf}

        ; Start Menu shortcuts removed for simplification
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
    
    ; Calculate and write estimated size for Add/Remove Programs
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "EstimatedSize" "$0"

    ; Create Start Menu shortcuts (for all installation types)
    CreateDirectory "$SMPROGRAMS\DDEV"
    CreateDirectory "$INSTDIR\Links"

    ; Documentation and website links (useful for all users)
    WriteIniStr "$INSTDIR\Links\DDEV Documentation.url" "InternetShortcut" "URL" "https://docs.ddev.com"
    CreateShortCut "$SMPROGRAMS\DDEV\DDEV Documentation.lnk" "$INSTDIR\Links\DDEV Documentation.url" "" "$INSTDIR\Icons\ddev.ico"

    WriteIniStr "$INSTDIR\Links\DDEV Website.url" "InternetShortcut" "URL" "https://ddev.com"
    CreateShortCut "$SMPROGRAMS\DDEV\DDEV Website.lnk" "$INSTDIR\Links\DDEV Website.url" "" "$INSTDIR\Icons\ddev.ico"

    ; Uninstall link opens Windows Settings Apps page (modern approach)
    CreateShortCut "$SMPROGRAMS\DDEV\Uninstall DDEV.lnk" "ms-settings:appsfeatures"
SectionEnd

Section Uninstall
    ; Uninstall mkcert if it was installed
    Call un.mkcertUninstall

    ; Clean up mkcert environment variables
    Call un.CleanupMkcertEnvironment

    ; Remove install directory from user PATH
    EnVar::SetHKCU
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

    ; Remove Start Menu shortcuts
    Delete "$SMPROGRAMS\DDEV\DDEV Terminal.lnk"
    Delete "$SMPROGRAMS\DDEV\DDEV Documentation.lnk"
    Delete "$SMPROGRAMS\DDEV\DDEV Website.lnk"
    Delete "$SMPROGRAMS\DDEV\Uninstall DDEV.lnk"
    RMDir "$SMPROGRAMS\DDEV"

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

Function GetDebianBasedDistros
    StrCpy $R0 ""  ; Result string

    Push "=== Starting GetDebianBasedDistros ==="
    Call LogPrint

    Push "Checking registry key HKCU\Software\Microsoft\Windows\CurrentVersion\Lxss..."
    Call LogPrint
    SetRegView 64
    ClearErrors
    EnumRegKey $R1 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss" 0
    ${If} ${Errors}
        Push "ERROR: Cannot access Lxss registry key - WSL may not be installed"
        Call LogPrint
        Push ""
        Return
    ${EndIf}
    Push "Registry key exists and is accessible"
    Call LogPrint

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
    Push "Found $R5 total WSL distributions"
    Call LogPrint

    ; Now enumerate and check each key
    StrCpy $R1 0   ; Reset index
    ${While} $R1 < $R5
        ClearErrors
        EnumRegKey $R2 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss" $R1
        ${If} ${Errors}
            Push "Error enumerating key at index $R1"
            Call LogPrint
            Goto next_key
        ${EndIf}

        ClearErrors
        ReadRegStr $R3 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss\$R2" "DistributionName"
        ${If} ${Errors}
            Push "Error reading DistributionName for key $R2"
            Call LogPrint
            Goto next_key
        ${EndIf}
        Push "Found distribution: $R3"
        Call LogPrint

        ; Check if Flavor is "ubuntu"
        ClearErrors
        ReadRegStr $R4 HKCU "Software\Microsoft\Windows\CurrentVersion\Lxss\$R2" "Flavor"
        ${If} ${Errors}
            Push "No Flavor field found for $R3 - falling back to name check"
            Call LogPrint
            ; No Flavor field - fall back to name check for backward compatibility
            StrCpy $R4 $R3 6
            Push "First 6 chars of '$R3': '$R4'"
            Call LogPrint
            ${If} $R4 == "Ubuntu"
            ${OrIf} $R4 == "Debian"
                Push "Found Debian-based distribution (name-based): $R3"
                Call LogPrint
                ${If} $R0 != ""
                    StrCpy $R0 "$R0|"
                ${EndIf}
                StrCpy $R0 "$R0$R3"
            ${Else}
                Push "Distribution '$R3' does not start with 'Ubuntu' or 'Debian' (starts with '$R4')"
                Call LogPrint
            ${EndIf}
        ${Else}
            Push "Found Flavor field for $R3: '$R4'"
            Call LogPrint
            ; Check if Flavor is a known Debian-based distro identifier
            ; (ubuntu, debian, kali, elxr — case-insensitive substring match).
            ; Note: $R5 holds the total distro count (loop bound), do not clobber.
            ; Use $R6 as a "matched" flag and $R7 as the per-test StrStr result.
            StrCpy $R6 ""
            ${StrStr} $R7 $R4 "ubuntu"
            ${If} $R7 != ""
                StrCpy $R6 "yes"
            ${EndIf}
            ${StrStr} $R7 $R4 "debian"
            ${If} $R7 != ""
                StrCpy $R6 "yes"
            ${EndIf}
            ${StrStr} $R7 $R4 "kali"
            ${If} $R7 != ""
                StrCpy $R6 "yes"
            ${EndIf}
            ${StrStr} $R7 $R4 "elxr"
            ${If} $R7 != ""
                StrCpy $R6 "yes"
            ${EndIf}
            ${If} $R6 == "yes"
                Push "Found Debian-based distribution (Flavor-based): $R3"
                Call LogPrint
                ${If} $R0 != ""
                    StrCpy $R0 "$R0|"
                ${EndIf}
                StrCpy $R0 "$R0$R3"
            ${Else}
                Push "Distribution '$R3' has Flavor '$R4' but does not contain 'ubuntu', 'debian', 'kali', or 'elxr'"
                Call LogPrint
            ${EndIf}
        ${EndIf}

        next_key:
        IntOp $R1 $R1 + 1
    ${EndWhile}

    Push "Registry enumeration complete. Final list: [$R0]"
    Call LogPrint
    Push $R0
FunctionEnd

Function InstallWSL2CommonSetup
    ; Note: WSL distros have already been enumerated from the registry and selected by the user.
    ; The distro type (Debian-based) has been verified from the registry Flavor field.
    ; Docker connectivity has already been validated with 'docker ps'.

    ; List WSL distros and versions (helpful for troubleshooting)
    Push "Listing WSL distributions and versions..."
    Call LogPrint
    nsExec::ExecToStack 'wsl.exe -l -v'
    Pop $R0
    Pop $R1
    ${If} $R0 == 0
        Push "WSL distros: $R1"
        Call LogPrint
    ${Else}
        Push "WARNING: Could not list WSL distros (exit code: $R0)"
        Call LogPrint
    ${EndIf}

    ; Verify selected distro is accessible
    Push "Verifying selected distro $SELECTED_DISTRO is accessible..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO echo "WSL connectivity test passed"'
    Pop $R0
    Pop $R1
    ${If} $R0 != 0
        Push "ERROR: Cannot access distro $SELECTED_DISTRO - exit code: $R0"
        Call LogPrint
        Push "Could not access the selected WSL distro. Please ensure it's working properly."
        Call ShowErrorAndAbort
    ${EndIf}
    Push "Selected distro $SELECTED_DISTRO is accessible"
    Call LogPrint

    ; Convert Windows temp path to WSL format manually
    ; Windows: C:\Users\username\AppData\Local\Temp -> WSL: /mnt/c/Users/username/AppData/Local/Temp
    Push "Converting Windows temp path to WSL format..."
    Call LogPrint
    Push "Windows TEMP: $TEMP"
    Call LogPrint

    ; Extract drive letter and path
    StrCpy $0 "$TEMP" 1  ; Get drive letter (e.g., "C")
    StrLen $1 "$TEMP"
    IntOp $1 $1 - 2  ; Length minus "C:"
    StrCpy $2 "$TEMP" $1 2  ; Get path after "C:"

    ; Convert drive letter to lowercase
    ${StrCase} $0 $0 "L"

    ; Replace backslashes with forward slashes
    ${StrRep} $3 $2 "\" "/"

    ; Construct WSL path: /mnt/{drive}/{path}
    StrCpy $WSL_WINDOWS_TEMP "/mnt/$0$3"

    Push "WSL temp path: $WSL_WINDOWS_TEMP"
    Call LogPrint

    ; Check that default user is not root (required for all installation types)
    Push "Checking for root user in selected distro..."
    Call LogPrint
    Call CheckRootUser
    Push "Root user check passed"
    Call LogPrint

    ; Install apt_install_with_log.sh helper into the distro once so the
    ; later apt-get install steps can surface the tail of apt's output
    ; (the NSIS output buffer truncates long apt traces mid-stream).
    Push "Installing apt_install_with_log.sh helper into $SELECTED_DISTRO..."
    Call LogPrint
    Push $SELECTED_DISTRO
    Push "apt_install_with_log.sh"
    Call InstallScriptToDistro
    Pop $R0

    ; Ensure systemd is enabled in this distro's /etc/wsl.conf so dockerd
    ; auto-starts and D-Bus is available (D-Bus is needed by
    ; docker-credential-secretservice and many other tools). Distros that
    ; ship with [boot] systemd=false cause Docker to fail
    ; silently and credential helpers to error with "Could not connect".
    Push "Ensuring systemd is enabled in $SELECTED_DISTRO's /etc/wsl.conf..."
    Call LogPrint
    Push $SELECTED_DISTRO
    Push "ensure_systemd_enabled.sh"
    Call InstallScriptToDistro
    Pop $R0
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/ensure_systemd_enabled.sh'
    Pop $R1
    Pop $R2
    Push "ensure_systemd_enabled.sh exit=$R1: $R2"
    Call LogPrint
    ${If} $R1 == 2
        Push "ERROR: could not write /etc/wsl.conf in $SELECTED_DISTRO: $R2"
        Call LogPrint
        Push "Could not enable systemd in /etc/wsl.conf in $SELECTED_DISTRO. Output: $R2"
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $R1 == 1
        ; systemd was just enabled — terminate the distro so it boots
        ; with systemd as PID 1, then bring it back up.
        Push "systemd was newly enabled; terminating $SELECTED_DISTRO so the change takes effect..."
        Call LogPrint
        nsExec::ExecToStack 'wsl.exe --terminate $SELECTED_DISTRO'
        Pop $R1
        Pop $R2
        Push "wsl --terminate exit=$R1: $R2"
        Call LogPrint
        ; Bring it back up. systemd takes a few seconds to settle. Use a
        ; helper script so we don't fight NSIS's $(...) language-string
        ; quoting rules. After --terminate, /tmp/* on Kali/eLxr is on
        ; the ext4 rootfs so previously-installed helpers persist, but
        ; reinstall to be safe.
        Push "Restarting $SELECTED_DISTRO with systemd..."
        Call LogPrint
        Push $SELECTED_DISTRO
        Push "apt_install_with_log.sh"
        Call InstallScriptToDistro
        Pop $R0
        Push $SELECTED_DISTRO
        Push "wait_for_systemd.sh"
        Call InstallScriptToDistro
        Pop $R0
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/wait_for_systemd.sh 20'
        Pop $R1
        Pop $R2
        Push "systemd readiness check exit=$R1: $R2"
        Call LogPrint
        ${If} $R1 != 0
            Push "WARNING: systemd did not report ready within 20s; continuing anyway. Output: $R2"
            Call LogPrint
        ${EndIf}
    ${EndIf}

    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Make sure we're not running docker-ce or docker.io daemon (conflicts with Docker Desktop)
        Push "Verifying Docker installation type..."
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO pgrep dockerd'
        Pop $1
        Pop $0
        ${If} $1 == 0
            Push "ERROR: Local Docker daemon detected in WSL2 - conflicts with Docker Desktop. Process list: $0"
            Call LogPrint
            Push "A local Docker daemon (from docker-ce or docker.io) is running in WSL2. This conflicts with Docker Desktop. Please remove Docker first ('sudo apt-get remove docker-ce' or 'sudo apt-get remove docker.io')."
            Call ShowErrorAndAbort
        ${EndIf}
    ${EndIf}

    ; Remove old Docker versions first (per Docker CE installation instructions)
    Push "WSL($SELECTED_DISTRO): Removing old Docker packages if present..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1 || true"'
    Pop $1
    Pop $0
    ; Note: This command is allowed to fail if packages aren't installed

    ; apt-get update
    Push "WSL($SELECTED_DISTRO): Updating package database using apt-get update..."
    Call LogPrint
    Push "Please be patient - updating package database..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root apt-get update'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "WARNING: apt-get update returned non-zero exit code: $1, output: $0"
        Call LogPrint
        ; Continue anyway - update warnings are often non-fatal
    ${EndIf}

    ; Install linux packages
    Push "WSL($SELECTED_DISTRO): Installing required linux packages..."
    Call LogPrint
    Push "Please be patient - installing required linux packages..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh prereq ca-certificates curl gnupg libsecret-1-0 lsb-release'
    Pop $1
    Pop $0
    ; Optionally try to install 'pass' (used by docker-credential-pass);
    ; not all minimal Debian derivatives (e.g. eLxr) carry it. A failure
    ; here is a warning, not fatal.
    Push "WSL($SELECTED_DISTRO): Trying optional package 'pass'..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh prereq-optional pass'
    Pop $R1
    Pop $R2
    ${If} $R1 != 0
        Push "WARNING: optional package 'pass' not installed (exit=$R1): $R2"
        Call LogPrint
    ${EndIf}
    ${If} $1 != 0
        Push "ERROR: Failed to apt-get install - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to apt-get install. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Create keyrings directory if it doesn't exist
    Push "WSL($SELECTED_DISTRO): Setting up keyrings directory..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root install -m 0755 -d /etc/apt/keyrings'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to create keyrings directory - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to create keyrings directory. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Detect distro family for Docker repository selection (ubuntu vs debian)
    Push "WSL($SELECTED_DISTRO): Detecting distro family for Docker repository..."
    Call LogPrint
    Push $SELECTED_DISTRO
    Push "detect_docker_family.sh"
    Call InstallScriptToDistro
    Pop $R0
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/detect_docker_family.sh'
    Pop $1
    Pop $DOCKER_DISTRO_FAMILY
    ${If} $DOCKER_DISTRO_FAMILY == ""
        StrCpy $DOCKER_DISTRO_FAMILY "debian"
    ${EndIf}
    Push "WSL($SELECTED_DISTRO): Using Docker repository for distro family: $DOCKER_DISTRO_FAMILY"
    Call LogPrint

    ; Detect Docker suite codename - handles Kali and other derivatives
    ; whose VERSION_CODENAME is not a valid Docker repo suite
    Push "WSL($SELECTED_DISTRO): Detecting Docker suite codename..."
    Call LogPrint
    Push $SELECTED_DISTRO
    Push "detect_docker_suite.sh"
    Call InstallScriptToDistro
    Pop $R0
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/detect_docker_suite.sh'
    Pop $1
    Pop $DOCKER_SUITE
    ${If} $DOCKER_SUITE == ""
        StrCpy $DOCKER_SUITE "bookworm"
    ${EndIf}
    Push "WSL($SELECTED_DISTRO): Using Docker suite: $DOCKER_SUITE"
    Call LogPrint

    ; Clean up old Docker repository files if present
    Push "WSL($SELECTED_DISTRO): Removing old Docker repository files if present..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg /etc/apt/sources.list.d/docker.list"'
    Pop $1
    Pop $0

    ; Add Docker GPG key
    Push "WSL($SELECTED_DISTRO): Adding Docker repository key..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "curl -fsSL https://download.docker.com/linux/$DOCKER_DISTRO_FAMILY/gpg -o /etc/apt/keyrings/docker.asc && chmod a+r /etc/apt/keyrings/docker.asc"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add Docker repository key - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add Docker apt repository key. Please check your internet connection. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add Docker repository in deb822 format
    Push "WSL($SELECTED_DISTRO): Adding Docker apt repository..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "printf \"Types: deb\nURIs: https://download.docker.com/linux/$DOCKER_DISTRO_FAMILY\nSuites: $DOCKER_SUITE\nComponents: stable\nSigned-By: /etc/apt/keyrings/docker.asc\n\" > /etc/apt/sources.list.d/docker.sources"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add Docker repository - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add Docker repository. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Clean up old DDEV repository files if present
    Push "WSL($SELECTED_DISTRO): Removing old DDEV repository files if present..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "rm -f /etc/apt/keyrings/ddev.gpg /etc/apt/sources.list.d/ddev.list"'
    Pop $1
    Pop $0

    ; Add DDEV GPG key
    Push "WSL($SELECTED_DISTRO): Adding DDEV apt repository key..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key -o /etc/apt/keyrings/ddev.asc && chmod a+r /etc/apt/keyrings/ddev.asc"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add DDEV repository key - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add DDEV repository key. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add DDEV repository in deb822 format
    Push "WSL($SELECTED_DISTRO): Adding DDEV apt repository..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "printf \"Types: deb\nURIs: https://pkg.ddev.com/apt/\nSuites: *\nComponents: *\nSigned-By: /etc/apt/keyrings/ddev.asc\n\" > /etc/apt/sources.list.d/ddev.sources"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add DDEV repository - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add DDEV repository. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Update package lists
    Push "WSL($SELECTED_DISTRO): apt-get update..."
    Call LogPrint
    Push "Please be patient - updating package database..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1 || true"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: apt-get update failed - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to apt-get update. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}
FunctionEnd

Function InstallWSL2Common
    Push "Starting WSL2 Docker installation for $SELECTED_DISTRO"
    Call LogPrint

    ; Skip pre-installation status tracking commands that use bash -c
    ; These were failing on some systems and are not critical for installation
    Push "Skipping pre-installation status file setup (not critical for install)"
    Call LogPrint

    Call InstallWSL2CommonSetup

    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Install packages needed for Docker Desktop (including ddev)
        StrCpy $0 "docker-ce-cli ddev"
    ${Else}
        ; Install full Docker CE packages (including ddev)
        StrCpy $0 "docker-ce docker-ce-cli containerd.io ddev"
    ${EndIf}
    
    ; Update status
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo \"PROGRESS: Installing essential packages\" >> /tmp/ddev_installation_status.txt"'
    Pop $1
    Pop $2

    ; Install packages in multiple steps for better progress feedback
    Push "WSL($SELECTED_DISTRO): Installing essential packages (1/3)..."
    Call LogPrint
    Push "Please be patient - installing essential packages..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh essential ca-certificates curl gnupg libsecret-1-0 lsb-release'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install essential packages - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install essential packages. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    ; Optionally retry 'pass' here too (idempotent if already installed).
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh essential-optional pass'
    Pop $R1
    Pop $R2
    ${If} $R1 != 0
        Push "WARNING: optional package 'pass' not installed (exit=$R1): $R2"
        Call LogPrint
    ${EndIf}
    
    ; Update status
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo \"PROGRESS: Installing Docker components\" >> /tmp/ddev_installation_status.txt"'
    Pop $1
    Pop $2

    Push "WSL($SELECTED_DISTRO): Installing Docker components (2/3)..."
    Call LogPrint
    Push "Please be patient - installing Docker components..."
    Call LogPrint
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh docker docker-ce docker-ce-cli containerd.io'
    ${Else}
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/apt_install_with_log.sh docker-cli docker-ce-cli'
    ${EndIf}
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install Docker components - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install Docker components. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Update status
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo \"PROGRESS: Installing DDEV\" >> /tmp/ddev_installation_status.txt"'
    Pop $1
    Pop $2

    Push "WSL($SELECTED_DISTRO): Installing DDEV (3/3)..."
    Call LogPrint
    Push "Please be patient - installing DDEV..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root env APT_EXTRA_ARGS=--no-install-recommends bash /tmp/apt_install_with_log.sh ddev ddev ddev-wsl2'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install DDEV - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install DDEV. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Overwrite the installed DDEV binary with the bundled version
    Push "WSL($SELECTED_DISTRO): Overwriting DDEV binaries with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "$WSL_WINDOWS_TEMP/ddev_installer/ddev_linux" /usr/bin/ddev'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: DDEV binaries overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite DDEV binaries. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Make it executable
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root chmod +x /usr/bin/ddev'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to make DDEV binary executable - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to make DDEV binary executable. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Overwrite the installed ddev-hostname binary with the bundled version
    Push "WSL($SELECTED_DISTRO): Overwriting ddev-hostname binary with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "$WSL_WINDOWS_TEMP/ddev_installer/ddev-hostname_linux" /usr/bin/ddev-hostname'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: ddev-hostname binary overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite ddev-hostname binary. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Make ddev-hostname executable
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root chmod +x /usr/bin/ddev-hostname'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to make ddev-hostname binary executable - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to make ddev-hostname binary executable. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Overwrite the installed mkcert binary with the bundled version
    Push "WSL($SELECTED_DISTRO): Overwriting mkcert binary with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "$WSL_WINDOWS_TEMP/ddev_installer/mkcert_linux" /usr/bin/mkcert'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: mkcert binary overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite mkcert binary. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Make mkcert executable
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root chmod +x /usr/bin/mkcert'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to make mkcert binary executable - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to make mkcert binary executable. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Overwrite ddev-hostname.exe in WSL2 /usr/bin
    Push "WSL($SELECTED_DISTRO): Overwriting ddev-hostname.exe with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "$WSL_WINDOWS_TEMP/ddev_installer/ddev-hostname.exe" /usr/bin/ddev-hostname.exe'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: ddev-hostname.exe overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite ddev-hostname.exe. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Overwrite mkcert.exe in WSL2 /usr/bin
    Push "WSL($SELECTED_DISTRO): Overwriting mkcert.exe with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "$WSL_WINDOWS_TEMP/ddev_installer/mkcert.exe" /usr/bin/mkcert.exe'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: mkcert.exe overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite mkcert.exe. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add the unprivileged user to the docker group for docker-ce installation
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        Push "WSL($SELECTED_DISTRO): Getting username of unprivileged user..."
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO whoami'
        Pop $1
        Pop $2
        ${If} $1 != 0
            Push "ERROR: Failed to get WSL2 username - exit code: $1, output: $2"
            Call LogPrint
            Push "Failed to get WSL2 username. Error: $2"
            Call ShowErrorAndAbort
        ${EndIf}
        
        ; Trim whitespace from username
        Push $2
        Call TrimNewline
        Pop $2
        
        Push "WSL($SELECTED_DISTRO): Adding user '$2' to docker group..."
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root usermod -aG docker $2'
        Pop $1
        Pop $3
        ${If} $1 != 0
            Push "Warning: Failed to add user to docker group. Error: $3"
            Call LogPrint
            MessageBox MB_ICONEXCLAMATION|MB_OK "Warning: Failed to add user '$2' to docker group. You may need to run 'sudo usermod -aG docker $2' manually in WSL2."
        ${Else}
            Push "Successfully added user '$2' to docker group."
            Call LogPrint
        ${EndIf}
    ${EndIf}

    ; Verify the Docker daemon is running. We earlier ensured systemd is
    ; enabled in /etc/wsl.conf (and restarted the distro if it wasn't),
    ; so dockerd should already be running via the docker.service unit
    ; that docker-ce's postinst enabled. If it isn't yet (race with the
    ; service starting), nudge it via systemctl and poll docker info.
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        Push "WSL($SELECTED_DISTRO): Verifying Docker daemon via systemd..."
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root systemctl start docker.service'
        Pop $1
        Pop $0
        Push "systemctl start docker.service exit=$1: $0"
        Call LogPrint
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root sh -c "for i in 1 2 3 4 5 6 7 8 9 10; do docker info >/dev/null 2>&1 && exit 0; sleep 1; done; exit 1"'
        Pop $1
        Pop $0
        ${If} $1 != 0
            Push "WARNING: Docker daemon did not become ready within 10s. Output: $0"
            Call LogPrint
        ${Else}
            Push "Docker daemon is ready."
            Call LogPrint
        ${EndIf}
    ${EndIf}

    ; Show DDEV version
    Push "Verifying DDEV installation with 'ddev version'..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO ddev version'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: DDEV version check failed - exit code: $1, output: $0"
        Call LogPrint
        Push "WSL($SELECTED_DISTRO) doesn't seem to have working 'ddev version'. Please execute it manually in $SELECTED_DISTRO to debug the problem. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Set up mkcert in WSL2 if we're doing WSL2 installation
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
    ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        Call SetupMkcertInWSL2
    ${EndIf}

    ; Configure WSL2 security settings to prevent Windows security warnings
    Push "Configuring WSL2 security settings to prevent Windows executable warnings..."
    Call LogPrint
    
    ; Configure WSL2 security settings directly via registry
    Push "Configuring WSL2 security settings..."
    Call LogPrint
    
    ; Try to add wsl.localhost to Local Intranet zone via registry
    Push "Adding file://*.wsl.localhost to Local Intranet security zone..."
    Call LogPrint
    nsExec::ExecToStack 'reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\ZoneMap\Domains\wsl.localhost" /v "file" /t REG_DWORD /d 1 /f'
    Pop $R0
    Pop $R1
    ${If} $R0 == 0
        Push "WSL2 ZoneMap Internet Domains settings configured successfully via registry"
        Call LogPrint
    ${Else}
        ; Fallback: Show manual instructions
        Push "Could not automatically configure WSL2 security settings."
        Call LogPrint
        Push "To resolve Windows internet zone security warnings manually:"
        Call LogPrint
        Push "1. Open Internet Options (Control Panel > Internet Options)"
        Call LogPrint
        Push "2. Go to Security tab > Local Intranet > Sites > Advanced"
        Call LogPrint
        Push "3. Add to the zone: \\wsl.localhost"
        Call LogPrint
        Push "4. Click OK to save"
        Call LogPrint

        ; Show message box with manual instructions
        MessageBox MB_OK "WSL2 Security Configuration Required$\r$\n$\r$\nCould not automatically configure WSL2 security settings.$\r$\nTo resolve Windows security warnings manually:$\r$\n$\r$\n1. Open Internet Options (Control Panel > Internet Options)$\r$\n2. Go to Security tab > Local Intranet > Sites > Advanced$\r$\n3. Add to the zone: \\wsl.localhost$\r$\n4. Click OK to save"
    ${EndIf}

    ; Mark installation as complete for external monitoring
    Push "Marking installation as complete..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo \"COMPLETED: Installation completed successfully\" >> /tmp/ddev_installation_status.txt"'
    Pop $1
    Pop $2
    Push "DDEV installation completed successfully"
    Call LogPrint

    ; Clean up temp directory
    Push "Cleaning up temporary files..."
    Call LogPrint
    Delete "$WINDOWS_TEMP\ddev_installer\check_root_user.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\install_temp_sudoers.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\mkcert_install.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\detect_docker_suite.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\detect_docker_family.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\apt_install_with_log.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\ensure_systemd_enabled.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\wait_for_systemd.sh"
    Delete "$WINDOWS_TEMP\ddev_installer\ddev_linux"
    Delete "$WINDOWS_TEMP\ddev_installer\ddev-hostname_linux"
    Delete "$WINDOWS_TEMP\ddev_installer\mkcert_linux"
    Delete "$WINDOWS_TEMP\ddev_installer\ddev-hostname.exe"
    Delete "$WINDOWS_TEMP\ddev_installer\mkcert.exe"
    RMDir "$WINDOWS_TEMP\ddev_installer"
    
    ; Leave installation status file for external monitoring
    ; This will be cleaned up by external tests or on next installation
    
    Push "All done! Installation completed successfully and validated."
    Call LogPrint

FunctionEnd

Function CheckGitForWindows
    Push "Checking for Git for Windows..."
    Call LogPrint
    
    ; Check if Git for Windows is installed in the standard location
    ${If} ${FileExists} "$PROGRAMFILES64\Git\bin\git.exe"
        ${If} ${FileExists} "$PROGRAMFILES64\Git\bin\bash.exe"
            Push "Git for Windows found in Program Files"
            Call LogPrint
            ; Verify it's working by checking version
            nsExec::ExecToStack '"$PROGRAMFILES64\Git\bin\git.exe" --version'
            Pop $R0
            Pop $R1
            ${If} $R0 == 0
                Push "Git version check successful: $R1"
                Call LogPrint
                ; Check if it contains "windows" to confirm it's Git for Windows
                ${StrStr} $R2 $R1 "windows"
                ${If} $R2 != ""
                    Push "Confirmed Git for Windows installation"
                    Call LogPrint
                    Push "1"
                    Return
                ${EndIf}
            ${EndIf}
        ${EndIf}
    ${EndIf}
    
    ; Also check if git and bash are available in PATH
    nsExec::ExecToStack 'git --version'
    Pop $R0
    Pop $R1
    ${If} $R0 == 0
        ${StrStr} $R2 $R1 "windows"
        ${If} $R2 != ""
            ; Check if bash is also available
            nsExec::ExecToStack 'bash --version'
            Pop $R3
            Pop $R4
            ${If} $R3 == 0
                Push "Git for Windows found in PATH: $R1"
                Call LogPrint
                Push "1"
                Return
            ${EndIf}
        ${EndIf}
    ${EndIf}
    
    Push "Git for Windows not found"
    Call LogPrint
    Push "0"
FunctionEnd

Function InstallTraditionalWindows
    Push "Starting InstallTraditionalWindows"
    Call LogPrint

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

    ; Create Start Menu folder first (must exist before creating shortcuts)
    CreateDirectory "$SMPROGRAMS\DDEV"

    ; Create Start Menu shortcuts for Traditional Windows
    ; DDEV Terminal - prefer Git Bash if available, fallback to PowerShell
    ${If} ${FileExists} "$PROGRAMFILES64\Git\bin\bash.exe"
        ; Git Bash found - create shortcut that opens Git Bash
        CreateShortCut "$SMPROGRAMS\DDEV\DDEV Terminal.lnk" "$PROGRAMFILES64\Git\bin\bash.exe" '--login -i' "$INSTDIR\Icons\ddev.ico"
        Push "Created DDEV Terminal shortcut using Git Bash"
        Call LogPrint
    ${Else}
        ; Fallback to PowerShell
        CreateShortCut "$SMPROGRAMS\DDEV\DDEV Terminal.lnk" "powershell.exe" "" "$INSTDIR\Icons\ddev.ico"
        Push "Created DDEV Terminal shortcut using PowerShell (Git Bash not found)"
        Call LogPrint
    ${EndIf}

    ; Verify installation completed by checking ddev.exe exists
    ; This ensures filesystem writes are complete before installer exits
    Push "Verifying installation files..."
    Call LogPrint

    StrCpy $R0 0  ; retry counter
    ${Do}
        ${If} ${FileExists} "$INSTDIR\ddev.exe"
            Push "Verified: ddev.exe exists at $INSTDIR\ddev.exe"
            Call LogPrint
            ${ExitDo}
        ${EndIf}
        IntOp $R0 $R0 + 1
        ${If} $R0 > 10
            Push "WARNING: ddev.exe verification failed after 10 retries"
            Call LogPrint
            ${ExitDo}
        ${EndIf}
        Sleep 500  ; Wait 500ms before retry
    ${Loop}

    Push "Traditional Windows installation completed."
    Call LogPrint

FunctionEnd

Function RunMkcertInstall
    ${If} ${Silent}
        ; In silent mode, skip mkcert.exe -install because it pops up a dialog window
        ; But still set up CAROOT environment variable
        Push "Setting up CAROOT environment variable in silent mode..."
        Call LogPrint
        Call SetupWindowsCAROOT
        Return
    ${EndIf}
    
    Push "Setting up mkcert.exe (Windows) for trusted HTTPS certificates..."
    Call LogPrint

    ; Unset CAROOT environment variable in current process
    System::Call 'kernel32::SetEnvironmentVariable(t "CAROOT", i 0) i .r0'

    ; Run mkcert.exe -install to create fresh certificate authority
    Push "Running mkcert.exe -install to create certificate authority..."
    Call LogPrint
    nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -install'
    Pop $R0
    Pop $R1 ; Output
    ${If} $R0 = 0
        Push "mkcert.exe -install completed successfully"
        Call LogPrint
        WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup" 1
        
        ; Set up CAROOT environment variable for WSL2 sharing (only used in WSL2 installs)
        Call SetupWindowsCAROOT
    ${Else}
        Push "mkcert.exe -install failed with exit code: $R0"
        Call LogPrint
        MessageBox MB_ICONEXCLAMATION|MB_OK "mkcert -install failed with exit code: $R0. Output: $R1. You may need to run 'mkcert.exe -install' manually on Windows."
    ${EndIf}
FunctionEnd

Function SetupWindowsCAROOT
    Push "Setting up mkcert CAROOT environment variable..."
    Call LogPrint

    ; Get the CAROOT directory from mkcert (mkcert -install already completed)
    nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -CAROOT'
    Pop $R0 ; error code
    Pop $R1 ; output (CAROOT path)

    ${If} $R0 = 0
        ; Trim whitespace from CAROOT path
        Push $R1
        Call TrimNewline
        Pop $R1

        Push "CAROOT directory: $R1"
        Call LogPrint

        ; Set CAROOT environment variable using EnVar plugin
        EnVar::SetHKCU
        EnVar::Delete "CAROOT"  ; Remove entire variable first
        Pop $0  ; Get error code from Delete
        Push "EnVar::Delete CAROOT result: $0"
        Call LogPrint

        EnVar::AddValue "CAROOT" "$R1"
        Pop $0  ; Get error code from AddValue
        Push "EnVar::AddValue CAROOT result: $0"
        Call LogPrint

        ; Only set up WSLENV for WSL2 installations (not traditional Windows)
        ${If} $INSTALL_OPTION == "traditional"
            Push "Skipping WSLENV setup for traditional Windows installation"
            Call LogPrint
            Push "CAROOT environment variable configured successfully."
            Call LogPrint
        ${Else}
            ; Get current WSLENV value from registry
            ReadRegStr $R2 HKCU "Environment" "WSLENV"
            ${If} ${Errors}
                StrCpy $R2 ""
            ${EndIf}

            ; Store original value for debugging
            StrCpy $R4 $R2

            ; Clean up any existing CAROOT/up entries first (handles both : and ; separators)
            StrCpy $R3 $R2  ; Copy to working variable

            ; Remove all instances of CAROOT/up with colon separator (correct WSLENV format)
            ${StrRep} $R3 $R3 "CAROOT/up:" ""
            ${StrRep} $R3 $R3 ":CAROOT/up" ""

            ; Remove all instances of CAROOT/up with semicolon separator (legacy/incorrect format)
            ${StrRep} $R3 $R3 "CAROOT/up;" ""
            ${StrRep} $R3 $R3 ";CAROOT/up" ""

            ; Remove if it's exactly "CAROOT/up" by itself
            ${If} $R3 == "CAROOT/up"
                StrCpy $R3 ""
            ${EndIf}

            ; Clean up any double separators that might have been created
            ${StrRep} $R3 $R3 "::" ":"
            ${StrRep} $R3 $R3 ";;" ";"

            ; Remove leading or trailing separators (: or ;)
            ${If} $R3 != ""
                StrCpy $R5 $R3 1  ; Get first character
                ${If} $R5 == ":"
                    StrCpy $R3 $R3 "" 1  ; Remove first character
                ${EndIf}
                ${If} $R5 == ";"
                    StrCpy $R3 $R3 "" 1  ; Remove first character
                ${EndIf}
                ${If} $R3 != ""
                    StrLen $R6 $R3
                    IntOp $R6 $R6 - 1
                    StrCpy $R5 $R3 1 $R6  ; Get last character
                    ${If} $R5 == ":"
                        StrCpy $R3 $R3 $R6  ; Remove last character
                    ${EndIf}
                    ${If} $R5 == ";"
                        StrCpy $R3 $R3 $R6  ; Remove last character
                    ${EndIf}
                ${EndIf}
            ${EndIf}

            ; Now add CAROOT/up to the cleaned string using colon separator (WSLENV standard)
            ${If} $R3 != ""
                StrCpy $R2 "$R3:CAROOT/up"
            ${Else}
                StrCpy $R2 "CAROOT/up"
            ${EndIf}

            Push "WSLENV cleaned and updated: [$R4] -> [$R2]"
            Call LogPrint

            EnVar::SetHKCU
            EnVar::Delete "WSLENV"  ; Remove existing WSLENV entirely
            Pop $0  ; Get error code from Delete

            EnVar::AddValue "WSLENV" "$R2"
            Pop $0  ; Get error code from AddValue

            ; Verify by reading back from registry and validate
            ReadRegStr $R5 HKCU "Environment" "WSLENV"

            Push "WSLENV after update: [$R5]"
            Call LogPrint

            ; Validate: WSLENV must contain CAROOT and must not contain semicolons
            ${StrStr} $R6 $R5 "CAROOT"
            ${If} $R6 == ""
                Push "WARNING: WSLENV does not contain CAROOT after update - WSL2 certificate sharing may not work"
                Call LogPrint
                ${IfNot} ${Silent}
                    MessageBox MB_ICONEXCLAMATION|MB_OK "Warning: WSLENV was not set correctly. WSL2 certificate sharing may not work. Check WSLENV in HKCU\Environment."
                ${EndIf}
            ${EndIf}
            ${StrStr} $R6 $R5 ";"
            ${If} $R6 != ""
                Push "WARNING: WSLENV contains semicolons which are not valid WSLENV separators: [$R5]"
                Call LogPrint
                ${IfNot} ${Silent}
                    MessageBox MB_ICONEXCLAMATION|MB_OK "Warning: WSLENV contains invalid semicolons. CAROOT may not propagate to WSL2. Current value: $R5"
                ${EndIf}
            ${EndIf}

            ; Validate: CAROOT must be non-empty
            ReadRegStr $R6 HKCU "Environment" "CAROOT"
            ${If} $R6 == ""
                Push "WARNING: CAROOT is empty after update - WSL2 certificate sharing may not work"
                Call LogPrint
            ${Else}
                Push "CAROOT validated: [$R6]"
                Call LogPrint
            ${EndIf}

            Push "mkcert certificate sharing with WSL2 configured successfully."
            Call LogPrint
        ${EndIf}
    ${Else}
        Push "Failed to get CAROOT directory from mkcert"
        Call LogPrint
        ${IfNot} ${Silent}
            MessageBox MB_ICONEXCLAMATION|MB_OK "Failed to get CAROOT directory from mkcert. WSL2 certificate sharing may not work properly."
        ${EndIf}
    ${EndIf}
FunctionEnd

Function SetupMkcertInWSL2
    Push "Setting up mkcert inside WSL2 distro: $SELECTED_DISTRO"
    Call LogPrint
    
    ; Check current Windows CAROOT environment variable from registry
    ReadRegStr $R2 HKCU "Environment" "CAROOT"
    StrCpy $WINDOWS_CAROOT $R2  ; Save to global variable for later use
    Push "Windows CAROOT environment variable: $WINDOWS_CAROOT"
    Call LogPrint

    ; Check current Windows WSLENV environment variable from registry
    ReadRegStr $R3 HKCU "Environment" "WSLENV"
    ; DetailPrint "Windows WSLENV environment variable: $R3"
    
    ; Install and run temporary sudoers script
    Push $SELECTED_DISTRO
    Push "install_temp_sudoers.sh"
    Call InstallScriptToDistro
    Pop $R4
    ${If} $R4 != 0
        Push "Failed to install temporary sudoers script"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "Failed to install temporary sudoers script to WSL2 distro"
        Abort
    ${EndIf}
    
    Push "Running temporary sudoers installation..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash /tmp/install_temp_sudoers.sh'
    Pop $R4
    Pop $R5
    ${If} $R4 != 0
        Push "Failed to create temporary sudoers: $R5"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "Failed to create temporary sudoers entry: $R5"
        Abort
    ${Else}
        Push "Temporary sudoers created successfully"
        Call LogPrint
    ${EndIf}
    
    ; Install mkcert_install.sh check script to WSL2 distro
    ; We use this, which consumes WINDOWS_CAROOT, because wsl commands issued
    ; from installer don't get the CAROOT environment variable.
    Push $SELECTED_DISTRO
    Push "mkcert_install.sh"
    Call InstallScriptToDistro
    Pop $R8  ; Check result
    ${If} $R8 != 0
        Push "Failed to install mkcert_install.sh script"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "Failed to mkcert_install.sh script to WSL2 distro"
        Abort
    ${EndIf}
    
    Push "Running /tmp/mkcert_install.sh in WSL2 distro: $SELECTED_DISTRO"
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "WINDOWS_CAROOT=\"$WINDOWS_CAROOT\" /tmp/mkcert_install.sh"'
    Pop $R0
    Pop $R1
    Push "mkcert_install.sh script exit code: $R0"
    Call LogPrint
    Push "mkcert_install.sh output: '$R1'"
    Call LogPrint

    ; Remove temporary passwordless sudo
    Push "Removing temporary passwordless sudo..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root rm -f /etc/sudoers.d/temp-mkcert-install'
    Pop $R6
    Pop $R7
    ${If} $R6 != 0
        Push "Warning: Failed to remove temporary sudoers entry: $R7"
        Call LogPrint
    ${Else}
        Push "Temporary sudoers entry removed successfully"
        Call LogPrint
    ${EndIf}

FunctionEnd

Function un.onInit
  ${IfNot} ${Silent}
    MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Are you sure you want to completely remove $(^Name) and all of its components?" IDYES DoUninstall
    Abort
  ${EndIf}

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
    Push "Command line parameters: $R0"
    Call LogPrint
    
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
        Push "Found /docker-ce argument"
        Call LogPrint
    ${EndIf}
    
    ; Check for /docker-desktop argument
    ${GetOptions} $R0 "/docker-desktop" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "wsl2-docker-desktop"
        Push "Found /docker-desktop argument"
        Call LogPrint
    ${EndIf}
    
    ; Check for /rancher-desktop argument
    ${GetOptions} $R0 "/rancher-desktop" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "wsl2-docker-desktop"
        Push "Found /rancher-desktop argument"
        Call LogPrint
    ${EndIf}
    
    ; Check for /traditional argument
    ${GetOptions} $R0 "/traditional" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_INSTALL_TYPE "traditional"
        Push "Found /traditional argument"
        Call LogPrint
    ${EndIf}
    
    ; Check for /distro argument
    ${GetOptions} $R0 "/distro=" $R1
    ${IfNot} ${Errors}
        StrCpy $SILENT_DISTRO $R1
        Push "Found /distro argument: $SILENT_DISTRO"
        Call LogPrint
    ${EndIf}
    
    ; Validate that distro is specified for WSL2 installation types
    ${If} $SILENT_INSTALL_TYPE == "wsl2-docker-ce"
    ${OrIf} $SILENT_INSTALL_TYPE == "wsl2-docker-desktop"
        ${If} $SILENT_DISTRO == ""
            Push "ERROR: Missing required /distro argument for WSL2 installation type: $SILENT_INSTALL_TYPE"
            Call LogPrint
            MessageBox MB_ICONSTOP|MB_OK "The /distro=<distro_name> argument is required when using /docker-ce, /docker-desktop, or /rancher-desktop.$\n$\nExample: installer.exe /docker-ce /distro=Ubuntu-22.04"
            Abort
        ${EndIf}
    ${EndIf}
    
    ; If any install type was specified via command line, enable silent mode
    ${If} $SILENT_INSTALL_TYPE != ""
        ${IfNot} ${Silent}
            SetSilent silent
            Push "Command line install type detected, enabling silent mode"
            Call LogPrint
        ${EndIf}
    ${EndIf}
FunctionEnd

; CheckOldSystemInstallation - Detect and offer to remove old system-wide DDEV installation
; Old versions installed to $PROGRAMFILES\DDEV; new versions install to $LOCALAPPDATA\Programs\DDEV
Function CheckOldSystemInstallation
    Push $R0
    Push $R1

    ; Check for old system-wide installation in Program Files
    ${If} ${FileExists} "$PROGRAMFILES64\DDEV\ddev.exe"
        Push "Found old system-wide DDEV installation at $PROGRAMFILES64\DDEV"
        Call LogPrint

        ; Check if uninstaller exists
        ${If} ${FileExists} "$PROGRAMFILES64\DDEV\ddev_uninstall.exe"
            Push "Old uninstaller found at $PROGRAMFILES64\DDEV\ddev_uninstall.exe"
            Call LogPrint

            ${IfNot} ${Silent}
                MessageBox MB_YESNO|MB_ICONQUESTION "An old system-wide DDEV installation was found at:$\n$PROGRAMFILES64\DDEV$\n$\nDDEV now installs per-user to avoid administrator account issues.$\n$\nWould you like to remove the old installation?$\n(Requires administrator privileges)" IDYES remove_old IDNO skip_old
            ${Else}
                ; In silent mode, try to remove automatically
                Goto remove_old
            ${EndIf}
            Goto skip_old

            remove_old:
                Push "Attempting to remove old system-wide installation..."
                Call LogPrint
                ; Run the old uninstaller - it will request elevation if needed
                ExecWait '"$PROGRAMFILES64\DDEV\ddev_uninstall.exe" /S' $R0
                ${If} $R0 == 0
                    Push "Old installation removed successfully"
                    Call LogPrint
                ${Else}
                    Push "Old uninstaller returned code: $R0"
                    Call LogPrint
                    ${IfNot} ${Silent}
                        MessageBox MB_OK|MB_ICONINFORMATION "The old installation could not be fully removed (you may have cancelled the elevation prompt).$\n$\nYou can manually uninstall it later from Programs and Features.$\n$\nContinuing with new installation..."
                    ${EndIf}
                ${EndIf}

            skip_old:
        ${Else}
            ; No uninstaller found - just warn the user
            Push "Old installation found but no uninstaller present"
            Call LogPrint
            ${IfNot} ${Silent}
                MessageBox MB_OK|MB_ICONINFORMATION "An old DDEV installation was found at:$\n$PROGRAMFILES64\DDEV$\n$\nNo uninstaller was found. You may need to manually remove this directory and clean up the system PATH.$\n$\nContinuing with new installation..."
            ${EndIf}
        ${EndIf}
    ${EndIf}

    Pop $R1
    Pop $R0
FunctionEnd

Function .onInit
    ; Set proper 64-bit handling
    SetRegView 64
    ${DisableX64FSRedirection}

    ; Get Windows TEMP environment variable
    ReadEnvStr $WINDOWS_TEMP "TEMP"

    ; Check that this installer matches the system architecture
    ; TARGET_ARCH is set at compile time to either "amd64" or "arm64"
    !if "${TARGET_ARCH}" == "amd64"
        ${IfNot} ${IsNativeAMD64}
            MessageBox MB_ICONSTOP|MB_OK "This installer is for AMD64 (x86-64) Windows systems.$\n$\nYour system appears to be ARM64. Please download the ARM64 installer instead:$\n${RELEASES_URL}"
            Abort
        ${EndIf}
    !else if "${TARGET_ARCH}" == "arm64"
        ${IfNot} ${IsNativeARM64}
            MessageBox MB_ICONSTOP|MB_OK "This installer is for ARM64 Windows systems.$\n$\nYour system appears to be AMD64 (x86-64). Please download the AMD64 installer instead:$\n${RELEASES_URL}"
            Abort
        ${EndIf}
    !endif

    ; Initialize directory to per-user location
    ${If} ${RunningX64}
        StrCpy $INSTDIR "$LOCALAPPDATA\Programs\DDEV"
    ${Else}
        MessageBox MB_ICONSTOP|MB_OK "This installer is for 64-bit Windows only."
        Abort
    ${EndIf}
    
    ; Initialize debug logging
    Call InitializeDebugLog
    Push "Debug log initialized at: $DEBUG_LOG_PATH"
    Call LogPrint

    ; Check for old system-wide installation and offer to remove it
    Call CheckOldSystemInstallation

    ; Parse command line arguments
    Call ParseCommandLine
    
    ; Handle installation type selection
    ${If} $SILENT_INSTALL_TYPE != ""
        ; Command line argument specified - use it
        StrCpy $INSTALL_OPTION $SILENT_INSTALL_TYPE
        ${If} $SILENT_DISTRO != ""
            StrCpy $SELECTED_DISTRO $SILENT_DISTRO
        ${EndIf}
        Push "Command line install with type: $INSTALL_OPTION"
        Call LogPrint
    ${ElseIf} ${Silent}
        ; Legacy silent install (Chocolatey) - default to traditional
        StrCpy $INSTALL_OPTION "traditional"
        Push "Silent install detected, defaulting to traditional Windows installation"
        Call LogPrint
    ${EndIf}
FunctionEnd

; Helper: Show error message with standard guidance and abort
; Call with error message on stack
Function ShowErrorAndAbort
    Exch $R0  ; Get error message from stack
    Push "INSTALLATION ERROR: $R0"
    Call LogPrint

    ; Write error status to WSL2 distro if available
    ${If} $SELECTED_DISTRO != ""
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "echo \"ERROR: $R0\" >> /tmp/ddev_installation_status.txt"'
        Pop $1
        Pop $2
    ${EndIf}

    ${IfNot} ${Silent}
        ; Lead with the log path so it stays visible even if the error
        ; body is long enough to scroll off-screen in the MessageBox.
        ; Truncate the error body to keep the dialog compact; the full
        ; error is always available in the debug log.
        StrLen $R1 $R0
        ${If} $R1 > 600
            StrCpy $R2 $R0 600
            StrCpy $R2 "$R2...$\n[truncated — see debug log for full output]"
        ${Else}
            StrCpy $R2 $R0
        ${EndIf}
        MessageBox MB_ICONSTOP|MB_OKCANCEL "DDEV installation failed.$\n$\nFull debug log (please include with any error report):$\n$DEBUG_LOG_PATH$\n$\n----- Error -----$\n$R2$\n$\nClick OK to open the debug log in Notepad, or Cancel to exit." IDOK open_log IDCANCEL skip_log
        open_log:
            ExecShell "open" "notepad.exe" "$DEBUG_LOG_PATH"
        skip_log:
    ${EndIf}
    Push "Exiting installer due to error. Debug log: $DEBUG_LOG_PATH (please include with any error report)"
    Call LogPrint
    SetErrorLevel 1
    Quit
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
    ; Initialize to not approved
    StrCpy $MKCERT_UNINSTALL_APPROVED "0"

    ${If} ${FileExists} "$INSTDIR\mkcert.exe"
        Push $0

        ; Read setup status from registry
        ReadRegDWORD $0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup"

        ; Check if setup was done
        ${If} $0 == 1
            ${If} ${Silent}
                ; Silent mode - skip mkcert -uninstall because it pops up a dialog window
                ; Just clean up the CAROOT directory manually if possible
            ${Else}
                ; Interactive mode - get user confirmation
                MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "mkcert was found in this installation. Do you like to remove the mkcert configuration?" /SD IDNO IDYES +2
                Goto Skip

                ; User said YES - set flag and proceed
                StrCpy $MKCERT_UNINSTALL_APPROVED "1"
                MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to disable trusted https. Please accept the mkcert dialog box that may follow."

                nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -uninstall'
                Pop $0 ; get return value
                Pop $1 ; get output
            ${EndIf}

        Skip:
        ${EndIf}

        Pop $0
    ${EndIf}
FunctionEnd

Function un.CleanupMkcertEnvironment
    ; Skip mkcert cleanup if user said no to mkcert removal (and not silent mode)
    ${IfNot} ${Silent}
        ${If} $MKCERT_UNINSTALL_APPROVED != "1"
            DetailPrint "Skipping mkcert environment cleanup (user declined mkcert removal)"
            Return
        ${EndIf}
    ${EndIf}

    DetailPrint "Cleaning up mkcert environment variables..."

    ; Get CAROOT directory before cleanup
    ReadRegStr $R0 HKCU "Environment" "CAROOT"
    ${IfNot} ${Errors}
        DetailPrint "CAROOT directory: $R0"

        ; mkcert -uninstall was already run in un.mkcertUninstall if user approved
        ; No need to run it again here

        ; Remove any remaining CAROOT directory
        ${If} ${FileExists} "$R0"
            DetailPrint "Removing remaining CAROOT directory: $R0"
            RMDir /r "$R0"
        ${EndIf}
    ${EndIf}

    ; Remove CAROOT environment variable (skip in silent mode to preserve for subsequent installs)
    ${IfNot} ${Silent}
        DeleteRegValue HKCU "Environment" "CAROOT"
        DetailPrint "Removed CAROOT environment variable"
    ${Else}
        DetailPrint "Preserving CAROOT environment variable in silent mode"
    ${EndIf}

    ; Clean up WSLENV by removing CAROOT/up (skip in silent mode to preserve for subsequent installs)
    ${IfNot} ${Silent}
        ReadRegStr $R0 HKCU "Environment" "WSLENV"
        ${If} ${Errors}
            DetailPrint "WSLENV not found, nothing to clean up"
            Return
        ${EndIf}

        DetailPrint "Current WSLENV: $R0"

        ; Remove all instances of CAROOT/up with colon separator (correct WSLENV format)
        ${UnStrRep} $R0 $R0 "CAROOT/up:" ""
        ${UnStrRep} $R0 $R0 ":CAROOT/up" ""

        ; Remove all instances of CAROOT/up with semicolon separator (legacy/incorrect format)
        ${UnStrRep} $R0 $R0 "CAROOT/up;" ""
        ${UnStrRep} $R0 $R0 ";CAROOT/up" ""

        ; Remove if it's exactly "CAROOT/up" by itself
        ${If} $R0 == "CAROOT/up"
            StrCpy $R0 ""
        ${EndIf}

        ; Clean up any double separators that might have been created
        ${UnStrRep} $R0 $R0 "::" ":"
        ${UnStrRep} $R0 $R0 ";;" ";"

        ; Remove leading or trailing separators (: or ;)
        ${If} $R0 != ""
            StrCpy $R1 $R0 1  ; Get first character
            ${If} $R1 == ":"
                StrCpy $R0 $R0 "" 1  ; Remove first character
            ${EndIf}
            ${If} $R1 == ";"
                StrCpy $R0 $R0 "" 1  ; Remove first character
            ${EndIf}
            ${If} $R0 != ""
                StrLen $R2 $R0
                IntOp $R2 $R2 - 1
                StrCpy $R1 $R0 1 $R2  ; Get last character
                ${If} $R1 == ":"
                    StrCpy $R0 $R0 $R2  ; Remove last character
                ${EndIf}
                ${If} $R1 == ";"
                    StrCpy $R0 $R0 $R2  ; Remove last character
                ${EndIf}
            ${EndIf}
        ${EndIf}

        ; Update or delete WSLENV
        ${If} $R0 == ""
            DeleteRegValue HKCU "Environment" "WSLENV"
            DetailPrint "Removed empty WSLENV"
        ${Else}
            WriteRegStr HKCU "Environment" "WSLENV" "$R0"
            DetailPrint "Updated WSLENV to: $R0"
        ${EndIf}
    ${Else}
        DetailPrint "Preserving WSLENV environment variable in silent mode"
    ${EndIf}

    DetailPrint "mkcert environment variables cleanup completed"
FunctionEnd


; LaunchSponsors - Open GitHub sponsors page
Function LaunchSponsors
    ExecShell "open" "https://github.com/sponsors/ddev"
FunctionEnd


; Installation completion callbacks for proper exit code handling
Function .onInstSuccess
    Push "Installation completed successfully"
    Call LogPrint
    SetErrorLevel 0
FunctionEnd

Function .onInstFailed
    Push "Installation failed"
    Call LogPrint
    SetErrorLevel 1
FunctionEnd
