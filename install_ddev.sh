#!/bin/bash
set -e

RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
YELLOW=$(tput setaf 3)
RESET=$(tput sgr0)
OS=$(uname)
BINOWNER=$(ls -ld /usr/local/bin | awk '{print $3}')
USER=$(whoami)
SHACMD=""
FILE=""
LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/drud/ddev/releases/latest)
# The releases are returned in the format {"id":3622206,"tag_name":"hello-1.0.0.11",...}, we have to extract the tag_name.
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
URL="https://github.com/drud/ddev/releases/download/$LATEST_VERSION"

if [[ "$OS" == "Darwin" ]]; then
    SHACMD="shasum -a 256"
    FILE="ddev_darwin_64"
elif [[ "$OS" == "Linux" ]]; then
    SHACMD="sha256sum"
    FILE="ddev_linux"
else
    echo "${RED}Sorry, your platform is not supported at this time.${RESET}"
    exit 1
fi

if ! docker --version >/dev/null 2>&1; then
    echo "${YELLOW}Docker is required for ddev. Download and install docker at https://www.docker.com/community-edition#/download before attempting to use ddev.${RESET}"
fi

curl -sSL "$URL/$FILE.tar.gz" -o "/tmp/$FILE.tar.gz"
curl -sSL "$URL/$FILE.sha256" -o "/tmp/$FILE.sha256"

cd /tmp; $SHACMD -c "$FILE.sha256"; cd -;
tar -xzf "/tmp/$FILE.tar.gz" -C /tmp
chmod ugo+x /tmp/ddev

echo "Download verified. Ready to place ddev in your /usr/local/bin."

if [[ "$BINOWNER" == "$USER" ]]; then
    mv /tmp/ddev /usr/local/bin/
else
    echo "${YELLOW}Running \"sudo mv /tmp/ddev /usr/local/bin/\" Please enter your password if prompted.${RESET}"
    sudo mv /tmp/ddev /usr/local/bin/
fi

rm "/tmp/$FILE.tar.gz"
rm "/tmp/$FILE.sha256"

echo "${GREEN}ddev is now installed. Run \"ddev\" to verify your installation and see usage.${RESET}"
