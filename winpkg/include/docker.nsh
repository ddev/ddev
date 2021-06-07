/**
 * docker.nsh - LogicLib extensions for handling docker checks
 *
 * The following "expressions" are available:
 *
 * DockerDesktopIsInstallable checks if docker is installable
 *
 *   ${If} ${DockerDesktopIsInstallable}
 *     DetailPrint "DDEV exists"
 *   ${Else}
 *     DetailPrint "$PROGRAMFILES64\DDEV\ddev.exe not found"
 *   ${EndIf}
 *
 *
 * DockerDesktopIsInstalled checks if docker is installed
 *
 *   ${If} ${DockerDesktopIsInstalled} "$PROGRAMFILES64\DDEV"
 *     DetailPrint "DDEV exists"
 *   ${Else}
 *     DetailPrint "$PROGRAMFILES64\DDEV\ddev.exe not found"
 *   ${EndIf}
 *
 *
 * DockerDesktopIsExecutable checks if docker can be executed
 *
 *   ${If} ${DockerDesktopIsExecutable}
 *     DetailPrint "DDEV is accessible"
 *   ${Else}
 *     Pop $R0 ; Output
 *     DetailPrint "DDEV is not accessible: $R0"
 *   ${EndIf}
 *
 *
 * DockerComposeIsExecutable checks if docker-compose can be executed
 *
 *   ${If} ${DockerComposeIsExecutable}
 *     DetailPrint "DDEV is accessible"
 *   ${Else}
 *     Pop $R0 ; Output
 *     DetailPrint "DDEV is not accessible: $R0"
 *   ${EndIf}
 *
 *
 * DOCKER_NO_PLUGINS disables the usage of external plugins and relays on built
 * in functions only but on the other hand disable some functionality like
 * accessing the output of the command execution.
 *
 *   !define DOCKER_NO_PLUGINS
 *   !include docker.nsh
 *
 */

!verbose push
!verbose 3
!ifndef DOCKER_VERBOSITY
  !define DOCKER_VERBOSITY 3
!endif
!define _DOCKER_VERBOSITY ${DOCKER_VERBOSITY}
!undef DOCKER_VERBOSITY
!verbose ${_DOCKER_VERBOSITY}

!ifndef DOCKER_NSH
  !define DOCKER_NSH

  ; Optional plugins
  !ifndef DOCKER_NO_PLUGINS
    ReserveFile /plugin nsExec.dll
  !endif

  ; Includes
  !include LogicLib.nsh
  !include WinVer.nsh

  ; Global constants
  !define DOCKER_DESKTOP_NAME `Docker Desktop`
  !define DOCKER_DESKTOP_URL `https://download.docker.com/win/stable/Docker%20Desktop%20Installer.exe`
  !define DOCKER_DESKTOP_SETUP `Docker Desktop Installer.exe`

  !define DOCKER_TOOLBOX_NAME `Docker Toolbox`
  !define DOCKER_TOOLBOX_URL `https://github.com/docker/toolbox/releases/latest`

  ; Macros
  !macro _DockerDesktopIsInstallable _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    !insertmacro _WinVer_BuildNumCheck U< 15063 DockerDesktopIsInstallableDone 0
    ReadRegStr $_LOGICLIB_TEMP HKLM `SOFTWARE\Microsoft\Windows NT\CurrentVersion` `EditionID`
    StrCmp $_LOGICLIB_TEMP `Home` DockerDesktopIsInstallableDone
    StrCmp $_LOGICLIB_TEMP `HomeEval` DockerDesktopIsInstallableDone
    StrCmp $_LOGICLIB_TEMP `Core` DockerDesktopIsInstallableDone
    StrCmp $_LOGICLIB_TEMP `CoreN` DockerDesktopIsInstallableDone
    StrCmp $_LOGICLIB_TEMP `CoreSingleLanguage` DockerDesktopIsInstallableDone
    StrCmp $_LOGICLIB_TEMP `CoreCountrySpecific` DockerDesktopIsInstallableDone
    StrCpy $_LOGICLIB_TEMP `Installable`
  DockerDesktopIsInstallableDone:
    !insertmacro _== $_LOGICLIB_TEMP `Installable` `${_t}` `${_f}`
  !macroend
  !define DockerDesktopIsInstallable `"" DockerDesktopIsInstallable ""`

  !macro _DockerDesktopIsInstalled _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    StrCpy $_LOGICLIB_TEMP ``
    !define DOCKER_REG_UNINST_KEY `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`
    SetRegView 64
    ReadRegStr $_LOGICLIB_TEMP HKLM `${DOCKER_REG_UNINST_KEY}\Docker Desktop` `UninstallString`
    StrCmp $_LOGICLIB_TEMP `` 0 DockerDesktopIsInstalled
    ReadRegStr $_LOGICLIB_TEMP HKLM `${DOCKER_REG_UNINST_KEY}\Docker for Windows` `UninstallString`
    StrCmp $_LOGICLIB_TEMP `` 0 DockerDesktopIsInstalled
    ReadRegStr $_LOGICLIB_TEMP HKLM `${DOCKER_REG_UNINST_KEY}\{C7F6BAA1-B432-4386-A4F0-395B8098C8D9}` `UninstallString`
    StrCmp $_LOGICLIB_TEMP `` 0 DockerDesktopIsInstalled
    Goto DockerDesktopIsInstalledDone
  DockerDesktopIsInstalled:
    StrCpy $_LOGICLIB_TEMP `Installed`
  DockerDesktopIsInstalledDone:
    SetRegView lastused
    !insertmacro _== $_LOGICLIB_TEMP `Installed` `${_t}` `${_f}`
  !macroend
  !define DockerDesktopIsInstalled `"" DockerDesktopIsInstalled ""`

  !macro _DockerDesktopIsExecutable _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    !ifdef DOCKER_NO_PLUGINS
      ExecWait `"docker.exe" -v` $_LOGICLIB_TEMP
    !else
      nsExec::ExecToStack `"docker.exe" -v`
      Pop $_LOGICLIB_TEMP ; Return, the Output remains on the stack
    !endif
    !insertmacro _== $_LOGICLIB_TEMP `0` `${_t}` `${_f}`
  !macroend
  !define DockerDesktopIsExecutable `"" DockerDesktopIsExecutable ""`

  !macro _DockerComposeIsExecutable _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    !ifdef DOCKER_NO_PLUGINS
      ExecWait `"docker-compose.exe" -v` $_LOGICLIB_TEMP
    !else
      nsExec::ExecToStack `"docker-compose.exe" -v`
      Pop $_LOGICLIB_TEMP ; Return, the Output remains on the stack
    !endif
    !insertmacro _== $_LOGICLIB_TEMP `0` `${_t}` `${_f}`
  !macroend
  !define DockerComposeIsExecutable `"" DockerComposeIsExecutable ""`

!endif ; DOCKER_NSH

!verbose 3
!define DOCKER_VERBOSITY ${_DOCKER_VERBOSITY}
!undef _DOCKER_VERBOSITY
!verbose pop
