/**
 * ddev.nsi - DDEV Local Setup Script
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
!define PRODUCT_NAME_FULL "${PRODUCT_NAME} Local"
!define PRODUCT_VERSION "${VERSION}"
!define PRODUCT_PUBLISHER "Drud Technology LLC"

!define PRODUCT_WEB_SITE "${PRODUCT_NAME} Website"
!define PRODUCT_WEB_SITE_URL "https://www.ddev.com"

!define PRODUCT_DOCUMENTATION "${PRODUCT_NAME} Documentation"
!define PRODUCT_DOCUMENTATION_URL "https://ddev.readthedocs.io"

!define PRODUCT_RELEASE_URL "https://github.com/drud/ddev/releases"
!define PRODUCT_RELEASE_NOTES "${PRODUCT_NAME} Release Notes"
!define PRODUCT_RELEASE_NOTES_URL "${PRODUCT_RELEASE_URL}/${RELEASE_TAG}"

!define PRODUCT_ISSUES "${PRODUCT_NAME} Issues"
!define PRODUCT_ISSUES_URL "https://github.com/drud/ddev/issues"

!define PRODUCT_PROJECT "${PRODUCT_NAME} GitHub"
!define PRODUCT_PROJECT_URL "https://github.com/drud/ddev#readme"



/**
 * Registry Settings
 */
!define REG_INSTDIR_ROOT "HKLM"
!define REG_INSTDIR_KEY "Software\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe"
!define REG_UNINST_ROOT "HKLM"
!define REG_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"



/**
 * Third Party Applications
 */
!define WINNFSD_NAME "WinNFSd"
!define WINNFSD_VERSION "2.4.0"
!define WINNFSD_SETUP "WinNFSd.exe"
!define WINNFSD_URL "https://github.com/winnfsd/winnfsd/releases/download/${WINNFSD_VERSION}/WinNFSd.exe"

!define NSSM_NAME "NSSM"
!define NSSM_VERSION "2.24-101-g897c7ad"
!define NSSM_SETUP "nssm.exe"
!define NSSM_URL "https://github.com/drud/nssm/releases/download/${NSSM_VERSION}/nssm.exe"



/**
 * Configuration
 *
 * Has to be done before including headers
 */
OutFile "..\.gotmp\bin\windows_amd64\ddev_windows_installer.${PRODUCT_VERSION}.exe"
Unicode true
SetCompressor /SOLID lzma

InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"

RequestExecutionLevel admin
;ManifestSupportedOS



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
;!include "Memento.nsh"
!include "Sections.nsh"
!include "x64.nsh"
!include "WinVer.nsh"

!include "ddev.nsh"
!ifndef DOCKER_EXCLUDE
  !include "docker.nsh"
!endif



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
 * This enables the remeber of the previously choosen language.
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

; License page sudo
!define MUI_PAGE_HEADER_TEXT "License Agreement for sudo"
!define MUI_PAGE_HEADER_SUBTEXT "Please review the license terms before installing sudo."
!define MUI_PAGE_CUSTOMFUNCTION_PRE sudoLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE sudoLicLeave
!insertmacro MUI_PAGE_LICENSE "..\.gotmp\bin\windows_amd64\sudo_license.txt"

; Components page
!ifdef DOCKER_NSH
  Var DockerVisible
  Var DockerSelected
!endif
Var MkcertSetup
!define MUI_PAGE_CUSTOMFUNCTION_PRE ComponentsPre
!insertmacro MUI_PAGE_COMPONENTS

; License page mkcert
!define MUI_PAGE_HEADER_TEXT "License Agreement for mkcert"
!define MUI_PAGE_HEADER_SUBTEXT "Please review the license terms before installing mkcert."
!define MUI_PAGE_CUSTOMFUNCTION_PRE mkcertLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE mkcertLicLeave
!insertmacro MUI_PAGE_LICENSE "..\.gotmp\bin\windows_amd64\mkcert_license.txt"

; License page WinNFSd
!define MUI_PAGE_HEADER_TEXT "License Agreement for WinNFSd"
!define MUI_PAGE_HEADER_SUBTEXT "Please review the license terms before installing WinNFSd."
!define MUI_PAGE_CUSTOMFUNCTION_PRE winNFSdLicPre
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE winNFSdLicLeave
!insertmacro MUI_PAGE_LICENSE "licenses\winnfsd_license.txt"

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
 * Version Information
 *
 * To use version information the VERSION constant has to be splitted into
 * the 4 version parts.
 */
