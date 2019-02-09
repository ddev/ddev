; ddev.nsi
;
; This script is based on example2.nsi. It remembers the directory,
; uninstall support and (optionally) installs start menu shortcuts.
;
; It will install ddev.exe into $PROGRAMFILES64/ddev,

;--------------------------------

!define MUI_PRODUCT "DDEV-Local"
!define MUI_VERSION "${VERSION}"
CRCCheck On

!include MUI2.nsh
!include LogicLib.nsh

; The name of the installer
Name "ddev ${MUI_VERSION}"

OutFile "../.gotmp/bin/windows_amd64/ddev_windows_installer_unsigned.${MUI_VERSION}.exe"

; The default installation directory
InstallDir $PROGRAMFILES64\ddev

; Registry key to check for directory (so if you install again, it will 
; overwrite the old one automatically)
InstallDirRegKey HKLM "Software\ddev" ""

; Request admin privileges
RequestExecutionLevel admin

;--------------------------------
;Interface Settings

  !define MUI_ABORTWARNING

;--------------------------------
;Pages

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

!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES


!define MUI_FINISHPAGE_TITLE_3LINES "Welcome to DDEV-Local"
!define MUI_FINISHPAGE_TEXT "Please review the release notes."
!define MUI_FINISHPAGE_SHOWREADME https://github.com/drud/ddev/releases
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Continue to review the release notes."
!define MUI_FINISHPAGE_LINK "github.com/drud/ddev"
!define MUI_FINISHPAGE_LINK_LOCATION "https://github.com/drud/ddev"
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

;--------------------------------
;Installer Sections

Section "ddev (github.com/drud/ddev)" SecDDEV
  SectionIn RO
  SetOutPath $INSTDIR
  
  File "../.gotmp/bin/windows_amd64/ddev.exe"

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
  SectionIn 1
  SetOutPath $INSTDIR
  File "../.gotmp/bin/windows_amd64/sudo.exe"
SectionEnd

Section "nssm (https://nssm.cc/download)" SecNSSM
  SectionIn 1
  SetOutPath $INSTDIR
  SetOverwrite off
  File "../.gotmp/bin/windows_amd64/nssm.exe"
SectionEnd

Section "WinNFSd (github.com/winnfsd/winnfsd)" SecWinNFSd
  SectionIn 1
  SetOutPath $INSTDIR
  SetOverwrite off
  File "../.gotmp/bin/windows_amd64/winnfsd.exe"
SectionEnd

Section "windows_ddev_nfs_setup.sh" SecNFSInstall
  SectionIn 1
  SetOutPath $INSTDIR
  File "../scripts/windows_ddev_nfs_setup.sh"
SectionEnd

Section "Add to PATH" SecAddToPath
  SectionIn 2
  Push $INSTDIR
  Call AddToPath
SectionEnd

Section "Start Menu Shortcuts" SecStartMenu
  CreateDirectory "$SMPROGRAMS\ddev"
  CreateShortcut "$SMPROGRAMS\ddev\Uninstall.lnk" "$INSTDIR\ddev_uninstall.exe" "" "$INSTDIR\ddev_uninstall.exe" 0
SectionEnd

;--------------------------------
;Descriptions

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

;--------------------------------
; Uninstaller

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
  Call un.RemoveFromPath

SectionEnd

; Check on startup for docker-compose. If it doesn't exist, warn the user.
Function .onInit
    nsExec::ExecToStack "docker-compose -v"
    Pop $0 # return value/error/timeout
    Pop $1
    ${If} $0 != "0"
      MessageBox MB_OK "Docker and docker-compose do not seem to be installed (or are not available in %PATH%), but they are required for ddev to function. Please install them after you complete ddev installation." /SD IDOK
    ${EndIf}
FunctionEnd


!ifndef _AddToPath_nsh
!define _AddToPath_nsh

!verbose 3
!include "WinMessages.NSH"
!verbose 4

