; ddev.nsi
;
; This script is based on example2.nsi. It remembers the directory,
; uninstall support and (optionally) installs start menu shortcuts.
;
; It will install ddev.exe into $PROGRAMFILES64/ddev,


; ---------------------------------------------------------------------------
; Initialization
; ---------------------------------------------------------------------------

; Maximize verbosity of compilation
!verbose 4

!define MUI_PRODUCT "DDEV-Local"
!define MUI_VERSION "${VERSION}"
CRCCheck On

!include MUI2.nsh
!include LogicLib.nsh

; The name of the installer
Name "ddev ${MUI_VERSION}"

OutFile "..\.gotmp\bin\windows_amd64\ddev_windows_installer.${MUI_VERSION}.exe"

; The default installation directory
InstallDir $PROGRAMFILES64\ddev

; Registry key to check for directory (so if you install again, it will 
; overwrite the old one automatically)
InstallDirRegKey HKLM "Software\ddev" ""

; Request admin privileges
RequestExecutionLevel admin


; ---------------------------------------------------------------------------
; Interface Settings
; ---------------------------------------------------------------------------

!define MUI_ABORTWARNING


; ---------------------------------------------------------------------------
; Pages
; ---------------------------------------------------------------------------

!define MUI_HEADERIMAGE

!define MUI_WELCOMEPAGE_TITLE "DDEV-Local, a local PHP development environment system"
!define MUI_WELCOMEPAGE_TEXT "From DRUD Tech, https://ddev.com$\r$\nWe welcome your input and contributions. $\r$\nDocs: ddev.readthedocs.io"
!insertmacro MUI_PAGE_WELCOME

!define MUI_LICENSEPAGE_TEXT_TOP "Apache 2.0 License for DDEV-Live (ddev)"
!define MUI_LICENSEPAGE_BUTTON "I agree"
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"

!define MUI_LICENSEPAGE_TEXT_TOP "MIT License for github.com/mattn/sudo"
!define MUI_LICENSEPAGE_BUTTON "I agree"
!insertmacro MUI_PAGE_LICENSE "../.gotmp/bin/windows_amd64/sudo_license.txt"

!define MUI_LICENSEPAGE_TEXT_TOP "GPL License for github.com/winnfsd/winnnsd"
!define MUI_LICENSEPAGE_BUTTON "I agree"
!insertmacro MUI_PAGE_LICENSE "../.gotmp/bin/windows_amd64/winnfsd_license.txt"

!define MUI_LICENSEPAGE_TEXT_TOP "BSD3 License for github.com/FiloSottile/mkcert"
!define MUI_LICENSEPAGE_BUTTON "I agree"
!insertmacro MUI_PAGE_LICENSE "../.gotmp/bin/windows_amd64/mkcert_license.txt"

!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES


!define MUI_FINISHPAGE_TITLE_3LINES "Welcome to DDEV-Local"
!define MUI_FINISHPAGE_TEXT "PLEASE RUN `mkcert -install` and please review the release notes."
!define MUI_FINISHPAGE_SHOWREADME https://github.com/drud/ddev/releases
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Continue to review the release notes."
!define MUI_FINISHPAGE_LINK "github.com/drud/ddev"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/drud/ddev"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"


; ---------------------------------------------------------------------------
; Installer Sections
; ---------------------------------------------------------------------------

Section "!ddev (github.com/drud/ddev)" SecDDEV
  SectionIn RO
  SetOutPath $INSTDIR
  
  File "..\.gotmp\bin\windows_amd64\ddev.exe"

  ; Write the installation path into the registry
  WriteRegStr HKLM SOFTWARE\NSIS_ddev "Install_Dir" "$INSTDIR"
  
  ; Write the uninstall keys for Windows
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev" "DisplayName" "ddev"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev" "UninstallString" '"$INSTDIR\ddev_uninstall.exe"'
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev" "NoRepair" 1
  WriteUninstaller "ddev_uninstall.exe"
SectionEnd

Section "sudo (github.com/mattn/sudo)" SecSudo
  SetOutPath $INSTDIR
  File "..\.gotmp\bin\windows_amd64\sudo.exe"