;VIAddVersionKey /LANG=${LANG_ENGLISH} "ProductName" "${PRODUCT_NAME_FULL}"
;VIAddVersionKey /LANG=${LANG_ENGLISH} "Comments" "A test comment"
;VIAddVersionKey /LANG=${LANG_ENGLISH} "CompanyName" "${PRODUCT_PUBLISHER}"
;VIAddVersionKey /LANG=${LANG_ENGLISH} "LegalTrademarks" "Test Application is a trademark of Fake company"
;VIAddVersionKey /LANG=${LANG_ENGLISH} "LegalCopyright" "https://github.com/drud/ddev/raw/master/LICENSE"
;VIAddVersionKey /LANG=${LANG_ENGLISH} "FileDescription" "Windows Installer of ${PRODUCT_NAME_FULL}"
;ProductName
;Comments
;CompanyName
;LegalCopyright
;FileDescription
;FileVersion
;ProductVersion
;InternalName
;LegalTrademarks
;OriginalFilename
;PrivateBuild
;SpecialBuild
;StrCpy
;VIProductVersion 1.2.3.4
;VIFileVersion 1.2.3.4



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
    File "..\.gotmp\bin\windows_amd64\ddev.exe"
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
 * Docker download and install
 */
!ifdef DOCKER_NSH
Section /o "${DOCKER_DESKTOP_NAME}" SecDocker
  ; Set URL and temporary file name
  !define DOCKER_DESKTOP_INSTALLER "$TEMP\${DOCKER_DESKTOP_SETUP}"

  ; Download installer
  INetC::get /CANCELTEXT "Skip download" /QUESTION "" "${DOCKER_DESKTOP_URL}" "${DOCKER_DESKTOP_INSTALLER}" /END
  Pop $R0 ; return value = exit code, "OK" if OK

  ; Check download result
  ${If} $R0 = "OK"
    ; Execute installer
    ExecWait '"${DOCKER_DESKTOP_INSTALLER}"' $R0

    ; Delete installer
    Delete "${DOCKER_DESKTOP_INSTALLER}"

    ${If} $R0 != 0
      ; Installation failed, show message and continue
      SetDetailsView show
      DetailPrint "Installation of `${DOCKER_DESKTOP_NAME}` failed:"
      DetailPrint " $R0"
      MessageBox MB_ICONEXCLAMATION|MB_OK "Installation of `${DOCKER_DESKTOP_NAME}` has failed, please download and install once this installation has finished. Continue the resting installation."
    ${EndIf}
  ${Else}
    ; Download failed, show message and continue
    SetDetailsView show
    DetailPrint "Download of `${DOCKER_DESKTOP_NAME}` failed:"
    DetailPrint " $R0"
    MessageBox MB_ICONEXCLAMATION|MB_OK "Download of `${DOCKER_DESKTOP_NAME}` has failed, please download and install once this installation has finished. Continue the resting installation."
  ${EndIf}

  !undef DOCKER_DESKTOP_INSTALLER
SectionEnd
!endif ; DOCKER_NSH

/**
 * sudo application install
 */
Section "sudo" SecSudo
  ; Force installation
  SectionIn 1 2 3 RO
  SetOutPath "$INSTDIR"
  SetOverwrite try

  ; Copy files
  File "..\.gotmp\bin\windows_amd64\sudo.exe"
  File "..\.gotmp\bin\windows_amd64\sudo_license.txt"
SectionEnd

/**
 * mkcert group
 */
SectionGroup /e "mkcert"
  /**
   * mkcert application install
   */
  Section "mkcert" SecMkcert
    ; Install in non choco mode only
    ${IfNot} ${Chocolatey}
      SectionIn 1 2
      SetOutPath "$INSTDIR"
      SetOverwrite try

      ; Copy files
      File "..\.gotmp\bin\windows_amd64\mkcert.exe"
      File "..\.gotmp\bin\windows_amd64\mkcert_license.txt"

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
    ${EndIf}
  SectionEnd

  /**
   * mkcert setup
   */
  Section "Setup mkcert" SecMkcertSetup
    ; Install in non silent and choco mode only
    ${IfNot} ${Silent}
    ${AndIfNot} ${Chocolatey}
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
 * WinNFSd group
 */
