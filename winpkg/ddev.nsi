/**
 * ddev.nsi - DDEV Setup Script
 *
 * Important hints on extending this installer, please follow this
 * instructions.
 *
 * Adding a new SectionGroup:
 *
 * - add the new SectionGroup to the `Installer Sections` but before the last
 *   Section `-Post`
 *
 *
 * Adding a new Section or SectionGroup:
 *
 * - add the new Section directly to the `Installer Sections` or into a
 *   SectionGroup but before the last Section `-Post`
 * - add new steps to the new Section see existing sections
 * - add a description to the Section see `Section Descriptions`
 *
 *
 * Adding new files:
 *
 * - add new files to a existing Section or create a new one see above
 * - check the output location and overwrite mode
 * - add the new files to the uninstaller
 * - files in Links or Icons directory must not be declared in the uninstaller
 *
 *
 * Adding new start menu short cuts:
 *
 * - add new short cuts to the according Section or create a new one
 * - see comment `Shortcuts` in Section `DDEV` for an example
 * - place the short cuts between `!insertmacro MUI_STARTMENU_WRITE_BEGIN Application`
 *   and `!insertmacro MUI_STARTMENU_WRITE_BEGIN`
 * - start menu short cuts must not be declared in the uninstaller
 */

/**
 * Add local include and plugin directories
 */
!addincludedir include

/**
 * Version fallback for manual compilation
 */
!ifndef VERSION
  !define VERSION 'anonymous-build'
  !define RELEASE_TAG "latest"
!else
  !define RELEASE_TAG "tag/${VERSION}"
!endif

/**
 * Product Settings
 *
 * Common used names, descriptions and URLs used in different places by the
 * installer. For a multilingual installer some of them needs to be localized
 * and therefor defined as LanguageString later in the script.
 */
!define PRODUCT_NAME "DDEV"
!define PRODUCT_NAME_FULL "${PRODUCT_NAME}"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "DDEV Foundation"

!define PRODUCT_WEB_SITE "${PRODUCT_NAME} Website"
!define PRODUCT_WEB_SITE_URL "https://ddev.com"

!define PRODUCT_DOCUMENTATION "${PRODUCT_NAME} Documentation"
!define PRODUCT_DOCUMENTATION_URL "https://ddev.readthedocs.io"

!define PRODUCT_RELEASE_URL "https://github.com/ddev/ddev/releases"
!define PRODUCT_RELEASE_NOTES "${PRODUCT_NAME} Release Notes"
!define PRODUCT_RELEASE_NOTES_URL "${PRODUCT_RELEASE_URL}/${RELEASE_TAG}"

!define PRODUCT_ISSUES "${PRODUCT_NAME} Issues"
!define PRODUCT_ISSUES_URL "https://github.com/ddev/ddev/issues"

!define PRODUCT_PROJECT "${PRODUCT_NAME} GitHub"
!define PRODUCT_PROJECT_URL "https://github.com/ddev/ddev#readme"

/**
 * Registry Settings
 */
!define REG_INSTDIR_ROOT "HKLM"
!define REG_INSTDIR_KEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe"
!define REG_UNINST_ROOT "HKLM"
!define REG_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

/**
 * Configuration
 *
 * Has to be done before including headers
 */
!ifndef TARGET_ARCH # passed on command-line
  !error "TARGET_ARCH define is missing!"
!endif
Var TARGET_ARCH
Var INSTALL_ARCH /* Architecture where installation is happening */

OutFile "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_windows_${TARGET_ARCH}_installer.exe"
Unicode true
SetCompressor /SOLID lzma

InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"

RequestExecutionLevel admin

/**
 * Installer Types
 */
InstType "Full"
InstType "Simple"
InstType "Minimal"

/**
 * Include Headers
 */
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "Sections.nsh"
!include "x64.nsh"
!include "WinVer.nsh"

!include "ddev.nsh"

/**
 * Local macros
 */
Var ChocolateyMode
!macro _Chocolatey _a _b _t _f
  !insertmacro _== $ChocolateyMode `1` `${_t}` `${_f}`
!macroend
!define Chocolatey `"" Chocolatey ""`

/**
 * Names
 */
!define INSTALLER_MODE_SETUP "SETUP"
!define INSTALLER_MODE_UPDATE "UPDATE"
Var InstallerMode
Var InstallerModeCaption
Name "${PRODUCT_NAME_FULL}"
Caption "${PRODUCT_NAME_FULL} ${PRODUCT_VERSION} $InstallerModeCaption"

