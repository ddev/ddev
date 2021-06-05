Name "DDEV Lib Test"
OutFile "..\.gotmp\bin\windows_amd64\ddev_lib_test.exe"
ShowInstDetails show
RequestExecutionLevel user

#!define DDEV_NO_PLUGINS
#!define DOCKER_NO_PLUGINS

!addincludedir include
!include ddev.nsh
!include docker.nsh

Page components "" ""
Page instfiles

Section "Run tests"

  ClearErrors

  SetRegView 64
  ReadRegStr $0 HKLM `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\ddev.exe` `Path`
  SetRegView Default

  ${If} ${DdevIsInstalled} "$0"
    DetailPrint "DDEV is installed"
  ${Else}
    DetailPrint "$0\ddev.exe not found"
  ${EndIf}

  ${If} ${DdevIsExecutable} "$0"
    Pop $R0 ; Output
    DetailPrint "DDEV is executable:"
    DetailPrint " $R0"
  ${Else}
    DetailPrint "DDEV is not executable"
  ${EndIf}

  ${If} ${DdevPowerOff} "$0"
    Pop $R0 ; Output
    DetailPrint "DDEV projects are powered off now:"
    DetailPrint " $R0"
  ${Else}
    Pop $R0 ; Output
    DetailPrint "DDEV power off failed:"
    DetailPrint " $R0"
  ${EndIf}
  
  ${DdevDoPowerOff} "$0"
  Pop $R0 ; Return
  Pop $R1 ; Output
  DetailPrint "DDEV power off result: $R0"
  DetailPrint "DDEV power off output:"
  DetailPrint " $R1"


  ${If} ${DockerDesktopIsInstallable}
    DetailPrint "Docker Desktop is installable"
  ${Else}
    DetailPrint "Docker Desktop is not installable"
  ${EndIf}

  ${If} ${DockerDesktopIsInstalled}
    DetailPrint "Docker Desktop is installed"
  ${Else}
    DetailPrint "Docker Desktop is not installed"
  ${EndIf}

  ${If} ${DockerDesktopIsExecutable}
    Pop $R0 ; Output
    DetailPrint "docker.exe is executable:"
    DetailPrint " $R0"
  ${Else}
    DetailPrint "docker.exe is not executable"
  ${EndIf}

  ${If} ${DockerComposeIsExecutable}
    Pop $R0 ; Output
    DetailPrint "docker-compose.exe is executable"
    DetailPrint " $R0"
  ${Else}
    DetailPrint "docker-compose.exe is not executable"
  ${EndIf}

SectionEnd