SectionEnd

Section "nssm (https://nssm.cc/download)" SecNSSM
  SetOutPath $INSTDIR
  SetOverwrite off
  File "..\.gotmp\bin\windows_amd64\nssm.exe"
SectionEnd

Section "WinNFSd (github.com/winnfsd/winnfsd)" SecWinNFSd
  SetOutPath $INSTDIR
  SetOverwrite off
  File "..\.gotmp\bin\windows_amd64\winnfsd.exe"
SectionEnd

Section "mkcert (https://github.com/FiloSottile/mkcert)" SecMkcert
  SetOutPath $INSTDIR
  SetOverwrite off
  File "..\.gotmp\bin\windows_amd64\mkcert.exe"
SectionEnd

Section "windows_ddev_nfs_setup.sh" SecNFSInstall
  SetOutPath $INSTDIR
  File "..\scripts\windows_ddev_nfs_setup.sh"
SectionEnd

Section "Start Menu Shortcuts" SecStartMenu
  CreateDirectory "$SMPROGRAMS\ddev"
  CreateShortcut "$SMPROGRAMS\ddev\Uninstall.lnk" "$INSTDIR\ddev_uninstall.exe" "" "$INSTDIR\ddev_uninstall.exe" 0
SectionEnd


; ---------------------------------------------------------------------------
; Descriptions
; ---------------------------------------------------------------------------

  ;Language strings
  LangString DESC_SecDDEV ${LANG_ENGLISH} "Install DDEV-local (required)."
  LangString DESC_SecSudo ${LANG_ENGLISH} "Sudo for Windows allows for elevated privileges which are used to add hostnames to the Windows hosts file"
  LangString DESC_SecNSSM ${LANG_ENGLISH} "nssm is used to install services, specifically WinNFSd for NFS"
  LangString DESC_SecWinNFSd ${LANG_ENGLISH} "WinNFSd is an optional NFS server that can be used with ddev"
  LangString DESC_SecNFSInstall ${LANG_ENGLISH} "NFS installation script windows_ddev_nfs_setup.sh"
  LangString DESC_SecAddToPath ${LANG_ENGLISH} "Adds the ddev (and sudo) path to the global PATH."
  LangString DESC_SecStartMenu ${LANG_ENGLISH} "Makes a shortcut for the uninstaller on the Start menu."

  ;Assign language strings to sections
  !insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SecDDEV} $(DESC_SecDDEV)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecSudo} $(DESC_SecSudo)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecNSSM} $(DESC_SecNSSM)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecWinNFSd} $(DESC_SecWinNFSd)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecNFSInstall} $(DESC_SecNFSInstall)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecAddToPath} $(DESC_SecAddToPath)
    !insertmacro MUI_DESCRIPTION_TEXT ${SecStartMenu} $(DESC_SecStartMenu)
  !insertmacro MUI_FUNCTION_DESCRIPTION_END


; ---------------------------------------------------------------------------
; Uninstaller Section
; ---------------------------------------------------------------------------

Section "Uninstall"
  
  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev"
  DeleteRegKey HKLM SOFTWARE\NSIS_ddev

  ; Remove files and uninstaller
  Delete $INSTDIR\ddev.exe
  Delete $INSTDIR\sudo.exe
  Delete $INSTDIR\ddev_uninstall.exe

  ; Remove shortcuts, if any
  Delete "$SMPROGRAMS\ddev\*.*"

  ; Remove directories used
  RMDir "$SMPROGRAMS\ddev"
  RMDir "$INSTDIR"

  Push $INSTDIR

SectionEnd


; ---------------------------------------------------------------------------
; Functions
; ---------------------------------------------------------------------------

; Check on startup for docker-compose. If it doesn't exist, warn the user.
Function .onInit
    nsExec::ExecToStack "docker-compose -v"
    Pop $0 # return value/error/timeout
    Pop $1
    ${If} $0 != "0"
      MessageBox MB_OK "Docker and docker-compose do not seem to be installed (or are not available in %PATH%), but they are required for ddev to function. Please install them after you complete ddev installation." /SD IDOK
    ${EndIf}
FunctionEnd
