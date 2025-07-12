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

!insertmacro WordFind
${StrStr}
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
Var /GLOBAL WINDOWS_CAROOT
Var /GLOBAL DEBUG_LOG_HANDLE
Var /GLOBAL DEBUG_LOG_PATH
Var StartMenuGroup

!define REG_INSTDIR_ROOT "HKLM"
!define REG_INSTDIR_KEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe"
!define REG_UNINST_ROOT "HKLM"
!define REG_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define REG_SETTINGS_ROOT "HKLM"
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
    StrCpy $DEBUG_LOG_PATH "$TEMP\ddev_installer_debug.log"
    FileOpen $DEBUG_LOG_HANDLE "$DEBUG_LOG_PATH" w
    ${If} $DEBUG_LOG_HANDLE != ""
        FileWrite $DEBUG_LOG_HANDLE "=== DDEV Installer Debug Log ===$\r$\n"
        FileWrite $DEBUG_LOG_HANDLE "Log location: $DEBUG_LOG_PATH$\r$\n"
        FileWrite $DEBUG_LOG_HANDLE "Installer started at: $\r$\n"
    ${EndIf}
FunctionEnd

; LogPrint - DetailPrint wrapper that also writes to debug log
; Usage: Push "message" ; Call LogPrint
Function LogPrint
    Exch $R0  ; Get message from stack
    Push $R1
    
    ; Always do DetailPrint
    DetailPrint "$R0"
    
    ; Write to log file if handle is open
    ${If} $DEBUG_LOG_HANDLE != ""
        FileWrite $DEBUG_LOG_HANDLE "$R0$\r$\n"
    ${EndIf}
    
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
    
    Push "Installing script $R0 to WSL2 distro $R1..."
    Call LogPrint
    
    ; Scripts should already be copied to temp directory by this point
    ${If} ${FileExists} "C:\Windows\Temp\ddev_installer\$R0"
        Push "Using script $R0 from temp directory"
        Call LogPrint
    ${Else}
        Push "ERROR: Script $R0 not found in temp directory"
        Call LogPrint
        Push 1
        Return
    ${EndIf}
    
    ; Copy script from Windows temp to WSL2 /tmp
    nsExec::ExecToStack 'wsl -d $R1 -u root cp "/mnt/c/Windows/Temp/ddev_installer/$R0" /tmp/'
    Pop $R2  ; Exit code
    Pop $R3  ; Output
    
    ${If} $R2 != 0
        Push "Failed to copy script $R0 to distro $R1: $R3"
        Call LogPrint
        Push $R2
        Return
    ${EndIf}
    
    ; Make script executable
    nsExec::ExecToStack 'wsl -d $R1 -u root chmod +x /tmp/$R0'
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

    ; Get Ubuntu distros before creating any controls
    Call GetUbuntuDistros
    Pop $R0
    Push "Got distros: [$R0]"
    Call LogPrint
    ${If} $R0 == ""
        Push "ERROR: No Ubuntu-based WSL2 distributions found"
        Call LogPrint
        MessageBox MB_ICONSTOP|MB_OK "No Ubuntu-based WSL2 distributions found. Please install Ubuntu for WSL2 first.$\n$\nDebug information has been written to: $DEBUG_LOG_PATH$\n$\nYou can check this file to see what distributions were detected."
        Push "No Ubuntu-based WSL2 distributions found. Please install Ubuntu for WSL2 first."
        Call ShowErrorAndAbort
    ${EndIf}

    Push "Creating label..."
    Call LogPrint
    ${NSD_CreateLabel} 0 0 100% 24u "Select your Ubuntu-based WSL2 distribution:"
    Pop $1

    Push "Creating radio buttons..."
    Call LogPrint

    ; Get previously selected distro
    ReadRegStr $R8 ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "SelectedDistro"
    Push "Previously selected distro: $R8"
    Call LogPrint

    ; Initialize variables for dynamic radio button creation
    Var /GLOBAL RADIO_BUTTON_COUNT
    Var /GLOBAL RADIO_BUTTON_HANDLES    ; Will store pipe-separated list of handles
    Var /GLOBAL RADIO_BUTTON_LABELS     ; Will store pipe-separated list of labels
    Var /GLOBAL SELECTED_RADIO_INDEX    ; Index of selected radio button
    
    StrCpy $RADIO_BUTTON_COUNT 0
    StrCpy $RADIO_BUTTON_HANDLES ""
    StrCpy $RADIO_BUTTON_LABELS ""
    StrCpy $SELECTED_RADIO_INDEX 0

    ; Process the pipe-separated list and create radio buttons
    StrCpy $R1 $R0    ; Working copy of the list
    StrCpy $R2 0      ; Current item index
    StrCpy $R3 0      ; Y position counter

    ${Do}
        ; Find position of next pipe or end
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
            Push "Adding radio button: [$R7]"
            Call LogPrint
            
            ; Calculate Y position for radio button
            IntOp $R9 $R3 * 24
            IntOp $R9 $R9 + 30
            
            ; Create radio button
            ${NSD_CreateRadioButton} 10 $R9u 280u 16u "$R7"
            Pop $9
            
            ; Store handle and label in our lists
            ${If} $RADIO_BUTTON_HANDLES == ""
                StrCpy $RADIO_BUTTON_HANDLES "$9"
                StrCpy $RADIO_BUTTON_LABELS "$R7"
            ${Else}
                StrCpy $RADIO_BUTTON_HANDLES "$RADIO_BUTTON_HANDLES|$9"
                StrCpy $RADIO_BUTTON_LABELS "$RADIO_BUTTON_LABELS|$R7"
            ${EndIf}
            
            ; Check if this matches the previously selected distro
            ${If} $R7 == $R8
                StrCpy $SELECTED_RADIO_INDEX $R2
                ${NSD_SetState} $9 ${BST_CHECKED}
                Push "Selected distro: $R7 (previous choice)"
                Call LogPrint
            ${ElseIf} $R2 == 0
            ${AndIf} $R8 == ""
                ${NSD_SetState} $9 ${BST_CHECKED}
                Push "Selected distro: $R7 (default)"
                Call LogPrint
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

    StrCpy $RADIO_BUTTON_COUNT $R2
    Push "Added $RADIO_BUTTON_COUNT radio buttons"
    Call LogPrint

    Push "About to show dialog..."
    Call LogPrint
    nsDialogs::Show
