#!/bin/bash

#ddev-generated
# Please run this script with "bash test_ddev.sh"
# You can copy and paste it (make a file named test_ddev.sh)
# Or use curl or wget to download the *raw* version.
# If you're on Windows (not WSL2) please run it in a git-bash window
# When you are reporting an issue, please include the full output of this script.
# If you have NFS enabled globally, please temporarily disable it with
# `ddev config global --performance-mode-reset`

PROJECT_NAME=tryddevproject-${RANDOM}

function cleanup {
  printf "\nPlease run cleanup after debugging with 'ddev debug testcleanup'\n"
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

if ! ddev describe >/dev/null 2>&1; then printf "Please try running this in an existing DDEV project directory, preferably the problem project.\nIt doesn't work in other directories.\n"; exit 2; fi

echo "======= Existing project config ========="
ddev debug configyaml | grep -v web_environment

PROJECT_DIR=../${PROJECT_NAME}
echo "======= Creating dummy project named  ${PROJECT_NAME} in ${PROJECT_DIR} ========="

set -eu
mkdir -p "${PROJECT_DIR}/web" || (echo "Unable to create test project at ${PROJECT_DIR}/web, please check ownership and permissions" && exit 2 )
cd "${PROJECT_DIR}" || exit 3
ddev config --project-type=php --docroot=web >/dev/null 2>&1  || (printf "\n\nPlease run 'ddev debug test' in the root of the existing project where you're having trouble.\n\n" && exit 4)
set +eu

echo -n "OS Information: " && uname -a
command -v sw_vers >/dev/null && sw_vers

echo "User information: $(id -a)"
echo "DDEV version: $(ddev version)"
echo "PROXY settings: HTTP_PROXY='${HTTP_PROXY:-}' HTTPS_PROXY='${HTTPS_PROXY:-}' http_proxy='${http_proxy:-}' NO_PROXY='${NO_PROXY:-}'"

echo "======= DDEV global info ========="
ddev config global | (grep -v "^web-environment" || true)
printf "\n======= DOCKER info =========\n"
echo -n "docker location: " && ls -l "$(which docker)"
if [ ${OSTYPE%-*} != "linux" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi
echo "docker version: " && docker version
echo "DOCKER_DEFAULT_PLATFORM=${DOCKER_DEFAULT_PLATFORM:-notset}"
echo "======= Mutagen Info ========="
if [ -f ~/.ddev/bin/mutagen ]; then
  echo "Mutagen is installed in ddev, version=$(~/.ddev/bin/mutagen version)"
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen sync list -l
fi

echo "======= Docker Info ========="

if ddev debug dockercheck -h| grep dockercheck >/dev/null; then
  ddev debug dockercheck 2>/dev/null
fi

echo "Docker disk space:" && docker run --rm busybox:stable df -h // && echo
ddev poweroff
echo "Existing docker containers: " && docker ps -a

cat <<END >web/index.php
<?php
  mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
  \$mysqli = new mysqli('db', 'db', 'db', 'db');
  printf("Success accessing database... %s\n", \$mysqli->host_info);
  print "ddev is working. You will want to delete this project with 'ddev delete -Oy ${PROJECT_NAME}'\n";
END
trap cleanup EXIT

ddev start -y || ( \
  set +x && \
  ddev list && \
  ddev describe && \
  printf "============= ddev-${PROJECT_NAME}-web healtcheck run =========\n" && \
  docker exec ddev-${PROJECT_NAME}-web bash -x 'rm -f /tmp/healthy && /healthcheck.sh' && \
  printf "========= web container healthcheck ======\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-${PROJECT_NAME}-web && \
  printf "============= ddev-router healthcheck =========\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-router && \
  printf "============= Global ddev homeadditions =========\n" && \
  ls -lR ~/.ddev/homeadditions/
  printf "============= ddev logs =========\n" && \
  ddev logs | tail -20l && \
  printf "============= contents of /mnt/ddev_config  =========\n" && \
  docker exec -it ddev-d9-db ls -l /mnt/ddev_config && \
  printf "Start failed. Please provide this output in a new gist at gist.github.com\n" && \
  exit 1 )

echo "======== Curl of site from inside container:"
ddev exec curl --fail -I http://127.0.0.1

echo "======== curl -I of http://${PROJECT_NAME}.ddev.site from outside:"
curl --fail -I http://${PROJECT_NAME}.ddev.site
if [ $? -ne 0 ]; then
  set +x
  echo "Unable to curl the requested project Please provide this output in a new gist at gist.github.com."
  exit 1
fi
echo "======== full curl of http://${PROJECT_NAME}.ddev.site from outside:"
curl http://${PROJECT_NAME}.ddev.site

echo "======== Project ownership on host:"
ls -ld ${PROJECT_DIR}
echo "======== Project ownership in container:"
ddev exec ls -ld /var/www/html
echo "======== In-container filesystem:"
ddev exec df -T /var/www/html
echo "======== curl again of ${PROJECT_NAME} from host:"
curl --fail http://${PROJECT_NAME}.ddev.site
if [ $? -ne 0 ]; then
  set +x
  echo "Unable to curl the requested project Please provide this output in a new gist at gist.github.com."
  exit 1
fi

echo "Thanks for running the diagnostic. It was successful."
echo "Please provide the output of this script in a new gist at gist.github.com"
echo "Running ddev launch in 5 seconds" && sleep 5
ddev launch
