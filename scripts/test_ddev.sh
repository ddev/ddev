#!/bin/bash

# Please run this script with "bash testddev.sh"
# You can copy and paste it (make a file named testddev.sh)
# Or use curl or wget to download the *raw* version.
# If you're on Windows (not WSL2) please run it in a git-bash window
# When you are reporting an issue, please include the full output of this script.
# If you have NFS enabled globally, please temporarily disable it with
# `ddev config global --nfs-mount-enabled=false`

PROJECT_NAME=tryddevproject-${RANDOM}
BASEDIR=$(dirname "$0")

function cleanup {
  printf "\nPlease delete this project after debugging with 'ddev delete -Oy ${PROJECT_NAME}'\n"
}
trap cleanup EXIT

uname -a
ddev version
ls -l "$(which docker)"
${BASEDIR}/docker-desktop-version.sh && echo
ddev poweroff
docker ps -a
docker run -it --rm busybox ls
mkdir -p ~/tmp/${PROJECT_NAME} && cd ~/tmp/${PROJECT_NAME}
printf "<?php\nprint 'ddev is working. You will want to delete this project with \"ddev delete -Oy ${PROJECT_NAME}\"';\n" >index.php
ddev config --project-type=php
ddev start || ( \
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