FunctionEnd

Function DistroSelectionPageLeave
    Push "Getting selected distro..."
    Call LogPrint
    
    ; Find which radio button is selected by iterating through all handles
    StrCpy $R1 $RADIO_BUTTON_HANDLES  ; Working copy of handles
    StrCpy $R2 $RADIO_BUTTON_LABELS   ; Working copy of labels
    StrCpy $R3 0                      ; Current index
    StrCpy $SELECTED_DISTRO ""        ; Clear selection
    
    ${Do}
        ; Extract current handle
        ${WordFind} "$R1" "|" "+1{" $R4  ; Get first handle
        ${If} $R4 == $R1
            ; Last item (no more separators)
            StrCpy $R5 $R1
            StrCpy $R1 ""
        ${Else}
            ; More items remain
            StrCpy $R5 $R4
            ${WordFind} "$R1" "|" "+1}" $R1  ; Remove first item
        ${EndIf}
        
        ; Extract corresponding label
        ${WordFind} "$R2" "|" "+1{" $R6  ; Get first label
        ${If} $R6 == $R2
            ; Last item (no more separators)
            StrCpy $R7 $R2
            StrCpy $R2 ""
        ${Else}
            ; More items remain
            StrCpy $R7 $R6
            ${WordFind} "$R2" "|" "+1}" $R2  ; Remove first item
        ${EndIf}
        
        ; Check if this radio button is selected
        ${NSD_GetState} $R5 $R0
        ${If} $R0 == ${BST_CHECKED}
            StrCpy $SELECTED_DISTRO $R7
            Push "Selected distro: $SELECTED_DISTRO"
            Call LogPrint
            ${Break}
        ${EndIf}
        
        IntOp $R3 $R3 + 1
        
        ; Check if we're done
        ${If} $R1 == ""
            ${Break}
        ${EndIf}
    ${Loop}
    
    ; Fallback - should not happen if we have proper radio button logic
    ${If} $SELECTED_DISTRO == ""
        Push "No distro selected - using first available"
        Call LogPrint
        ${WordFind} "$RADIO_BUTTON_LABELS" "|" "+1{" $SELECTED_DISTRO
    ${EndIf}
    
    ; Store the selected distro for next time
    WriteRegStr ${REG_SETTINGS_ROOT} "${REG_SETTINGS_KEY}" "SelectedDistro" $SELECTED_DISTRO
    Push "Stored selected distro: $SELECTED_DISTRO"
    Call LogPrint

    ; Copy all scripts to temp directory for later use
    Push "Copying all scripts to temp directory..."
    Call LogPrint
    CreateDirectory "C:\Windows\Temp\ddev_installer"
    SetOutPath "C:\Windows\Temp\ddev_installer"
    File /oname=check_root_user.sh "scripts\check_root_user.sh"
    File /oname=mkcert_install.sh "scripts\mkcert_install.sh"
    File /oname=install_temp_sudoers.sh "scripts\install_temp_sudoers.sh"
    File /oname=ddev-wsl2-postinstall.sh "scripts\ddev-wsl2-postinstall.sh"
    Push "All scripts copied to temp directory"
    Call LogPrint
    
    ; Check for root user immediately after distro selection (WSL2 only)
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
    ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
        Push "Checking for root user in selected distro..."
        Call LogPrint
        Call CheckRootUser
        Push "Root user check passed"
        Call LogPrint
    ${EndIf}
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
        MessageBox MB_ICONSTOP|MB_OK "Failed to install check_root_user to WSL2 distro"
        Abort
    ${EndIf}

    Push "Running check_root_user.sh..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash /tmp/check_root_user.sh'
    Pop $R4
    Pop $R5
    ${If} $R4 != 0
        Push "Root user detected in distro $SELECTED_DISTRO"
        Call LogPrint
        Push "Exiting installer due to root user detection"
        Call LogPrint
        Push "The default user in distro $SELECTED_DISTRO is still 'root', but it needs to be a normal user.$\n$\nPlease configure your WSL2 distro to use a normal user account instead of root.$\n$\nRefer to WSL documentation for instructions on changing the default user."
        Call ShowErrorAndAbort
    ${Else}
        Push "Root user check passed for distro $SELECTED_DISTRO"
        Call LogPrint
    ${EndIf}

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

