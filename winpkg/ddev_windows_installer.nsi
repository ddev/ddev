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

    ; Check if the default WSL distro is Ubuntu
    nsExec::ExecToStack 'wsl bash -c "grep ^NAME..Ubuntu /etc/os-release"'
    Pop $0
    Pop $1
    DetailPrint "DEBUG: Output of grep /etc/os-release for Ubuntu: $1"
    ${If} $1 == ""
        MessageBox MB_ICONSTOP|MB_OK "Your default WSL distro is not Ubuntu. Please set Ubuntu as your default WSL distro."
        Abort "Default WSL distro is not Ubuntu"
    ${EndIf}

    ; Install Docker CE in the default WSL Ubuntu distro
    DetailPrint "DEBUG: Installing Docker CE in default WSL Ubuntu distro..."
    MessageBox MB_ICONINFORMATION|MB_OK "DEBUG: About to run: wsl -- bash -c curl -fsSL https://get.docker.com | sh"
    nsExec::ExecToLog 'wsl -- bash -c curl -fsSL https://get.docker.com | sh'
    Pop $0
    MessageBox MB_OK "DEBUG: Result of Docker CE install: $0"
    ${If} $0 != 0
        MessageBox MB_ICONSTOP|MB_OK "Failed to install Docker CE in WSL2. Please check the logs."
        Abort "Docker CE installation failed"
    ${EndIf}

    ; Configure Docker to start automatically
    DetailPrint "DEBUG: Enabling Docker to start automatically"
    MessageBox MB_OK "DEBUG: About to run: wsl -- bash -c sudo systemctl enable docker"
    nsExec::ExecToLog 'wsl -- bash -c sudo systemctl enable docker'
    Pop $8
    MessageBox MB_OK "DEBUG: Result of systemctl enable docker: $8"

    ; Install required Windows components
    SetOutPath $INSTDIR
    SetOverwrite on

    ; Only install ddev-hostname.exe, not ddev.exe
    DetailPrint "DEBUG: Installing ddev-hostname.exe"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"

    ; Install mkcert
    DetailPrint "DEBUG: Installing mkcert"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"

    ; Install icons for mkcert
    SetOutPath "$INSTDIR\Icons"
    SetOverwrite try
    File /oname=ca-install.ico "graphics\ca-install.ico"
    File /oname=ca-uninstall.ico "graphics\ca-uninstall.ico"

    ; Create mkcert shortcuts
    CreateShortcut "$INSTDIR\mkcert install.lnk" "$INSTDIR\mkcert.exe" "-install" "$INSTDIR\Icons\ca-install.ico"
    CreateShortcut "$INSTDIR\mkcert uninstall.lnk" "$INSTDIR\mkcert.exe" "-uninstall" "$INSTDIR\Icons\ca-uninstall.ico"

    ; Initialize mkcert
    MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to enable trusted https. Please accept the mkcert dialog box that may follow."
    nsExec::ExecToLog '$INSTDIR\mkcert.exe -install'
    Pop $R0
    MessageBox MB_OK "DEBUG: mkcert -install result: $R0"
    ${If} $R0 = 0
        WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup" 1
    ${EndIf}

    ; Add to PATH (needed for ddev-hostname.exe)
    DetailPrint "DEBUG: Adding $INSTDIR to PATH"
    EnVar::SetHKLM
    EnVar::AddValue "Path" "$INSTDIR"

    DetailPrint "Docker CE installation completed."
    MessageBox MB_ICONINFORMATION|MB_OK "DDEV WSL2 Docker CE installation completed successfully."
FunctionEnd

Function InstallWSL2DockerDesktop
    DetailPrint "DEBUG: Starting InstallWSL2DockerDesktop"
    MessageBox MB_OK "DEBUG: Entered InstallWSL2DockerDesktop"

    ; Check if Docker Desktop is installed
    ${If} ${FileExists} "$PROGRAMFILES\Docker\Docker\Docker Desktop.exe"
        ; Start Docker Desktop if not running
        nsExec::ExecToLog 'docker version'
        Pop $0
        MessageBox MB_OK "DEBUG: docker version result: $0"
        ${If} $0 != 0
            DetailPrint "Starting Docker Desktop..."
            MessageBox MB_OK "DEBUG: About to run: docker desktop start"
            nsExec::ExecToLog 'docker desktop start'
            Sleep 10000 ; Wait for Docker to start
        ${EndIf}
    ${Else}
        MessageBox MB_ICONSTOP|MB_OK "Docker Desktop is not installed. Please install Docker Desktop with WSL2 backend first."
        Abort "Docker Desktop not found"
    ${EndIf}

    ; Enable WSL2 integration
    DetailPrint "Ensuring WSL2 integration is enabled..."
    MessageBox MB_OK "DEBUG: About to run: wsl --set-default-version 2"
    nsExec::ExecToLog 'wsl --set-default-version 2'

    DetailPrint "Docker Desktop configuration completed."
    MessageBox MB_ICONINFORMATION|MB_OK "DDEV WSL2 Docker Desktop installation completed successfully."
FunctionEnd

Function InstallTraditionalWindows
    DetailPrint "DEBUG: Starting InstallTraditionalWindows"
    MessageBox MB_OK "DEBUG: Entered InstallTraditionalWindows"

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
