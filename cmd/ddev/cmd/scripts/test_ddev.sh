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


if [[ ${PWD} != ${HOME}* ]]; then
  printf "\n\nWARNING: Project should usually be in a subdirectory of the user's home directory.\nInstead it's in ${PWD}\n\n"
fi

header "Output file will be in $1"
if ! ddev describe >/dev/null 2>&1; then printf "Please try running this in an existing DDEV project directory, preferably the problem project.\nIt doesn't work in other directories.\n"; exit 2; fi


header "Existing project config"

echo "ddev installation alternate locations:"
which -a ddev
echo

ddev debug configyaml | grep -v web_environment

header "existing project customizations"
grep -r -L "#ddev-generated" .ddev/docker-compose.*.yaml .ddev/php .ddev/mutagen .ddev/apache .ddev/nginx* .ddev/*-build .ddev/mysql .ddev/postgres 2>/dev/null

header "installed DDEV add-ons"

ddev get --installed

header "mutagen situation"

echo "looking for #ddev-generated in mutagen.yml in project ${PWD}"
if [ -f .ddev/mutagen/mutagen.yml ]; then
  if grep '#ddev-generated' .ddev/mutagen/mutagen.yml; then
    echo "unmodified #ddev-generated found in .ddev/mutagen/mutagen.yml"
  else
    echo "MODIFIED .ddev/mutagen/mutagen.yml found"
  fi
else
  echo ".ddev/mutagen/mutagen.yml not found"
fi

PROJECT_DIR=../${PROJECT_NAME}
header "Creating dummy project named ${PROJECT_NAME} in ${PROJECT_DIR}"

set -eu
mkdir -p "${PROJECT_DIR}/web" || (echo "Unable to create test project at ${PROJECT_DIR}/web, please check ownership and permissions" && exit 2 )
cd "${PROJECT_DIR}" || exit 3

function cleanup {
  printf "\n\nCleanup: deleting test project ${PROJECT_NAME}\n"
  ddev delete -Oy ${PROJECT_NAME}
  printf "\nPlease remove the files from this test with 'rm -r ${PROJECT_DIR}'\n"
}

ddev config --project-type=php --docroot=web --disable-upload-dirs-warning || (printf "\n\nPlease run 'ddev debug test' in the root of the existing project where you're having trouble.\n\n" && exit 4)

printf "RUN apt-get update\nRUN curl -I https://www.google.com\n" > .ddev/web-build/Dockerfile.test

set +eu

trap cleanup SIGINT SIGTERM SIGQUIT EXIT

header "OS Information"
uname -a
command -v sw_vers >/dev/null && sw_vers

header "User information"
id -a

header "DDEV version"
ddev version
docker_platform=$(ddev version -j | docker run -i --rm ddev/ddev-utilities jq -r  '.raw."docker-platform"' 2>/dev/null)

header "proxy settings"
echo "
 HTTP_PROXY='${HTTP_PROXY:-}'
 HTTPS_PROXY='${HTTPS_PROXY:-}'
 http_proxy='${http_proxy:-}'
 NO_PROXY='${NO_PROXY:-}'
 "

header "DDEV global info"
ddev config global | (grep -v "^web-environment" || true)

header "DOCKER provider info"
docker_client="$(which docker)"
printf "docker client location: $(ls -l "${docker_client}")\n\n"

echo "docker client alternate locations:"
which -a docker
echo

printf "Docker provider: ${docker_platform}\n"
if [ "${WSL_DISTRO_NAME}" = "" ] && [ "${OSTYPE%-*}" = "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  printf "ERROR: Using Docker Desktop on Linux is not supported.\n"
fi

if [ ${OSTYPE%-*} != "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi
echo "docker version: " && docker version
printf "\nDOCKER_DEFAULT_PLATFORM=${DOCKER_DEFAULT_PLATFORM:-notset}\n"

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
header "Existing docker containers"
docker ps -a

if command -v mkcert >/dev/null; then
  header "mkcert information"
  which -a mkcert
  mkcert -CAROOT
  ls -l "$(mkcert -CAROOT)"
fi

if command -v ping >/dev/null; then
  header "ping attempt on ddev.site"
  ping -c 1 dkdkd.ddev.site
fi

cat <<END >web/index.php
<?php
  mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
  \$mysqli = new mysqli('db', 'db', 'db', 'db');
  printf("Success accessing database... %s<br />\n", \$mysqli->host_info);
  print "ddev is working.<br />\n";
  printf("The output file for Discord or issue queue is in\n<b>%s</b><br />\nfile://%s<br />\n", "$1", "$1", "$1");
END

header "ddev debug refresh"
ddev debug refresh

header "Project startup"
DDEV_DEBUG=true ddev start -y || ( \
  set +x && \
  ddev list && \
  ddev describe && \
  printf "============= ddev-${PROJECT_NAME}-web healthcheck run =========\n" && \
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

host_http_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r  '.raw.services.web.host_http_url' 2>/dev/null)
http_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r  '.raw.httpURLs[0]' 2>/dev/null)
https_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r  '.raw.httpsURLs[0]' 2>/dev/null)

header "Curl of site from inside container"
ddev exec curl --fail -I http://127.0.0.1

header "curl -I of ${host_http_url} (web container http docker bind port) from outside"
curl --fail -I ${host_http_url}

header "curl -I of ${http_url} (router http URL) from outside"
curl --fail -I "${http_url}"

header "Full curl of ${http_url} (router http URL) from outside"
curl "${http_url}"

header "Full curl of ${https_url} (router https URL) from outside"
curl "${https_url}"

header "Curl google.com to check internet access and VPN"
curl -I https://www.google.com

header "host.docker.internal status"
ddev exec ping -c 1 host.docker.internal

header "Project ownership on host"
ls -ld "${PROJECT_DIR}"
header "Project ownership in container"
ddev exec ls -ld //var/www/html
header "In-container filesystem"
ddev exec df -T //var/www/html

header 'Thanks for running the diagnostic!'
echo "Running ddev launch in 3 seconds" && sleep 3
echo "Running ddev launch"
ddev launch
echo "Waiting for ddev launch to complete before deleting project"
sleep 10