SectionGroup /e "WinNFSd"
  /**
   * WinNFSd application install
   */
  Section "${WINNFSD_NAME}" SecWinNFSd
    SectionIn 1
    SetOutPath "$INSTDIR"
    SetOverwrite try

    ; Copy files
    File "licenses\winnfsd_license.txt"
    File "..\scripts\windows_ddev_nfs_setup.sh"

    ; Set URL and temporary file name
    !define WINNFSD_DEST "$INSTDIR\${WINNFSD_SETUP}"

    ; Download installer
    INetC::get /CANCELTEXT "Skip download" /QUESTION "" "${WINNFSD_URL}" "${WINNFSD_DEST}" /END
    Pop $R0 ; return value = exit code, "OK" if OK

    ; Check download result
    ${If} $R0 != "OK"
      ; Download failed, show message and continue
      SetDetailsView show
      DetailPrint "Download of `${WINNFSD_NAME}` failed:"
      DetailPrint " $R0"
      MessageBox MB_ICONEXCLAMATION|MB_OK "Download of `${WINNFSD_NAME}` has failed, please download it to the DDEV installation folder `$INSTDIR` once this installation has finished. Continue the resting installation."
    ${EndIf}

    !undef WINNFSD_DEST
  SectionEnd

  /**
   * NSSM application install
   */
  Section "${NSSM_NAME}" SecNSSM
    ; Install in non choco mode only
    ${IfNot} ${Chocolatey}
      SectionIn 1
      SetOutPath "$INSTDIR"
      SetOverwrite try

      ; Set URL and temporary file name
      !define NSSM_DEST "$INSTDIR\${NSSM_SETUP}"

      ; Download installer
      INetC::get /CANCELTEXT "Skip download" /QUESTION "" "${NSSM_URL}" "${NSSM_DEST}" /END
      Pop $R0 ; return value = exit code, "OK" if OK

      ; Check download result
      ${If} $R0 != "OK"
        ; Download failed, show message and continue
        SetDetailsView show
        DetailPrint "Download of `${NSSM_NAME}` failed:"
        DetailPrint " $R0"
        MessageBox MB_ICONEXCLAMATION|MB_OK "Download of `${NSSM_NAME}` has failed, please download it to the DDEV installation folder `$INSTDIR` once this installation has finished. Continue the resting installation."
      ${EndIf}

      !undef NSSM_DEST
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

  ; Remeber install directory for updates
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
LangString DESC_SecAddToPath ${LANG_ENGLISH} "Add the ${PRODUCT_NAME} (and sudo) directory to the global PATH"
!ifdef DOCKER_NSH
LangString DESC_SecDocker ${LANG_ENGLISH} "Download and install ${DOCKER_DESKTOP_NAME} (www.docker.com) which do not seem to be installed, but is required for $(^Name) to function"
!endif ; DOCKER_NSH
LangString DESC_SecSudo ${LANG_ENGLISH} "Sudo for Windows (github.com/ mattn/sudo) allows for elevated privileges which are used to add hostnames to the Windows hosts file (required)"
LangString DESC_SecMkcert ${LANG_ENGLISH} "mkcert (github.com/ FiloSottile/mkcert) is a simple tool for making locally-trusted development certificates. It requires no configuration"
LangString DESC_SecMkcertSetup ${LANG_ENGLISH} "Run `mkcert -install` to setup a local CA"
LangString DESC_SecWinNFSd ${LANG_ENGLISH} "WinNFSd (github.com/ winnfsd/winnfsd) is an optional NFS server that can be used with ${PRODUCT_NAME_FULL}"
LangString DESC_SecNSSM ${LANG_ENGLISH} "NSSM (nssm.cc) is used to install services, specifically WinNFSd for NFS"



/**
 * Section Descriptions
 *
 * Assign a decription to each section
 */
!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDDEV} $(DESC_SecDDEV)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAddToPath} $(DESC_SecAddToPath)
  !ifdef DOCKER_NSH
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDocker} $(DESC_SecDocker)
  !endif ; DOCKER_NSH
  !insertmacro MUI_DESCRIPTION_TEXT ${SecSudo} $(DESC_SecSudo)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMkcert} $(DESC_SecMkcert)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMkcertSetup} $(DESC_SecMkcertSetup)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecWinNFSd} $(DESC_SecWinNFSd)
  !insertmacro MUI_DESCRIPTION_TEXT ${SecNSSM} $(DESC_SecNSSM)
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

/**
 * Initialization, called on installer start
 */
Function .onInit
  ; Check OS architecture, 64 bit supported only
  ${IfNot} ${IsNativeAMD64}
    MessageBox MB_ICONSTOP|MB_OK "Unsupported CPU architecture, $(^Name) runs on 64 bit only."
    Abort "Unsupported CPU architecture!"
  ${EndIf}

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
  !ifdef DOCKER_NSH
  StrCpy $DockerVisible ""
  StrCpy $DockerSelected ""
  !endif ; DOCKER_NSH
  StrCpy $mkcertSetup ""

  ; Check parameters
  ${GetParameters} $R0
  ClearErrors
  ${GetOptions} $R0 "/C" $0
  ${IfNot} ${Errors}
    StrCpy $ChocolateyMode "1"
  ${Else}
    StrCpy $ChocolateyMode "0"
  ${EndIf}
FunctionEnd