/**
 * Interface Configuration
 */
!define MUI_ICON "graphics\ddev-install.ico"
!define MUI_UNICON "graphics\ddev-uninstall.ico"

!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "graphics\ddev-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "graphics\ddev-wizard.bmp"

!define MUI_ABORTWARNING

!define MUI_CUSTOMFUNCTION_GUIINIT onGUIInit

/**
 * Language Selection Dialog Settings
 *
 * This enables the remember of the previously chosen language.
 */
!define MUI_LANGDLL_REGISTRY_ROOT ${REG_UNINST_ROOT}
!define MUI_LANGDLL_REGISTRY_KEY "${REG_UNINST_KEY}"
!define MUI_LANGDLL_REGISTRY_VALUENAME "NSIS:Language"

/**
 * Installer Pages
 *
 * Pages shown by the installer are declared here in the showing order.
 */

; Welcome page
!insertmacro MUI_PAGE_WELCOME

; License page
!define MUI_PAGE_CUSTOMFUNCTION_PRE ddevLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE ddevLicLeave
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"

; Components page
Var MkcertSetup
!define MUI_PAGE_CUSTOMFUNCTION_PRE ComponentsPre
!insertmacro MUI_PAGE_COMPONENTS

; License page mkcert
!define MUI_PAGE_HEADER_TEXT "License Agreement for mkcert"
!define MUI_PAGE_HEADER_SUBTEXT "Please review the license terms before installing mkcert."
!define MUI_PAGE_CUSTOMFUNCTION_PRE mkcertLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE mkcertLicLeave
!insertmacro MUI_PAGE_LICENSE "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"

; Directory page
!define MUI_PAGE_CUSTOMFUNCTION_PRE DirectoryPre
!insertmacro MUI_PAGE_DIRECTORY

; Start menu page
Var ICONS_GROUP
!define MUI_STARTMENUPAGE_DEFAULTFOLDER "${PRODUCT_NAME}"
!define MUI_STARTMENUPAGE_REGISTRY_ROOT ${REG_UNINST_ROOT}
!define MUI_STARTMENUPAGE_REGISTRY_KEY "${REG_UNINST_KEY}"
!define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "NSIS:StartMenuDir"
!define MUI_PAGE_CUSTOMFUNCTION_PRE StartMenuPre
!insertmacro MUI_PAGE_STARTMENU Application $ICONS_GROUP

; Instfiles page
!insertmacro MUI_PAGE_INSTFILES

; Finish page
!define MUI_FINISHPAGE_SHOWREADME "${PRODUCT_RELEASE_NOTES_URL}"
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Review the release notes"
!define MUI_FINISHPAGE_LINK "${PRODUCT_PROJECT} (${PRODUCT_PROJECT_URL})"
!define MUI_FINISHPAGE_LINK_LOCATION ${PRODUCT_PROJECT_URL}
!insertmacro MUI_PAGE_FINISH

/**
 * Uninstaller Pages
 *
 * Currently we use a minimal uninstaller without a GUI. Only INSTFILES is
 * used to process the sections.
 */

; Instfiles page
!insertmacro MUI_UNPAGE_INSTFILES

/**
 * Language Files
 *
 * Base language of this installer is English, additional languages can be
 * added here. Internal used strings must be defined below see
 * `Language Strings`.
 */
!insertmacro MUI_LANGUAGE "English"

/**
 * Reserve Files
 *
 * Files used in a early stage e.g. .onInit should be declared here to speed
 * up the installer start.
 */
!insertmacro MUI_RESERVEFILE_LANGDLL ; Language selection dialog
ReserveFile /plugin EnVar.dll
ReserveFile /plugin nsExec.dll
ReserveFile /plugin INetC.dll

/**
 * Installer Sections
 *
 * Steps processed by the installer.
 */

/**
 * DDEV group
 */
