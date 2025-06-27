!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "FileFunc.nsh"
!include "Sections.nsh"

!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif

Name "DDEV Windows Installer"
OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_windows_${TARGET_ARCH}_installer.exe"

InstallDir "$PROGRAMFILES\DDEV"
RequestExecutionLevel admin

!define PRODUCT_NAME "DDEV"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "DDEV Foundation"

Var /GLOBAL INSTALL_OPTION
Var /GLOBAL DOCKER_OPTION

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

; Welcome page
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
Var ICONS_GROUP
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "${PRODUCT_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT ${REG_UNINST_ROOT}
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINST_KEY}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "NSIS:StartMenuDir"
!insertmacro MUI_PAGE_STARTMENU Application $ICONS_GROUP

; Installation page
!insertmacro MUI_PAGE_INSTFILES

; Finish page with release notes link
!define MUI_FINISHPAGE_SHOWREADME "https://github.com/ddev/ddev/releases/tag/${VERSION}"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Review the release notes"
!insertmacro MUI_PAGE_FINISH

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

Section "Install DDEV"
  ${If} $INSTALL_OPTION == "traditional"
    Call InstallTraditionalWindows
  ${Else}
    Call CheckWSL2Requirements
    ${If} $INSTALL_OPTION == "wsl2-docker-ce"
      StrCpy $DOCKER_OPTION "docker-ce"
    ${Else}
      StrCpy $DOCKER_OPTION "docker-desktop"
    ${EndIf}
    Call InstallWSL2
  ${EndIf}
SectionEnd

Function InstallTraditionalWindows
  DetailPrint "Installing DDEV for traditional Windows..."

  SetOutPath $INSTDIR

  ; Copy core files
  File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev.exe"
  File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev-hostname.exe"
  File /oname=license.txt "..\LICENSE"

  ; Install mkcert
  File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
  File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"

  ; Install icons
  SetOutPath "$INSTDIR\Icons"
  SetOverwrite try
  File /oname=ddev.ico "graphics\ddev-install.ico"

  ; Add to PATH
  EnVar::SetHKLM
  EnVar::AddValue "Path" "$INSTDIR"

  ; Install mkcert root CA
  DetailPrint "Installing mkcert root CA..."
  MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to enable trusted https. Please accept the mkcert dialog box that may follow."
  nsExec::ExecToLog '"$INSTDIR\mkcert.exe" -install'

  ; Create start menu shortcuts
  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\DDEV.lnk" "$INSTDIR\ddev.exe" "" "$INSTDIR\Icons\ddev.ico"

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\ddev_uninstall.exe"

  ; Write uninstaller keys
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "UninstallString" "$INSTDIR\ddev_uninstall.exe"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}" "Publisher" "${PRODUCT_PUBLISHER}"

  DetailPrint "Traditional Windows installation completed."
FunctionEnd

Function CheckWSL2Requirements
  ; ... existing WSL2 check code ...
FunctionEnd

Function InstallWSL2
  ; ... existing WSL2 installation code ...
FunctionEnd

Section "Uninstall"
  ; Remove start menu shortcuts
  RMDir /r "$SMPROGRAMS\${PRODUCT_NAME}"

  ; Remove installed files
  Delete "$INSTDIR\ddev.exe"
  Delete "$INSTDIR\ddev-hostname.exe"
  Delete "$INSTDIR\license.txt"
  Delete "$INSTDIR\mkcert.exe"
  Delete "$INSTDIR\mkcert_license.txt"
  Delete "$INSTDIR\ddev_uninstall.exe"

  ; Remove icons
  RMDir /r "$INSTDIR\Icons"

  ; Remove from PATH
  EnVar::SetHKLM
  EnVar::DeleteValue "Path" "$INSTDIR"

  ; Uninstall mkcert CA if installed
  nsExec::ExecToLog '"$INSTDIR\mkcert.exe" -uninstall'

  ; Remove installation directory
  RMDir "$INSTDIR"

  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

  SetAutoClose true
SectionEnd

Function un.onInit
  MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Are you sure you want to completely remove $(^Name) and all of its components?" IDYES +2
  Abort
FunctionEnd

Function un.onUninstSuccess
  HideWindow
  MessageBox MB_ICONINFORMATION|MB_OK "$(^Name) was successfully removed from your computer."
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
