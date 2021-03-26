#!/bin/bash

MACOS_INFO_PATH=/Applications/Docker.app/Contents/Info.plist

if [ "${OSTYPE%%[0-9]*}" = "darwin" ] && ! command -v xq >/dev/null; then
  printf "Please install xq, brew install python-yq, to parse macOS Info.plist"
  exit
fi

if command -v powershell >/dev/null; then
  printf "Docker Desktop for Windows "
  powershell.exe -command '[System.Diagnostics.FileVersionInfo]::GetVersionInfo("C:\Program Files\Docker\Docker\Docker Desktop.exe").FileVersion'
elif [ -x /usr/libexec/PlistBuddy ] ; then
  version=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" ${MACOS_INFO_PATH})
  build=$(/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" ${MACOS_INFO_PATH})
  printf "Docker Desktop for Mac %s build %s" ${version} ${build}
else
  printf "Unknown Docker Desktop version"
fi