; Git for Windows check page for traditional installation
Page custom GitCheckPage GitCheckPageLeave

; Docker provider check page for all installations
Page custom DockerCheckPage DockerCheckPageLeave

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

; Standard MUI finish page with custom run action
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Support DDEV - Open GitHub Sponsors page"
!define MUI_FINISHPAGE_RUN_FUNCTION "LaunchSponsors"
!define MUI_FINISHPAGE_TITLE "DDEV Installation Complete"
!define MUI_FINISHPAGE_TEXT "Thank you for installing DDEV!$\r$\n$\r$\nPlease consider supporting DDEV so we can continue supporting you."
!define MUI_FINISHPAGE_RUN_CHECKED  ; Pre-check the box to encourage action
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
FunctionEnd

Function DockerCancelButtonClick
    Push "User clicked Cancel Installation button"
    Call LogPrint
    MessageBox MB_ICONINFORMATION|MB_OK "Installation cancelled.$\n$\nA Docker provider is required for DDEV installation.$\n$\nThe installer will now exit."
    Push "Exiting installer - user cancelled Docker installation"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
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
            File /oname=mkcert_install.sh "scripts\mkcert_install.sh"
            File /oname=install_temp_sudoers.sh "scripts\install_temp_sudoers.sh"
            File /oname=check_root_user.sh "scripts\check_root_user.sh"
        ${EndIf}

        ; Install icons
        SetOutPath "$INSTDIR\Icons"
        SetOverwrite try
        File /oname=ddev.ico "graphics\ddev-install.ico"

        ; Run mkcert.exe -install early for all installation types (needed for WSL2 setup)
        Call RunMkcertInstall

        ; Add DDEV installation directory to PATH (EnVar::AddValue handles duplicates)
        Push "Adding DDEV installation directory to system PATH..."
        Call LogPrint
        ReadRegStr $R0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "Path"
        Push "PATH before addition: $R0"
        Call LogPrint
        
        EnVar::SetHKLM
        EnVar::AddValue "Path" "$INSTDIR"
        Pop $R1
        Push "EnVar::AddValue result: $R1"
        Call LogPrint
        
        Push "PATH addition completed with result: $R1"
        Call LogPrint

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
    
    ; Calculate and write estimated size for Add/Remove Programs
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "EstimatedSize" "$0"

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
    StrCpy $R0 ""  ; Result string

    Push "=== Starting GetUbuntuDistros ==="
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
                Push "Found Ubuntu distribution (name-based): $R3"
                Call LogPrint
                ${If} $R0 != ""
                    StrCpy $R0 "$R0|"
                ${EndIf}
                StrCpy $R0 "$R0$R3"
            ${Else}
                Push "Distribution '$R3' does not start with 'Ubuntu' (starts with '$R4')"
                Call LogPrint
            ${EndIf}
        ${Else}
            Push "Found Flavor field for $R3: '$R4'"
            Call LogPrint
            ; Check if Flavor is "ubuntu" (case-insensitive)
            ${StrStr} $R6 $R4 "ubuntu"
            ${If} $R6 != ""
                Push "Found Ubuntu distribution (Flavor-based): $R3"
                Call LogPrint
                ${If} $R0 != ""
                    StrCpy $R0 "$R0|"
                ${EndIf}
                StrCpy $R0 "$R0$R3"
            ${Else}
                Push "Distribution '$R3' has Flavor '$R4' but does not contain 'ubuntu'"
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

