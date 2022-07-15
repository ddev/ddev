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
cat <<END >index.php
<?php
  mysqli_report(MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT);
  \$mysqli = new mysqli('db', 'db', 'db', 'db');
  printf("Success accessing database... %s\n", \$mysqli->host_info);
  print "ddev is working. You will want to delete this project with 'ddev delete -Oy ${PROJECT_NAME}\n";
END
ddev config --project-type=php
trap cleanup EXIT

# This is a potential experiment to force failure when needed
#echo '
#services:
#  web:
#    healthcheck:
#      test: "false"
#      timeout: 15s
#      retries: 2
#      start_period: 30s
#' >.ddev/docker-compose.failhealth.yaml

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
  printf "Start failed. Please provide this output in a new gist at gist.github.com\n" && \
  exit 1 )

echo "======== Curl of site from inside container:"
ddev exec curl --fail -I http://127.0.0.1

echo "======== Curl of site from outside:"
curl --fail -I http://${PROJECT_NAME}.ddev.site
if [ $? -ne 0 ]; then
  set +x
  echo "Unable to curl the requested project Please provide this output in a new gist at gist.github.com."
  exit 1
fi

echo "======== Project ownership on host:"
ls -ld ~/tmp/${PROJECT_NAME}
echo "======== Project ownership in container:"
ddev exec ls -ld /var/www/html
echo "======== In-container filesystem:"
ddev exec df -T /var/www/html

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

echo "If you're brave and you have jq you can delete all tryddevproject instances with this one-liner:"
echo '    ddev delete -Oy $(ddev list -j |jq -r .raw[].name | grep tryddevproject)'
echo "In the future ddev debug test will also provide this option."
