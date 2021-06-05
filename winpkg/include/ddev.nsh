/**
 * ddev.nsh - LogicLib extensions for handling DDEV checks and actions
 *
 * The following "expressions" are available:
 *
 * DdevIsInstalled checks if ddev.exe is available in the provided directory
 *
 *   ${If} ${DdevIsInstalled} "$PROGRAMFILES64\DDEV"
 *     DetailPrint "DDEV exists"
 *   ${Else}
 *     DetailPrint "$PROGRAMFILES64\DDEV\ddev.exe not found"
 *   ${EndIf}
 *
 *
 * DdevIsExecutable checks if ddev.exe can be executed in the provided directory
 *
 *   ${If} ${DdevIsExecutable} "$PROGRAMFILES64\DDEV"
 *     DetailPrint "DDEV is executable"
 *   ${Else}
 *     Pop $R0 ; Output
 *     DetailPrint "DDEV is not executable: $R0"
 *   ${EndIf}
 *
 *
 * DdevPowerOff executes ddev.exe poweroff in the provided directory
 *
 *   ${If} ${DdevPowerOff} "$PROGRAMFILES64\DDEV"
 *     DetailPrint "DDEV projects are powered off now"
 *   ${Else}
 *     Pop $R0 ; Output
 *     DetailPrint "DDEV power off failed: $R0"
 *   ${EndIf}
 *
 *
 * The following "statements" are available:
 *
 * DdevDoPowerOff executes ddev.exe poweroff in the provided directory
 *
 *   ${DdevDoPowerOff} "$PROGRAMFILES64\DDEV"
 *   Pop $R0 ; Output
 *   DetailPrint "DDEV power off output: $R0"
 *
 *
 * DDEV_NO_PLUGINS disables the usage of external plugins and relays on built
 * in functions only but on the other hand disable some functionality like
 * accessing the output of the command execution.
 *
 *   !define DDEV_NO_PLUGINS
 *   !include ddev.nsh
 *
 */

!verbose push
!verbose 3
!ifndef DDEV_VERBOSITY
  !define DDEV_VERBOSITY 3
!endif
!define _DDEV_VERBOSITY ${DDEV_VERBOSITY}
!undef DDEV_VERBOSITY
!verbose ${_DDEV_VERBOSITY}

!ifndef DDEV_NSH
  !define DDEV_NSH

  !ifndef DDEV_NO_PLUGINS
    ReserveFile /plugin nsExec.dll
  !endif

  !include LogicLib.nsh

  !macro _DdevIsInstalled _a _b _t _f
    !insertmacro _FileExists `${_a}` `${_b}\ddev.exe` `${_t}` `${_f}`
  !macroend
  !define DdevIsInstalled `"" DdevIsInstalled`

  !macro _DdevIsExecutable _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    !ifdef DDEV_NO_PLUGINS
      ExecWait `"${_b}\ddev.exe" version` $_LOGICLIB_TEMP
    !else
      nsExec::ExecToStack `"${_b}\ddev.exe" version`
      Pop $_LOGICLIB_TEMP ; Return, the Output remains on the stack
    !endif
    !insertmacro _== $_LOGICLIB_TEMP `0` `${_t}` `${_f}`
  !macroend
  !define DdevIsExecutable `"" DdevIsExecutable`

  !macro _DdevPowerOff _a _b _t _f
    !insertmacro _LOGICLIB_TEMP
    !insertmacro _DdevDoPowerOff `${_b}`
    Pop $_LOGICLIB_TEMP ; Return, the Output remains on the stack
    !insertmacro _== $_LOGICLIB_TEMP `0` `${_t}` `${_f}`
  !macroend
  !define DdevPowerOff `"" DdevPowerOff`

  !macro _DdevDoPowerOff _path
    !insertmacro _LOGICLIB_TEMP
    !ifdef DDEV_NO_PLUGINS
      ExecWait `"${_path}\ddev.exe" poweroff` $_LOGICLIB_TEMP
      Push $_LOGICLIB_TEMP
    !else
      nsExec::ExecToStack `"${_path}\ddev.exe" poweroff`
    !endif
  !macroend
  !define DdevDoPowerOff `!insertmacro _DdevDoPowerOff`

!endif ; DDEV_NSH

!verbose 3
!define DDEV_VERBOSITY ${_DDEV_VERBOSITY}
!undef _DDEV_VERBOSITY
!verbose pop
