#!/bin/bash
set -e

# Download and install latest ddev release

RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
RESET='\033[0m'
OS=$(uname)
BINOWNER=$(ls -ld /usr/local/bin | awk '{print $3}')
USER=$(whoami)
SHACMD=""
FILEBASE=""
LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/drud/ddev/releases/latest)
# The releases are returned in the format {"id":3622206,"tag_name":"hello-1.0.0.11",...}, we have to extract the tag_name.
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
URL="https://github.com/drud/ddev/releases/download/$LATEST_VERSION"

if [[ "$OS" == "Darwin" ]]; then
    SHACMD="shasum -a 256"
    FILEBASE="ddev_osx"
elif [[ "$OS" == "Linux" ]]; then
    SHACMD="sha256sum"
    FILEBASE="ddev_linux"
else
    printf "${RED}Sorry, this installer does not support your platform at this time.${RESET}\n"
    exit 1
fi

if ! docker --version >/dev/null 2>&1; then
    printf "${YELLOW}Docker is required for ddev. Download and install docker at https://www.docker.com/community-edition#/download before attempting to use ddev.${RESET}\n"
fi

if ! docker-compose --version >/dev/null 2>&1; then
    printf "${YELLOW}Docker Compose is required for ddev. Download and install docker-compose at https://www.docker.com/community-edition#/download before attempting to use ddev.${RESET}\n"
fi

TARBALL="$FILEBASE.$LATEST_VERSION.tar.gz"
SHAFILE="$TARBALL.sha256.txt"

curl -sSL "$URL/$TARBALL" -o "/tmp/$TARBALL"
curl -sSL "$URL/$SHAFILE" -o "/tmp/$SHAFILE"

cd /tmp; $SHACMD -c "$SHAFILE"
tar -xzf $TARBALL -C /tmp
chmod ugo+x /tmp/ddev

printf "Download verified. Ready to place ddev in your /usr/local/bin.\n"

if [[ "$BINOWNER" == "$USER" ]]; then
    mv /tmp/ddev /usr/local/bin/
else
    printf "${YELLOW}Running \"sudo mv /tmp/ddev /usr/local/bin/\" Please enter your password if prompted.${RESET}\n"
    sudo mv /tmp/ddev /usr/local/bin/
fi

if which brew &&  [ -f `brew --prefix`/etc/bash_completion ]; then
	bash_completion_dir=$(brew --prefix)/etc/bash_completion.d
    cp /tmp/ddev_bash_completion.sh $bash_completion_dir/ddev
    printf "${GREEN}Installed ddev bash completions in $bash_completion_dir${RESET}\n"
    rm /tmp/ddev_bash_completion.sh
else
	printf "${YELLOW}Bash completion for ddev was not installed. You may manually install /tmp/ddev_bash_completions.sh in your bash_completions.d directory.${RESET}\n"
fi

rm /tmp/$TARBALL /tmp/$SHAFILE

printf "${GREEN}ddev is now installed. Run \"ddev\" to verify your installation and see usage.${RESET}\n"