; TODO: there seem to be missing error checks here.
Function InstallWSL2CommonSetup
    ; Check for WSL2
    Push "Checking WSL2 version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl.exe -l -v'
    Pop $1
    Pop $0
    Push "WSL version check output: $0"
    Call LogPrint
    Push "WSL version check exit code: $1"
    Call LogPrint
    ${If} $1 != 0
        Push "ERROR: WSL2 not detected - exit code: $1, output: $0"
        Call LogPrint
        Push "WSL2 does not seem to be installed. Please install WSL2 and Ubuntu before running this installer."
        Call ShowErrorAndAbort
    ${EndIf}

    ; Check for Ubuntu in selected distro
    Push "Checking selected distro $SELECTED_DISTRO..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO bash -c "cat /etc/os-release | grep -i ^NAME="'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Cannot access distro $SELECTED_DISTRO - exit code: $1, output: $0"
        Call LogPrint
        Push "Could not access the selected distro. Please ensure it's working properly."
        Call ShowErrorAndAbort
    ${EndIf}

    ; Check for WSL2 kernel
    Push "Checking WSL2 kernel..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO uname -v'
    Pop $1
    Pop $0
    Push "WSL kernel version: $0"
    Call LogPrint
    ${If} $1 != 0
        Push "ERROR: WSL version check failed - exit code: $1, output: $0"
        Call LogPrint
        Push "Could not check WSL version. Please ensure WSL is working."
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $0 == ""
        Push "ERROR: Empty WSL version output"
        Call LogPrint
        Push "Could not detect WSL version. Please ensure WSL is working."
        Call ShowErrorAndAbort
    ${EndIf}
    ${If} $0 == "WSL"
        Push "ERROR: WSL1 detected instead of WSL2 - version output: $0"
        Call LogPrint
        Push "The selected distro ($SELECTED_DISTRO) is not WSL2. Please use a WSL2 distro."
        Call ShowErrorAndAbort
    ${EndIf}
    Push "WSL2 detected successfully."
    Call LogPrint


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

    ; Remove old Docker versions first
    Push "WSL($SELECTED_DISTRO): Removing old Docker packages if present..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "apt-get remove -y -qq docker docker-engine docker.io containerd runc >/dev/null 2>&1"'
    Pop $1
    Pop $0
    ; Note: This command is allowed to fail if packages aren't installed

    ; apt-get update
    Push "WSL($SELECTED_DISTRO): Updating package database using apt-get update..."
    Call LogPrint
    Push "Please be patient - updating package database..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "apt-get update >/dev/null 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: apt-get update failed - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to apt-get update. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Install linux packages
    Push "WSL($SELECTED_DISTRO): Installing required linux packages..."
    Call LogPrint
    Push "Please be patient - installing required linux packages..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root apt-get install -y ca-certificates curl gnupg gnupg2 libsecret-1-0 lsb-release pass'
    Pop $1
    Pop $0
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

    ; Add Docker GPG key
    Push "WSL($SELECTED_DISTRO): Adding Docker repository key..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "rm -f /etc/apt/keyrings/docker.gpg && mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add Docker repository key - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add Docker apt repository key. Please check your internet connection. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add Docker repository
    Push "WSL($SELECTED_DISTRO): Adding Docker apt repository..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root -e bash -c "echo deb [arch=$$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $$(lsb_release -cs) stable | tee /etc/apt/sources.list.d/docker.list > /dev/null 2>&1"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add Docker repository - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add Docker repository. Exit code: $1, Output: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add DDEV GPG key
    Push "WSL($SELECTED_DISTRO): Adding DDEV apt repository key..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "curl -fsSL https://pkg.ddev.com/apt/gpg.key | gpg --dearmor | tee /etc/apt/keyrings/ddev.gpg > /dev/null"'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Failed to add DDEV repository key - exit code: $1, output: $0"
        Call LogPrint
        Push "Failed to add DDEV repository key. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Add DDEV repository
    Push "WSL($SELECTED_DISTRO): Adding DDEV apt repository..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root -e bash -c "echo \"deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://pkg.ddev.com/apt/ * *\" > /etc/apt/sources.list.d/ddev.list"'
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
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get update 2>&1"'
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
    Call InstallWSL2CommonSetup

    ${If} $INSTALL_OPTION == "wsl2-docker-desktop"
        ; Install packages needed for Docker Desktop (including ddev)
        StrCpy $0 "docker-ce-cli wslu ddev"
    ${Else}
        ; Install full Docker CE packages (including ddev)
        StrCpy $0 "docker-ce docker-ce-cli containerd.io wslu ddev"
    ${EndIf}

    ; Install packages in multiple steps for better progress feedback
    Push "WSL($SELECTED_DISTRO): Installing essential packages (1/4)..."
    Call LogPrint
    Push "Please be patient - installing essential packages..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gnupg gnupg2 libsecret-1-0 lsb-release pass 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install essential packages - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install essential packages. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    Push "WSL($SELECTED_DISTRO): Installing Docker components (2/4)..."
    Call LogPrint
    Push "Please be patient - installing Docker components..."
    Call LogPrint
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io 2>&1"'
    ${Else}
        nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce-cli 2>&1"'
    ${EndIf}
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install Docker components - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install Docker components. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    Push "WSL($SELECTED_DISTRO): Installing WSL utilities (3/4)..."
    Call LogPrint
    Push "Please be patient - installing WSL utilities..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y wslu 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install WSL utilities - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install WSL utilities. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    Push "WSL($SELECTED_DISTRO): Installing DDEV (4/4)..."
    Call LogPrint
    Push "Please be patient - installing DDEV..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root bash -c "DEBIAN_FRONTEND=noninteractive apt-get install -y ddev ddev-wsl2 2>&1"'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: Failed to install DDEV - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to install DDEV. Error: $2"
        Call ShowErrorAndAbort
    ${EndIf}

    ; Overwrite the installed DDEV binary with the bundled version
    Push "WSL($SELECTED_DISTRO): Overwriting DDEV binary with bundled version..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "/mnt/c/Windows/Temp/ddev_installer/ddev_linux" /usr/bin/ddev'
    Pop $1
    Pop $2
    ${If} $1 != 0
        Push "ERROR: DDEV binary overwrite failed - exit code: $1, output: $2"
        Call LogPrint
        Push "Failed to overwrite DDEV binary. Error: $2"
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
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO -u root cp "/mnt/c/Windows/Temp/ddev_installer/ddev-hostname_linux" /usr/bin/ddev-hostname'
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

    ; Final validation - ensure DDEV is actually working
    Push "Performing final validation of DDEV installation..."
    Call LogPrint
    nsExec::ExecToStack 'wsl -d $SELECTED_DISTRO ddev version'
    Pop $1
    Pop $0
    ${If} $1 != 0
        Push "ERROR: Final DDEV validation failed - exit code: $1, output: $0"
        Call LogPrint
        Push "Installation validation failed. DDEV may not be working properly. Error: $0"
        Call ShowErrorAndAbort
    ${EndIf}
    
    ; Configure WSL2 security settings to prevent Windows security warnings
    Push "Configuring WSL2 security settings to prevent Windows executable warnings..."
    Call LogPrint
    
    ; Configure WSL2 security settings directly via registry
    Push "Configuring WSL2 security settings..."
    Call LogPrint
    
    ; Try to add wsl.localhost to Local Intranet zone via registry
    Push "Adding *.wsl.localhost to Local Intranet security zone..."
    Call LogPrint
    nsExec::ExecToStack 'reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings\ZoneMap\Domains\wsl.localhost" /v "file" /t REG_DWORD /d 1 /f'
    Pop $R0
    Pop $R1
    ${If} $R0 == 0
        Push "WSL2 security settings configured successfully via registry"
        Call LogPrint
    ${Else}
        ; Fallback: Try PowerShell approach
        Push "Registry method failed, trying PowerShell approach..."
        Call LogPrint
        nsExec::ExecToStack 'powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "try { $$path = \"HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\\ZoneMap\\Domains\\wsl.localhost\"; if (-not (Test-Path $$path)) { New-Item -Path $$path -Force | Out-Null }; Set-ItemProperty -Path $$path -Name \"file\" -Value 1 -Type DWord; Write-Host \"Success\" } catch { exit 1 }"'
        Pop $R2
        Pop $R3
        ${If} $R2 == 0
            Push "WSL2 security settings configured via PowerShell"
            Call LogPrint
        ${Else}
            ; Final fallback: Show manual instructions
            Push "Could not automatically configure WSL2 security settings."
            Call LogPrint
            Push "To resolve Windows security warnings manually:"
            Call LogPrint
            Push "1. Open Internet Options (Control Panel > Internet Options)"
            Call LogPrint
            Push "2. Go to Security tab > Local Intranet > Sites > Advanced"
            Call LogPrint
            Push "3. Add this website to the zone: \\\\wsl.localhost"
            Call LogPrint
            Push "4. Click OK to save"
            Call LogPrint
        ${EndIf}
    ${EndIf}
    
    ; Clean up temp directory
    Push "Cleaning up temporary files..."
    Call LogPrint
    Delete "C:\Windows\Temp\ddev_installer\ddev_linux"
    Delete "C:\Windows\Temp\ddev_installer\ddev-hostname_linux"
    Delete "C:\Windows\Temp\ddev_installer\ddev-wsl2-postinstall.sh"
    RMDir "C:\Windows\Temp\ddev_installer"
    
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

    ; Remove CAROOT environment variable for traditional Windows (WSL2-specific)
    Push "Removing CAROOT environment variable (not needed for traditional Windows)"
    Call LogPrint
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

    Push "Traditional Windows installation completed."
    Call LogPrint

FunctionEnd

Function RunMkcertInstall
    ${If} ${Silent}
        ; In silent mode, skip mkcert.exe -install to avoid UAC prompts
        ; But still set up CAROOT environment variable for WSL2 installs
        ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
            Push "Setting up CAROOT environment variable for WSL2 in silent mode..."
            Call LogPrint
            Call SetupWindowsCAROOT
        ${Else}
            Push "Skipping mkcert setup in silent mode for traditional Windows install"
            Call LogPrint
        ${EndIf}
        Return
    ${EndIf}
    
    Push "Setting up mkcert.exe (Windows) for trusted HTTPS certificates..."
    Call LogPrint
    
    ; Unset CAROOT environment variable in current process
    System::Call 'kernel32::SetEnvironmentVariable(t "CAROOT", i 0)'
    Pop $0

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
        
        ; Set up CAROOT environment variable for WSL2 sharing (only for WSL2 installs)
        ${If} $INSTALL_OPTION == "wsl2-docker-ce"
        ${OrIf} $INSTALL_OPTION == "wsl2-docker-desktop"
            Call SetupWindowsCAROOT
        ${EndIf}
    ${Else}
        Push "mkcert.exe -install failed with exit code: $R0"
        Call LogPrint
        MessageBox MB_ICONEXCLAMATION|MB_OK "mkcert -install failed with exit code: $R0. Output: $R1. You may need to run 'mkcert.exe -install' manually on Windows."
    ${EndIf}
FunctionEnd

Function SetupWindowsCAROOT
    Push "Setting up mkcert certificate sharing with WSL2..."
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
        EnVar::SetHKLM
        EnVar::Delete "CAROOT"  ; Remove entire variable first
        Pop $0  ; Get error code from Delete
        Push "EnVar::Delete CAROOT result: $0"
        Call LogPrint
        
        EnVar::AddValue "CAROOT" "$R1"
        Pop $0  ; Get error code from AddValue
        Push "EnVar::AddValue CAROOT result: $0"
        Call LogPrint
        
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
        
        Push "mkcert certificate sharing with WSL2 configured successfully."
        Call LogPrint
        
        ; Read current value from registry for verification
        ; ReadRegStr $R6 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
        ; DetailPrint "WSLENV verification - Original: [$R4], Set to: [$R2], Actual: [$R6]"
    ${Else}
        Push "Failed to get CAROOT directory from mkcert"
        Call LogPrint
        MessageBox MB_ICONEXCLAMATION|MB_OK "Failed to get CAROOT directory from mkcert. WSL2 certificate sharing may not work properly."
    ${EndIf}
FunctionEnd

Function SetupMkcertInWSL2
    Push "Setting up mkcert inside WSL2 distro: $SELECTED_DISTRO"
    Call LogPrint
    
    ; Check current Windows CAROOT environment variable from registry
    ReadRegStr $R2 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
    StrCpy $WINDOWS_CAROOT $R2  ; Save to global variable for later use
    Push "Windows CAROOT environment variable: $WINDOWS_CAROOT"
    Call LogPrint
    
    ; Check current Windows WSLENV environment variable from registry
    ReadRegStr $R3 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "WSLENV"
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
    
    ; Initialize debug logging
    Call InitializeDebugLog
    Push "Debug log initialized at: $DEBUG_LOG_PATH"
    Call LogPrint
    
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
; Helper: Show error message with standard guidance and abort
; Call with error message on stack
Function ShowErrorAndAbort
    Exch $R0  ; Get error message from stack
    Push "INSTALLATION ERROR: $R0"
    Call LogPrint
    ${IfNot} ${Silent}
        MessageBox MB_ICONSTOP|MB_OK "$R0$\n$\nDebug information has been written to: $DEBUG_LOG_PATH$\n$\nPlease fix the issue and retry the installer."
    ${EndIf}
    Push "Exiting installer due to error"
    Call LogPrint
    SendMessage $HWNDPARENT ${WM_CLOSE} 0 0
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
            ${If} ${Silent}
                ; Silent mode - skip mkcert -uninstall to avoid UAC/certificate store popups
                ; Just clean up the CAROOT directory manually if possible
            ${Else}
                ; Interactive mode - get user confirmation
                MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "mkcert was found in this installation. Do you like to remove the mkcert configuration?" /SD IDNO IDYES +2
                Goto Skip
                
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
    DetailPrint "Cleaning up mkcert environment variables..."
    
    ; Get CAROOT directory before cleanup
    ReadRegStr $R0 HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
    ${IfNot} ${Errors}
        DetailPrint "CAROOT directory: $R0"
        
        ; Run mkcert -uninstall first to properly clean up certificates (skip in silent mode)
        ${If} ${FileExists} "$INSTDIR\mkcert.exe"
            ${IfNot} ${Silent}
                DetailPrint "Running mkcert -uninstall to clean up certificates..."
                nsExec::ExecToStack '"$INSTDIR\mkcert.exe" -uninstall'
                Pop $R1
                Pop $R2 ; get output
                ${If} $R1 = 0
                    DetailPrint "mkcert -uninstall completed successfully"
                ${Else}
                    DetailPrint "mkcert -uninstall failed with exit code: $R1"
                ${EndIf}
            ${Else}
                DetailPrint "Skipping mkcert -uninstall in silent mode to avoid UAC prompts"
            ${EndIf}
        ${EndIf}
        
        ; Remove any remaining CAROOT directory
        ${If} ${FileExists} "$R0"
            DetailPrint "Removing remaining CAROOT directory: $R0"
            RMDir /r "$R0"
        ${EndIf}
    ${EndIf}
    
    ; Remove CAROOT environment variable (skip in silent mode to preserve for subsequent installs)
    ${IfNot} ${Silent}
        DeleteRegValue HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment" "CAROOT"
        DetailPrint "Removed CAROOT environment variable"
    ${Else}
        DetailPrint "Preserving CAROOT environment variable in silent mode"
    ${EndIf}
    
    ; Clean up WSLENV by removing CAROOT/up (skip in silent mode to preserve for subsequent installs)
    ${IfNot} ${Silent}
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
    ${Else}
        DetailPrint "Preserving WSLENV environment variable in silent mode"
    ${EndIf}
    
    DetailPrint "mkcert environment variables cleanup completed"
FunctionEnd


; LaunchSponsors - Open GitHub sponsors page
Function LaunchSponsors
    Push "User clicked Support DDEV button - opening GitHub sponsors page"
    Call LogPrint
    ExecShell "open" "https://github.com/sponsors/ddev"
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