SectionGroup /e "${PRODUCT_NAME_FULL}"
  /**
   * DDEV application install
   */
  Section "${PRODUCT_NAME_FULL}" SecDDEV
    ; Force installation
    SectionIn 1 2 3 RO
    SetOutPath "$INSTDIR"

    ; Important to enable downgrades from non stable
    SetOverwrite on

    ; Copy files
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev.exe"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\ddev_hostname.exe"
    File /oname=license.txt "..\LICENSE"

    ; Install icons
    SetOutPath "$INSTDIR\Icons"
    SetOverwrite try
    File /oname=ddev.ico "graphics\ddev-install.ico"

    ; Clean up current user PATH created multiple times from old installer
    EnVar::SetHKCU
    EnVar::DeleteValue "Path" "$INSTDIR"

    ; Power off all projects
    ${If} ${DdevPowerOff} "$INSTDIR\DDEV"
      DetailPrint "${PRODUCT_NAME} projects are powered off now"
    ${Else}
      Pop $R0 ; Output
      DetailPrint "${PRODUCT_NAME} power off failed: $R0"
    ${EndIf}

    ; Shortcuts
    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application

    CreateDirectory "$INSTDIR\Links"
    CreateDirectory "$SMPROGRAMS\$ICONS_GROUP"

    ; DDEV Website
    WriteIniStr "$INSTDIR\Links\${PRODUCT_WEB_SITE}.url" "InternetShortcut" "URL" "${PRODUCT_WEB_SITE_URL}"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\${PRODUCT_WEB_SITE}.lnk" "$INSTDIR\Links\${PRODUCT_WEB_SITE}.url" "" "$INSTDIR\Icons\ddev.ico"

    ; DDEV Doc
    WriteIniStr "$INSTDIR\Links\${PRODUCT_DOCUMENTATION}.url" "InternetShortcut" "URL" "${PRODUCT_DOCUMENTATION_URL}"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\${PRODUCT_DOCUMENTATION}.lnk" "$INSTDIR\Links\${PRODUCT_DOCUMENTATION}.url" "" "$INSTDIR\Icons\ddev.ico"

    ; DDEV Release Notes
    WriteIniStr "$INSTDIR\Links\${PRODUCT_RELEASE_NOTES}.url" "InternetShortcut" "URL" "${PRODUCT_RELEASE_NOTES_URL}"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\${PRODUCT_RELEASE_NOTES}.lnk" "$INSTDIR\Links\${PRODUCT_RELEASE_NOTES}.url" "" "$INSTDIR\Icons\ddev.ico"

    ; DDEV Issues
    WriteIniStr "$INSTDIR\Links\${PRODUCT_ISSUES}.url" "InternetShortcut" "URL" "${PRODUCT_ISSUES_URL}"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\${PRODUCT_ISSUES}.lnk" "$INSTDIR\Links\${PRODUCT_ISSUES}.url" "" "$INSTDIR\Icons\ddev.ico"

    ; DDEV Source Code
    WriteIniStr "$INSTDIR\Links\${PRODUCT_PROJECT}.url" "InternetShortcut" "URL" "${PRODUCT_PROJECT_URL}"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\${PRODUCT_PROJECT}.lnk" "$INSTDIR\Links\${PRODUCT_PROJECT}.url" "" "$INSTDIR\Icons\ddev.ico"

    !insertmacro MUI_STARTMENU_WRITE_END
  SectionEnd

  /**
   * Add install directory to Path variable
   */
  Section "Add to PATH" SecAddToPath
    SectionIn 1 2 3
    EnVar::SetHKLM
    EnVar::AddValue "Path" "$INSTDIR"
  SectionEnd
SectionGroupEnd


/**
 * mkcert group
 */
SectionGroup /e "mkcert"
  /**
   * mkcert application install
   */
  Section "mkcert" SecMkcert
    SectionIn 1 2
    SetOutPath "$INSTDIR"
    SetOverwrite try

    ; Copy files
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert.exe"
    File "..\.gotmp\bin\windows_${TARGET_ARCH}\mkcert_license.txt"

    ; Install icons
    SetOutPath "$INSTDIR\Icons"
    SetOverwrite try
    File /oname=ca-install.ico "graphics\ca-install.ico"
    File /oname=ca-uninstall.ico "graphics\ca-uninstall.ico"

    ; Shortcuts
    CreateShortcut "$INSTDIR\mkcert install.lnk" "$INSTDIR\mkcert.exe" "-install" "$INSTDIR\Icons\ca-install.ico"
    CreateShortcut "$INSTDIR\mkcert uninstall.lnk" "$INSTDIR\mkcert.exe" "-uninstall" "$INSTDIR\Icons\ca-uninstall.ico"

    !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
    CreateDirectory "$SMPROGRAMS\$ICONS_GROUP\mkcert"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\mkcert\mkcert install trusted https.lnk" "$INSTDIR\mkcert install.lnk"
    CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\mkcert\mkcert uninstall trusted https.lnk" "$INSTDIR\mkcert uninstall.lnk"
    !insertmacro MUI_STARTMENU_WRITE_END
  SectionEnd

  /**
   * mkcert setup
   */
  Section "Setup mkcert" SecMkcertSetup
    ; Install in non choco mode only
    ${IfNot} ${Chocolatey}
      MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to enable trusted https. Please accept the mkcert dialog box that may follow."

      ; Run setup
      nsExec::ExecToLog '"$INSTDIR\mkcert.exe" -install'
      Pop $R0 ; get return value

      ; Check return value and write setup status to registry on success
      ${If} $R0 = 0
        WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup" 1
      ${EndIf}
    ${EndIf}
  SectionEnd