!ifndef WriteEnvStr_RegKey
  !ifdef ALL_USERS
    !define WriteEnvStr_RegKey \
       'HKLM "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"'
  !else
    !define WriteEnvStr_RegKey 'HKCU "Environment"'
  !endif
!endif

; AddToPath - Adds the given dir to the search path.
;        Input - head of the stack
;        Note - Win9x systems requires reboot

Function AddToPath
  Exch $0
  Push $1
  Push $2
  Push $3

  # don't add if the path doesn't exist
  IfFileExists "$0\*.*" "" AddToPath_done

  ReadEnvStr $1 PATH
  Push "$1;"
  Push "$0;"
  Call StrStr
  Pop $2
  StrCmp $2 "" "" AddToPath_done
  Push "$1;"
  Push "$0\;"
  Call StrStr
  Pop $2
  StrCmp $2 "" "" AddToPath_done
  GetFullPathName /SHORT $3 $0
  Push "$1;"
  Push "$3;"
  Call StrStr
  Pop $2
  StrCmp $2 "" "" AddToPath_done
  Push "$1;"
  Push "$3\;"
  Call StrStr
  Pop $2
  StrCmp $2 "" "" AddToPath_done

  Call IsNT
  Pop $1
  StrCmp $1 1 AddToPath_NT
    ; Not on NT
    StrCpy $1 $WINDIR 2
    FileOpen $1 "$1\autoexec.bat" a
    FileSeek $1 -1 END
    FileReadByte $1 $2
    IntCmp $2 26 0 +2 +2 # DOS EOF
      FileSeek $1 -1 END # write over EOF
    FileWrite $1 "$\r$\nSET PATH=%PATH%;$3$\r$\n"
    FileClose $1
    SetRebootFlag true
    Goto AddToPath_done

  AddToPath_NT:
    ReadRegStr $1 ${WriteEnvStr_RegKey} "PATH"
    StrCmp $1 "" AddToPath_NTdoIt
      Push $1
      Call Trim
      Pop $1
      StrCpy $0 "$1;$0"
    AddToPath_NTdoIt:
      WriteRegExpandStr ${WriteEnvStr_RegKey} "PATH" $0
      SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  AddToPath_done:
    Pop $3
    Pop $2
    Pop $1
    Pop $0
FunctionEnd

; RemoveFromPath - Remove a given dir from the path
;     Input: head of the stack

Function un.RemoveFromPath
  Exch $0
  Push $1
  Push $2
  Push $3
  Push $4
  Push $5
  Push $6

  IntFmt $6 "%c" 26 # DOS EOF

  Call un.IsNT
  Pop $1
  StrCmp $1 1 unRemoveFromPath_NT
    ; Not on NT
    StrCpy $1 $WINDIR 2
    FileOpen $1 "$1\autoexec.bat" r
    GetTempFileName $4
    FileOpen $2 $4 w
    GetFullPathName /SHORT $0 $0
    StrCpy $0 "SET PATH=%PATH%;$0"
    Goto unRemoveFromPath_dosLoop

    unRemoveFromPath_dosLoop:
      FileRead $1 $3
      StrCpy $5 $3 1 -1 # read last char
      StrCmp $5 $6 0 +2 # if DOS EOF
        StrCpy $3 $3 -1 # remove DOS EOF so we can compare
      StrCmp $3 "$0$\r$\n" unRemoveFromPath_dosLoopRemoveLine
      StrCmp $3 "$0$\n" unRemoveFromPath_dosLoopRemoveLine
      StrCmp $3 "$0" unRemoveFromPath_dosLoopRemoveLine
      StrCmp $3 "" unRemoveFromPath_dosLoopEnd
      FileWrite $2 $3
      Goto unRemoveFromPath_dosLoop
      unRemoveFromPath_dosLoopRemoveLine:
        SetRebootFlag true
        Goto unRemoveFromPath_dosLoop

    unRemoveFromPath_dosLoopEnd:
      FileClose $2
      FileClose $1
      StrCpy $1 $WINDIR 2
      Delete "$1\autoexec.bat"
      CopyFiles /SILENT $4 "$1\autoexec.bat"
      Delete $4
      Goto unRemoveFromPath_done

  unRemoveFromPath_NT:
    ReadRegStr $1 ${WriteEnvStr_RegKey} "PATH"
    StrCpy $5 $1 1 -1 # copy last char
    StrCmp $5 ";" +2 # if last char != ;
      StrCpy $1 "$1;" # append ;
    Push $1
    Push "$0;"
    Call un.StrStr ; Find `$0;` in $1
    Pop $2 ; pos of our dir
    StrCmp $2 "" unRemoveFromPath_done
      ; else, it is in path
      # $0 - path to add
      # $1 - path var
      StrLen $3 "$0;"
      StrLen $4 $2
      StrCpy $5 $1 -$4 # $5 is now the part before the path to remove
      StrCpy $6 $2 "" $3 # $6 is now the part after the path to remove
      StrCpy $3 $5$6

      StrCpy $5 $3 1 -1 # copy last char
      StrCmp $5 ";" 0 +2 # if last char == ;
        StrCpy $3 $3 -1 # remove last char

      WriteRegExpandStr ${WriteEnvStr_RegKey} "PATH" $3
      SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  unRemoveFromPath_done:
    Pop $6
    Pop $5
    Pop $4
    Pop $3
    Pop $2
    Pop $1
    Pop $0
