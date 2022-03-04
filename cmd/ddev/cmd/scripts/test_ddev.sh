#!/bin/bash

#ddev-generated
# Please run this script with "bash test_ddev.sh"
# You can copy and paste it (make a file named test_ddev.sh)
# Or use curl or wget to download the *raw* version.
# If you're on Windows (not WSL2) please run it in a git-bash window
# When you are reporting an issue, please include the full output of this script.
# If you have NFS enabled globally, please temporarily disable it with
# `ddev config global --nfs-mount-enabled=false`

PROJECT_NAME=tryddevproject-${RANDOM}

function cleanup {
  printf "\nPlease delete this project after debugging with 'ddev delete -Oy ${PROJECT_NAME}'\n"
}

function docker_desktop_version {
  MACOS_INFO_PATH=/Applications/Docker.app/Contents/Info.plist

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
}

echo -n "OS Information: " && uname -a
echo "User information: $(id -a)"
echo "DDEV version: $(ddev version)"

echo "======= DDEV global info ========="
ddev config global | (grep -v "^web-environment" || true)
printf "\n======= DOCKER info =========\n"
echo -n "docker location: " && ls -l "$(which docker)"
if [ ${OSTYPE%-*} != "linux" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi

echo "======= Mutagen Info ========="
if [ -f ~/.ddev/bin/mutagen ]; then
  echo "Mutagen is installed in ddev, version=$(~/.ddev/bin/mutagen version)"
  ~/.ddev/bin/mutagen sync list
fi
if command -v mutagen >/dev/null ; then
  echo "mutagen additionally installed in PATH at $(command -v mutagen), version $(mutagen version)"
fi
if killall -0 mutagen 2>/dev/null; then
  echo "mutagen is running on this system:"
  ps -ef | grep mutagen
fi

echo "======= Docker Info ========="

if ddev debug dockercheck -h| grep dockercheck >/dev/null; then
  ddev debug dockercheck 2>/dev/null
fi

echo "Docker disk space:" && docker run --rm busybox:stable df -h // && echo
ddev poweroff
echo "Existing docker containers: " && docker ps -a
mkdir -p ~/tmp/${PROJECT_NAME} && cd ~/tmp/${PROJECT_NAME}
printf "<?php\nprint 'ddev is working. You will want to delete this project with \"ddev delete -Oy ${PROJECT_NAME}\"';\n" >index.php
ddev config --project-type=php
trap cleanup EXIT

ddev start -y || ( \
  set +x && \
  ddev list && \
  printf "========= web container healthcheck ======\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-${PROJECT_NAME}-web && \
  printf "============= ddev-router healthcheck =========\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-router && \
  ddev logs >logs.txt && \
  printf "Start failed. Please provide this output and the contents of ~/tmp/${PROJECT_NAME}/logs.txt in a new gist at gist.github.com\n" && \
  exit 1 )
set -x
curl --fail -I ${PROJECT_NAME}.ddev.site
if [ $? -ne 0 ]; then
  set +x
  echo "Unable to curl the requested project Please provide this output in a new gist at gist.github.com."
  exit 1
fi
set +x
echo "Thanks for running the diagnostic. It was successful."
echo "Please provide the output of this script in a new gist at gist.github.com"
echo "Running ddev launch in 5 seconds" && sleep 5
ddev launch