SectionGroupEnd

/**
 * Last processed section
 *
 * Insert new section groups and sections before this point!
 */
Section -Post
  ; Write the uninstaller
  WriteUninstaller "$INSTDIR\ddev_uninstall.exe"

  ; Remember install directory for updates
  WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "" "$INSTDIR\ddev.exe"
  WriteRegStr ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "Path" "$INSTDIR"

  ; Clean up registry keys mistakenly created in old installers
  SetRegView 32
  DeleteRegKey HKLM "SOFTWARE\NSIS_ddev"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\ddev"
  SetRegView lastused

  ; Write uninstaller keys for Windows
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "UninstallString" "$INSTDIR\ddev_uninstall.exe"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayIcon" "$INSTDIR\Icons\ddev.ico"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE_URL}"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "URLUpdateInfo" "${PRODUCT_RELEASE_URL}"
  WriteRegStr ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NoRepair" 1

  ; Shortcuts
  !insertmacro MUI_STARTMENU_WRITE_BEGIN Application
  ; Uninstaller
  CreateShortCut "$SMPROGRAMS\$ICONS_GROUP\Uninstall ${PRODUCT_NAME_FULL}.lnk" "$INSTDIR\ddev_uninstall.exe"
  !insertmacro MUI_STARTMENU_WRITE_END
SectionEnd

/**
 * Language Strings
 *
 * Provide language dependant descriptions
 */
LangString DESC_SecDDEV ${LANG_ENGLISH} "Install ${PRODUCT_NAME_FULL} (required)"
LangString DESC_SecAddToPath ${LANG_ENGLISH} "Add the ${PRODUCT_NAME} directory to the global PATH"
LangString DESC_SecMkcert ${LANG_ENGLISH} "mkcert (github.com/FiloSottile/mkcert) is a simple tool for making locally-trusted development certificates. It requires no configuration"
LangString DESC_SecMkcertSetup ${LANG_ENGLISH} "Run `mkcert -install` to setup a local CA"

/**
 * Section Descriptions
 *
 * Assign a description to each section
 */
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDDEV} $(DESC_SecDDEV)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAddToPath} $(DESC_SecAddToPath)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMkcert} $(DESC_SecMkcert)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMkcertSetup} $(DESC_SecMkcertSetup)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

/**
 * Installer Macros
 */
!macro _IsSetupMode _a _b _t _f
  !insertmacro _== $InstallerMode `${INSTALLER_MODE_SETUP}` `${_t}` `${_f}`
  ;!insertmacro _If true `$InstallerMode` `==` `${INSTALLER_MODE_SETUP}`
  ;!insertmacro _FileExists `${_a}` `${_b}\ddev.exe` `${_t}` `${_f}`
!macroend
!define IsSetupMode `"" IsSetupMode ""`

!macro _IsUpdateMode _a _b _t _f
  !insertmacro _== $InstallerMode `${INSTALLER_MODE_UPDATE}` `${_t}` `${_f}`
  ;!insertmacro _If true `$InstallerMode` `==` `${INSTALLER_MODE_UPDATE}`
  ;!insertmacro _FileExists `${_a}` `${_b}\ddev.exe` `${_t}` `${_f}`
!macroend
!define IsUpdateMode `"" IsUpdateMode ""`

/**
 * Installer Functions
 *
 * Place functions used in the installer here. Function names must not start
 * with `un.`
 */
Function GetOSArch
    ; Get TARGET_ARCH into a variable from the argument/define
    StrCpy $TARGET_ARCH ${TARGET_ARCH}

    ; First, check the PROCESSOR_ARCHITEW6432 environment variable (used in 32-bit processes on 64-bit systems)
    ReadEnvStr $INSTALL_ARCH "PROCESSOR_ARCHITEW6432"

    ${If} $INSTALL_ARCH == ""
        ; If PROCESSOR_ARCHITEW6432 is not set, fall back to PROCESSOR_ARCHITECTURE
        ReadEnvStr $INSTALL_ARCH "PROCESSOR_ARCHITECTURE"
    ${EndIf}

    ; Check for common architectures
    ${If} $INSTALL_ARCH == "AMD64"
        StrCpy $INSTALL_ARCH "amd64"
    ${ElseIf} $INSTALL_ARCH == "ARM64"
        StrCpy $INSTALL_ARCH "arm64"
    ${Else}
        StrCpy $INSTALL_ARCH "unknown"
    ${EndIf}
