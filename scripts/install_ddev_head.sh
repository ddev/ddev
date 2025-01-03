#!/usr/bin/env bash

#ddev-generated
# Script to download and install DDEV HEAD build from GitHub Actions artifacts
# Usage: install_ddev_head.sh

set -o errexit
set -o pipefail
set -o nounset

if [ ! -d /usr/local/bin ]; then echo 'using sudo to mkdir missing /usr/local/bin' && sudo mkdir -p /usr/local/bin; fi

GITHUB_OWNER=${GITHUB_OWNER:-ddev}
TMPDIR=/tmp

RED='\033[31m'
GREEN='\033[32m'
YELLOW='\033[33m'
RESET='\033[0m'
OS=$(uname)
BINOWNER=$(ls -ld /usr/local/bin | awk '{print $3}')
USER=$(whoami)
SHACMD=""

if [[ $EUID -eq 0 ]]; then
  echo "This script must NOT be run with sudo/root. Please re-run without sudo." 1>&2
  exit 102
fi

unamearch=$(uname -m)
case ${unamearch} in
  x86_64) ARCH="amd64";
  ;;
  aarch64) ARCH="arm64";
  ;;
  arm64) ARCH="arm64"
  ;;
  *) printf "${RED}Sorry, your machine architecture ${unamearch} is not currently supported.\n${RESET}" && exit 106
  ;;
esac

if [[ "$OS" == "Darwin" ]]; then
    SHACMD="shasum -a 256"
    OS="macos"
elif [[ "$OS" == "Linux" ]]; then
    SHACMD="sha256sum"
    OS="linux"
else
    printf "${RED}Sorry, this installer does not support your platform at this time.${RESET}\n"
    exit 1
fi

if ! docker --version >/dev/null 2>&1; then
    printf "${YELLOW}A docker provider is required for ddev. Please see https://ddev.readthedocs.io/en/stable/users/install/docker-installation/.${RESET}\n"
fi

# Define artifact URLs based on OS and architecture
ARTIFACTS_BASE_URL="https://nightly.link/${GITHUB_OWNER}/ddev/workflows/main-build/main"
BINARY_ARTIFACT_URL="${ARTIFACTS_BASE_URL}/ddev-${OS}-${ARCH}.zip"

printf "${GREEN}Downloading artifacts for ${OS}_${ARCH}...${RESET}\n"

cd ${TMPDIR}

curl -fsSLO "$BINARY_ARTIFACT_URL"  || (printf "${RED}Failed downloading $BINARY_ARTIFACT_URL${RESET}\n" && exit 108)

# Extract the binary
unzip -o "ddev-${OS}-${ARCH}.zip"

printf "${GREEN}Download verified. Ready to place ddev in your /usr/local/bin.${RESET}\n"

if command -v brew >/dev/null && brew info ddev >/dev/null 2>/dev/null ; then
  echo "Attempting to unlink any homebrew-installed ddev with 'brew unlink ddev'"
  brew unlink ddev >/dev/null 2>&1 || true
fi

if [ -L /usr/local/bin/ddev ] ; then
    printf "${RED}ddev already exists as a link in /usr/local/bin/ddev. Was it installed with homebrew?${RESET}\n"
    printf "${RED}Cowardly refusing to install over existing symlink${RESET}\n"
    exit 101
fi

SUDO=""
if [[ "$BINOWNER" != "$USER" ]]; then
  SUDO=sudo
fi

if [ ! -z "${SUDO}" ]; then
    printf "${YELLOW}Running \"sudo mv ddev /usr/local/bin/\" Please enter your password if prompted.${RESET}\n"
fi

chmod +x ddev
${SUDO} mv ddev /usr/local/bin/

# Cleanup
rm -f "ddev-${OS}-${ARCH}.zip"

hash -r

printf "${GREEN}ddev HEAD build is now installed. Run \"ddev\" and \"ddev --version\" to verify your installation and see usage.${RESET}\n"