/**
 * GUI initialization, called before window is shown
 */
Function onGUIInit
  ; Check for docker-compose
  !ifdef DOCKER_NSH
  Call checkDocker
  !endif ; DOCKER_NSH

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
    !ifdef DOCKER_NSH
    ${If} $DockerVisible == 1
    ${AndIf} $DockerSelected == 1
      !insertmacro SelectSection ${SecDocker}
    ${EndIf}
    !endif ; DOCKER_NSH

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
  !ifdef DOCKER_NSH
  ${If} $DockerVisible != 1
    !insertmacro RemoveSection ${SecDocker}
  ${ElseIf} $DockerSelected == 1
    !insertmacro SelectSection ${SecDocker}
  ${EndIf}
  !endif ; DOCKER_NSH

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
 * Disable sudo license page if component is not selected or already accepted before
 */
Function sudoLicPre
  ReadRegDWORD $R0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:SudoLicenseAccepted"
  ${If} $R0 = 1
    Abort
  ${EndIf}
FunctionEnd

/**
 * Set sudo license accepted flag
 */
Function sudoLicLeave
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:SudoLicenseAccepted" 0x00000001
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
 * Disable WinNFSd license page if component is not selected or already accepted before
 */
Function winNFSdLicPre
  ReadRegDWORD $R0 ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:WinNFSdLicenseAccepted"
  ${If} $R0 = 1
  ${OrIfNot} ${SectionIsSelected} ${SecWinNFSd}
    Abort
  ${EndIf}
FunctionEnd

/**
 * Set WinNFSd license accepted flag
 */
Function winNFSdLicLeave
  WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:WinNFSdLicenseAccepted" 0x00000001
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
 * Check for docker-compose
 */
!ifdef DOCKER_NSH
Function checkDocker
  ${IfNot} ${Silent}
    Var /GLOBAL DockerIgnore

    ; Read setup status from registry
    ReadRegDWORD $DockerIgnore ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:DockerIgnore"

    ; Check if ignore flag is set
    ${If} $DockerIgnore != 1
      ; Check if docker-compose is executable
      ${IfNot} ${DockerComposeIsExecutable}
        ; Check if docker is supported on this system
        ${If} ${DockerDesktopIsInstallable}
          ; Show Docker Desktop section
          StrCpy $DockerVisible "1"

          ; Check if docker is installed
          ${IfNot} ${DockerDesktopIsInstalled}
            MessageBox MB_ICONQUESTION|MB_YESNOCANCEL "`${DOCKER_DESKTOP_NAME}` is not installed, but it is required for $(^Name) to function. Would you like to download and install `${DOCKER_DESKTOP_NAME}` during this setup? Cancel will not show this message again." IDYES DockerDesktopSelect IDCANCEL CheckDockerIgnore
          ${Else}
            MessageBox MB_ICONINFORMATION|MB_OK "`${DOCKER_DESKTOP_NAME}` is installed but docker-compose is not available in variable `Path` ), but they are required for $(^Name) to function. Please install them after you complete $(^Name) installation."
          ${EndIf}

          Goto CheckDockerEnd

        DockerDesktopSelect:
          StrCpy $DockerSelected "1"
        ${Else}
          MessageBox MB_ICONQUESTION|MB_YESNOCANCEL "`${DOCKER_TOOLBOX_NAME}` is not installed, but it is required for $(^Name) to function. Would you like to go to the download page of `${DOCKER_TOOLBOX_NAME}`? Cancel will not show this message again." IDYES DockerToolboxDownload IDCANCEL CheckDockerIgnore
          Goto CheckDockerEnd

        DockerToolboxDownload:
          ExecShell "open" "${DOCKER_TOOLBOX_URL}"
        ${EndIf}

        Goto CheckDockerEnd

      CheckDockerIgnore:
        WriteRegDWORD ${REG_UNINST_ROOT} "${REG_UNINST_KEY}" "NSIS:DockerIgnore" 1
      ${EndIf}
    ${EndIf}
  ${EndIf}
  CheckDockerEnd:
FunctionEnd
!endif ; DOCKER_NSH



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

  Delete "$INSTDIR\nssm.exe"

  Delete "$INSTDIR\windows_ddev_nfs_setup.sh"
  Delete "$INSTDIR\winnfsd_license.txt"
  Delete "$INSTDIR\winnfsd.exe"

  Delete "$INSTDIR\mkcert uninstall.lnk"
  Delete "$INSTDIR\mkcert install.lnk"
  Delete "$INSTDIR\mkcert_license.txt"
  Delete "$INSTDIR\mkcert.exe"

  Delete "$INSTDIR\sudo_license.txt"
  Delete "$INSTDIR\sudo.exe"

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