FunctionEnd



/**
 * Initialization, called on installer start
 */
Function .onInit
  ; Check OS architecture
  Call GetOSArch

  ; Compare detected architecture ($ARCH) with the target architecture ($NSIS_ARCH)
  ${If} $INSTALL_ARCH != $TARGET_ARCH
    MessageBox MB_ICONSTOP|MB_OK "Unsupported CPU architecture: $INSTALL_ARCH . This installer is built for ${TARGET_ARCH}."
    Abort "Unsupported CPU architecture!"
  ${EndIf}

  ; Check Windows version
  ${IfNot} ${AtLeastWin10}
    MessageBox MB_ICONSTOP|MB_OK "Unsupported Windows version, $(^Name) requires Windows 10 or later."
    Abort "Unsupported Windows version!"
  ${EndIf}

  InitPluginsDir

  ; Switch to 64 bit view and disable FS redirection
  SetRegView 64
  ${DisableX64FSRedirection}

  ; Show language select dialog
  !insertmacro MUI_LANGDLL_DISPLAY

  ; Load last $INSTDIR for upgrades. InstallDirRegKey does not work because of
  ; the usage of SetRegView 64
  ReadRegStr $R0 ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" "Path"

  ${If} ${Errors}
    ; Backward compatibility with older installers
    ReadRegStr $R0 ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" ""

    ${If} ${Errors}
      SetRegView 32
      ReadRegStr $R0 ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}" ""
      SetRegView lastused
    ${EndIf}

    GetFullPathName $R0 $R0
  ${EndIf}

  ; Set last $INSTDIR and $InstallerMode
  ${If} ${DdevIsInstalled} "$R0"
    StrCpy $INSTDIR $R0
    StrCpy $InstallerMode ${INSTALLER_MODE_UPDATE}
    StrCpy $InstallerModeCaption "Update"
  ${Else}
    StrCpy $InstallerMode ${INSTALLER_MODE_SETUP}
    StrCpy $InstallerModeCaption "Setup"
  ${EndIf}

  ; Initialize global variables
  StrCpy $mkcertSetup ""

FunctionEnd

/**
 * GUI initialization, called before window is shown
 */
Function onGUIInit
  ; Read setup status from registry
  ${IfNot} ${Silent}
    ReadRegDWORD $mkcertSetup ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup"
  ${Else}
    StrCpy $mkcertSetup "1" ; this disables the auto selection
  ${EndIf}
FunctionEnd

/**
 * Auto select/unselect components
 */
Function .onSelChange
  ; Apply special selections on install type change
  ${If} $0 = -1
    ${If} $mkcertSetup != 1
    ${AndIf} ${SectionIsSelected} ${SecMkcert}
    ${AndIfNot} ${Silent}
      !insertmacro SelectSection ${SecMkcertSetup}
    ${EndIf}
  ${Else}
    ; Unselect if required component is not selected
    ${If} $0 = ${SecMkcert}
      ${IfNot} ${SectionIsSelected} $0
        !insertmacro UnselectSection ${SecMkcertSetup}
      ${EndIf}
    ${EndIf}

    ; Select required component
    ${If} $0 = ${SecMkcertSetup}
      ${If} ${SectionIsSelected} $0
        !insertmacro SelectSection ${SecMkcert}
      ${EndIf}
    ${EndIf}
  ${EndIf}
FunctionEnd

/**
 * Disable not applicable sections
 */
Function ComponentsPre
  ${If} $mkcertSetup != 1
  ${AndIf} ${SectionIsSelected} ${SecMkcert}
  ${AndIfNot} ${Silent}
    !insertmacro SelectSection ${SecMkcertSetup}
  ${Else}
    !insertmacro UnselectSection ${SecMkcertSetup}
  ${EndIf}
FunctionEnd

/**
 * Disable ddev license page if it was already accepted before
 */
Function ddevLicPre
  ReadRegDWORD $R0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:ddevLicenseAccepted"
  ${If} $R0 = 1
    Abort
  ${EndIf}
FunctionEnd

