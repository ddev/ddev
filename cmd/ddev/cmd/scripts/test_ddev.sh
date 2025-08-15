#!/usr/bin/env bash

#ddev-generated
# Please run this script with "bash test_ddev.sh"
# You can copy and paste it (make a file named test_ddev.sh)
# Or use curl or wget to download the *raw* version.
# If you're on Windows (not WSL2) please run it in a git-bash window
# When you are reporting an issue, please include the full output of this script.
# If you have NFS enabled globally, please temporarily disable it with
# `ddev config global --performance-mode-reset`

# Disable instrumentation inside `ddev debug test`
export DDEV_NO_INSTRUMENTATION=true

PROJECT_NAME=tryddevproject-${RANDOM}

function header {
  printf "\n\n======== %s ========\n" "$1"
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
  printf "\n\nWARNING: Project should usually be in a subdirectory of the user's home directory.\nInstead it's in %s\n\n" "${PWD}"
fi

header "Output file will be in $1"
if ! ddev describe >/dev/null 2>&1; then printf "Please try running this in an existing DDEV project directory, preferably the problem project.\nIt doesn't work in other directories.\n"; exit 2; fi


header "Existing project config"

header "ddev installation alternate locations:"
which -a ddev
echo

header "project configuration via ddev debug configyaml"
ddev debug configyaml --full-yaml --omit-keys=web_environment

header "existing project customizations"
grep -r -L "#ddev-generated" .ddev/docker-compose.*.yaml .ddev/php .ddev/mutagen .ddev/apache .ddev/nginx* .ddev/*-build .ddev/mysql .ddev/postgres .ddev/traefik/config .ddev/.env .ddev/.env.* 2>/dev/null | grep -v '\.example$' 2>/dev/null

if ls .ddev/nginx >/dev/null 2>&1 ; then
  echo "Customizations in .ddev/nginx:"
  ls -l .ddev/nginx
fi

header "installed DDEV add-ons"

ddev add-on list --installed

if [ -f /proc/version ] && grep -qEi "(microsoft|wsl)" /proc/version; then
  header "WSL2 information"

  if command -v wslinfo >/dev/null ; then
    echo "WSL version=$(wsl.exe --version | tr -d '\0')";
    echo "WSL2 networking mode=$(wslinfo --networking-mode)"
  fi
fi

if [ -f .ddev/mutagen/mutagen.yml ]; then
  header "mutagen situation"
  echo "looking for #ddev-generated in mutagen.yml in project ${PWD}"
  echo
  if grep -q '#ddev-generated' .ddev/mutagen/mutagen.yml; then
    echo "unmodified #ddev-generated found in .ddev/mutagen/mutagen.yml"
  else
    echo "MODIFIED .ddev/mutagen/mutagen.yml found"
  fi
fi

PROJECT_DIR=../${PROJECT_NAME}
header "Creating dummy project named ${PROJECT_NAME} in ${PROJECT_DIR}"

set -eu
mkdir -p "${PROJECT_DIR}/web" || (echo "Unable to create test project at ${PROJECT_DIR}/web, please check ownership and permissions" && exit 2 )
cd "${PROJECT_DIR}" || exit 3

function cleanup {
  printf "\n\nCleanup: deleting test project %s\n" "${PROJECT_NAME}"
  ddev delete -Oy ${PROJECT_NAME}
  printf "\nPlease remove the files from this test with 'rm -r %s'\n" "${PROJECT_DIR}"
}

ddev config --project-type=php --docroot=web --disable-upload-dirs-warning || (printf "\n\nPlease run 'ddev debug test' in the root of the existing project where you're having trouble.\n\n" && exit 4)

mkdir -p .ddev/web-build
printf "RUN timeout 30 apt-get update || true\nRUN curl --connect-timeout 10 --max-time 20 --fail -I https://www.google.com || true\n" > .ddev/web-build/Dockerfile.test

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
printf "docker client location: %s\n\n" "$(ls -l "${docker_client}")"

echo "docker client alternate locations:"
which -a docker
echo

printf "Docker provider: %s\n" "${docker_platform}"
if [ "${WSL_DISTRO_NAME}" = "" ] && [ "${OSTYPE%-*}" = "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  printf "Warning: Docker Desktop for Linux is not explicitly supported and does not have automated test coverage.\n"
fi

if [ "${OSTYPE%-*}" != "linux" ] && [ "$docker_platform" = "docker-desktop" ]; then
  echo -n "Docker Desktop Version: " && docker_desktop_version && echo
fi

header "docker version"
docker version

header "docker context ls"
DOCKER_HOST="" docker context ls

printf "\nDOCKER_HOST=%s\nDOCKER_DEFAULT_PLATFORM=%s\n" "${DOCKER_HOST:-notset}" "${DOCKER_DEFAULT_PLATFORM:-notset}"

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

if ddev debug dockercheck -h | grep dockercheck >/dev/null; then
  header "ddev debug dockercheck"
  ddev debug dockercheck 2>/dev/null
fi

printf "\nDocker disk space:\n" && docker run --rm ddev/ddev-utilities df -h //

header "Existing docker containers"
docker ps -a

header "docker system df"
docker system df

echo "
  Tips:
  1. Periodically check your Docker filesystem usage with 'docker system df'
  2. Use 'docker builder prune' to remove unused Docker build cache (it doesn't remove your data)
  3. To remove all containers and images (it doesn't remove your data):
    \`\`\`
    ddev poweroff
    docker rm -f \$(docker ps -aq) || true
    docker rmi -f \$(docker images -q)
    \`\`\`
    (DDEV images will be downloaded again on 'ddev start')"

if command -v mkcert >/dev/null; then
  header "mkcert information"
  which -a mkcert
  echo "CAROOT=${CAROOT:-} WSLENV=${WSLENV:-} JAVA_HOME=${JAVA_HOME:-}"
  mkcert -CAROOT
  ls -l "$(mkcert -CAROOT)"
fi

if command -v ping >/dev/null; then
  header "ping attempt on ddev.site"
  ping -c 1 dkdkd.ddev.site
fi

if command -v curl >/dev/null; then
  header "curl information"
  which -a curl
fi

cat <<END >web/index.php
<?php
  mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
  \$mysqli = new mysqli('db', 'db', 'db', 'db');
  printf("Success accessing database... %s<br />\n", \$mysqli->host_info);
  print "ddev is working.<br />\n";
  printf("The output file for Discord or issue queue is in\n<b>%s</b><br />\nfile://%s<br />\n", "$1", "$1", "$1");
END

header "ddev debug rebuild"
if ddev debug rebuild -h | grep rebuild >/dev/null; then
  ddev debug rebuild
else
  ddev debug refresh
fi

header "Project startup"
DDEV_DEBUG=true ddev start -y || ( \
  set +x && \
  ddev list && \
  ddev describe && \
  printf "============= ddev-%s-web healthcheck run =========\n" "${PROJECT_NAME}" && \
  docker exec ddev-${PROJECT_NAME}-web bash -xc 'rm -f /tmp/healthy && /healthcheck.sh' && \
  printf "========= web container healthcheck ======\n" && \
  docker inspect --format "{{json .State.Health }}" ddev-${PROJECT_NAME}-web && \
  printf "============= ddev-router healthcheck =========\n" && \
  ( docker inspect --format "{{json .State.Health }}" ddev-router || true ) && \
  printf "============= Global ddev homeadditions =========\n" && \
  ls -lR ~/.ddev/homeadditions/
  printf "============= ddev logs =========\n" && \
  ddev logs | tail -20l && \
  printf "============= contents of /mnt/ddev_config  =========\n" && \
  docker exec -it ddev-${PROJECT_NAME}-db ls -l /mnt/ddev_config && \
  printf "Start failed.\n" && \
  exit 1 )

host_http_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r '.raw.services.web.host_http_url' 2>/dev/null)
http_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r '.raw.httpURLs[0]' 2>/dev/null)
https_url=$(ddev describe -j | docker run -i --rm ddev/ddev-utilities jq -r '.raw.httpsURLs[0]' 2>/dev/null)

header "curl -I of http://127.0.0.1 from inside container"
ddev exec curl --connect-timeout 10 --max-time 20 --fail -I http://127.0.0.1

if command -v curl >/dev/null; then
  header "curl -I of ${host_http_url} (web container http docker bind port) from outside"
  curl --connect-timeout 10 --max-time 20 --fail -I "${host_http_url}"

  header "curl -I of ${http_url} (router http URL) from outside"
  curl --connect-timeout 10 --max-time 20 --fail -I "${http_url}"

  header "Full curl of ${http_url} (router http URL) from outside"
  curl --connect-timeout 10 --max-time 20 "${http_url}"

  header "Full curl of ${https_url} (router https URL) from outside"
  curl --connect-timeout 10 --max-time 20 "${https_url}"

  header "curl -I of https://www.google.com to check internet access and VPN"
  curl --connect-timeout 10 --max-time 20 -I https://www.google.com
else
  header "curl is not available on the host"
fi

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