FunctionEnd



!ifndef IsNT_KiCHiK
!define IsNT_KiCHiK

###########################################
#            Utility Functions            #
###########################################

; IsNT
; no input
; output, top of the stack = 1 if NT or 0 if not
;
; Usage:
;   Call IsNT
;   Pop $R0
;  ($R0 at this point is 1 or 0)

!macro IsNT un
Function ${un}IsNT
  Push $0
  ReadRegStr $0 HKLM "SOFTWARE\Microsoft\Windows NT\CurrentVersion" CurrentVersion
  StrCmp $0 "" 0 IsNT_yes
  ; we are not NT.
  Pop $0
  Push 0
  Return

  IsNT_yes:
    ; NT!!!
    Pop $0
    Push 1
FunctionEnd
!macroend
!insertmacro IsNT ""
!insertmacro IsNT "un."

!endif ; IsNT_KiCHiK

; StrStr
; input, top of stack = string to search for
;        top of stack-1 = string to search in
; output, top of stack (replaces with the portion of the string remaining)
; modifies no other variables.
;
; Usage:
;   Push "this is a long ass string"
;   Push "ass"
;   Call StrStr
;   Pop $R0
;  ($R0 at this point is "ass string")

!macro StrStr un
Function ${un}StrStr
Exch $R1 ; st=haystack,old$R1, $R1=needle
  Exch    ; st=old$R1,haystack
  Exch $R2 ; st=old$R1,old$R2, $R2=haystack
  Push $R3
  Push $R4
  Push $R5
  StrLen $R3 $R1
  StrCpy $R4 0
  ; $R1=needle
  ; $R2=haystack
  ; $R3=len(needle)
  ; $R4=cnt
  ; $R5=tmp
  loop:
    StrCpy $R5 $R2 $R3 $R4
    StrCmp $R5 $R1 done
    StrCmp $R5 "" done
    IntOp $R4 $R4 + 1
    Goto loop
done:
  StrCpy $R1 $R2 "" $R4
  Pop $R5
  Pop $R4
  Pop $R3
  Pop $R2
  Exch $R1
FunctionEnd
!macroend
!insertmacro StrStr ""
!insertmacro StrStr "un."

Function Trim ; Added by Pelaca
	Exch $R1
	Push $R2
Loop:
	StrCpy $R2 "$R1" 1 -1
	StrCmp "$R2" " " RTrim
	StrCmp "$R2" "$\n" RTrim
	StrCmp "$R2" "$\r" RTrim
	StrCmp "$R2" ";" RTrim
	GoTo Done
RTrim:
	StrCpy $R1 "$R1" -1
	Goto Loop
Done:
	Pop $R2
	Exch $R1
FunctionEnd

!endif ; _AddToPath_nsh
