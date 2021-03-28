#!/bin/bash

# Please run this script with "bash testddev.sh"
# You can copy and paste it (make a file named testddev.sh)
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
ddev version
echo -n "docker location: " && ls -l "$(which docker)"
if [ ${OSTYPE%-*} != "linux" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi
ddev poweroff
echo "Existing docker containers: " && docker ps -a
docker run -it --rm busybox sh -c "echo 'docker can run busybox image'"
mkdir -p ~/tmp/${PROJECT_NAME} && cd ~/tmp/${PROJECT_NAME}
printf "<?php\nprint 'ddev is working. You will want to delete this project with \"ddev delete -Oy ${PROJECT_NAME}\"';\n" >index.php
ddev config --project-type=php
trap cleanup EXIT

echo y | ddev start || ( \
  set +x && \
  ddev list && \
  printf "========= web container healthcheck ======\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-${PROJECT_NAME}-web && \
  printf "============= ddev-router healthcheck =========\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-router && \
  ddev logs >logs.txt && \
  printf "Start failed. Please provide this output and the contents of ~/tmp/${PROJECT_NAME}/logs.txt in a new gist at gist.github.com\n" && \
  exit 1 )
echo "Thanks for running the diagnostic. It was successful."
echo "Please provide the output of this script in a new gist at gist.github.com"
echo "Running ddev launch in 5 seconds" && sleep 5
ddev launch