/**
 * Set ddev license accepted flag
 */
Function ddevLicLeave
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:ddevLicenseAccepted" 0x00000001
FunctionEnd


/**
 * Disable mkcert license page if component is not selected or already accepted before
 */
Function mkcertLicPre
  ReadRegDWORD $R0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertLicenseAccepted"
  ${If} $R0 = 1
  ${OrIfNot} ${SectionIsSelected} ${SecMkcert}
    Abort
  ${EndIf}
FunctionEnd

/**
 * Set mkcert license accepted flag
 */
Function mkcertLicLeave
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertLicenseAccepted" 0x00000001
FunctionEnd

/**
 * Disable on updates
 */
Function DirectoryPre
  ${If} ${IsUpdateMode}
    Abort
  ${EndIf}
FunctionEnd

/**
 * Disable on updates
 */
Function StartMenuPre
  ${If} ${IsUpdateMode}
    Abort
  ${EndIf}
FunctionEnd


/**
 * Uninstaller Section
 *
 * Steps processed by the uninstaller.
 */
Section Uninstall
  ; Uninstall mkcert
  Call un.mkcertUninstall

  ; Remove install directory from system and current user PATH
  EnVar::SetHKCU
  EnVar::DeleteValue "Path" "$INSTDIR"
  EnVar::SetHKLM
  EnVar::DeleteValue "Path" "$INSTDIR"

  ; Remove installed files
  Delete "$INSTDIR\ddev_uninstall.exe"

  Delete "$INSTDIR\mkcert uninstall.lnk"
  Delete "$INSTDIR\mkcert install.lnk"
  Delete "$INSTDIR\mkcert_license.txt"
  Delete "$INSTDIR\mkcert.exe"

  Delete "$INSTDIR\license.txt"
  Delete "$INSTDIR\ddev.exe"

  ; Load start menu folder
  !insertmacro MUI_STARTMENU_GETFOLDER "Application" $ICONS_GROUP

  ; Remove created directories
  RMDir /r "$SMPROGRAMS\$ICONS_GROUP"
  RMDir /r "$INSTDIR\Links"
  RMDir /r "$INSTDIR\Icons"
  RMDir "$INSTDIR" ; do not delete recursively!

  ; Show a hint in case install directory was not removed
  ${If} ${FileExists} "$INSTDIR"
    MessageBox MB_ICONINFORMATION|MB_OK "Note: $INSTDIR could not be removed!"
  ${EndIf}

  ; Clean up registry
  DeleteRegKey ${REG_UNINST_ROOT} "${REG_UNINST_KEY}"
  DeleteRegKey ${REG_INSTDIR_ROOT} "${REG_INSTDIR_KEY}"

  ; Close uninstaller window
  SetAutoClose true
SectionEnd


/**
 * Uninstaller Functions
 *
 * Place functions used in the uninstaller here. Function names must start
 * with `un.`
 */

/**
 * Initialization, called on uninstaller start
 */
Function un.onInit
  ; Load language
  !insertmacro MUI_UNGETLANGUAGE

  MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Are you sure you want to completely remove $(^Name) and all of its components?" /SD IDYES IDYES DoUninstall
  Abort

DoUninstall:
  ; Switch to 64 bit view and disable FS redirection
  SetRegView 64
  ${DisableX64FSRedirection}
FunctionEnd

/**
 * Successful uninstall, show information to user
 */
Function un.onUninstSuccess
  HideWindow
  MessageBox MB_ICONINFORMATION|MB_OK "$(^Name) was successfully removed from your computer." /SD IDOK
FunctionEnd

/**
 * Run mkcert uninstall with a previous information about the following warning
 * if mkcert is installed
 */
Function un.mkcertUninstall
  ${If} ${FileExists} "$INSTDIR\mkcert.exe"
    Push $0

    ; Read setup status from registry
    ReadRegDWORD $0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:mkcertSetup"

    ; Check if setup has done
    ${If} $0 == 1
      ; Get user confirmation
      MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "mkcert was found in this installation. Do you like to remove the mkcert configuration?" /SD IDNO IDYES +2
      Goto Skip

      MessageBox MB_ICONINFORMATION|MB_OK "Now running mkcert to disable trusted https. Please accept the mkcert dialog box that may follow."

      nsExec::ExecToLog '"$INSTDIR\mkcert.exe" -uninstall'
      Pop $0 ; get return value

    Skip:
    ${EndIf}

    Pop $0
  ${EndIf}
FunctionEnd
