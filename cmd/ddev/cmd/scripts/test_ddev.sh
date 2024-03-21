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

function header {
  printf "\n\n======== $1 ========\n"
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

header "Output file will be in $1"
if ! ddev describe >/dev/null 2>&1; then printf "Please try running this in an existing DDEV project directory, preferably the problem project.\nIt doesn't work in other directories.\n"; exit 2; fi


header "Existing project config"
if [[ ${PWD} != ${HOME}* ]]; then
  printf "\n\nWARNING: Project should be in a subdirectory of the user's home directory.\nInstead it's in ${PWD}\n\n"
fi
ddev debug configyaml | grep -v web_environment

PROJECT_DIR=../${PROJECT_NAME}
header "Creating dummy project named  ${PROJECT_NAME} in ${PROJECT_DIR}"

set -eu
mkdir -p "${PROJECT_DIR}/web" || (echo "Unable to create test project at ${PROJECT_DIR}/web, please check ownership and permissions" && exit 2 )
cd "${PROJECT_DIR}" || exit 3
ddev config --project-type=php --docroot=web >/dev/null 2>&1  || (printf "\n\nPlease run 'ddev debug test' in the root of the existing project where you're having trouble.\n\n" && exit 4)
set +eu

header "OS Information"
uname -a
command -v sw_vers >/dev/null && sw_vers

header "User information"
id -a

header "ddev version"
ddev version
docker_platform=$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r  '.raw."docker-platform"' 2>/dev/null)

header "proxy settings"
 HTTP_PROXY='${HTTP_PROXY:-}'
 HTTPS_PROXY='${HTTPS_PROXY:-}'
 http_proxy='${http_proxy:-}'
 NO_PROXY='${NO_PROXY:-}'

header "DDEV global info"
ddev config global | (grep -v "^web-environment" || true)

header "DOCKER provider info"
echo -n "docker client location: " && ls -l "$(which docker)"
printf "Docker provider: ${docker_platform}\n"
if [ "${OSTYPE%-*}" = "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  printf "ERROR: Using Docker Desktop on Linux is not supported.\n"
fi

if [ ${OSTYPE%-*} != "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi
echo "docker version: " && docker version
echo "DOCKER_DEFAULT_PLATFORM=${DOCKER_DEFAULT_PLATFORM:-notset}"

case $docker_platform in
colima)
  colima --version && colima status
  ;;
lima)
  limactl --version && limactl list
  ;;
orbstack)
  orb version
  ;;
rancher-desktop)
  ~/.rd/bin/rdctl version
  ;;
esac

if ddev debug dockercheck -h| grep dockercheck >/dev/null; then
  ddev debug dockercheck 2>/dev/null
fi

printf "Docker disk space:" && docker run --rm busybox:stable df -h //

ddev poweroff
echo "Existing docker containers: " && docker ps -a

cat <<END >web/index.php
<?php
  mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
  \$mysqli = new mysqli('db', 'db', 'db', 'db');
  printf("Success accessing database... %s\n", \$mysqli->host_info);
  print "ddev is working. You will want to delete this project with 'ddev debug testcleanup'\n";
  printf("The output file for Discord or issue queue is in\n<b>%s</b><br />\n%s<br />\n", "$1", "$1", "$1");
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
  printf "Start failed.\n" && \
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
