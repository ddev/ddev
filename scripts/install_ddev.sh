#!/bin/bash
set -o errexit
set -o pipefail
set -o nounset

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
    FILEBASE="ddev_macos"
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
NFS_INSTALLER=macos_ddev_nfs_setup.sh

curl -sSL "$URL/$TARBALL" -o "/tmp/$TARBALL"
curl -sSL "$URL/$SHAFILE" -o "/tmp/$SHAFILE"
curl -sSL "$URL/macos_ddev_nfs_setup.sh" -o /tmp/macos_ddev_nfs_setup.sh

cd /tmp; $SHACMD -c "$SHAFILE"
tar -xzf $TARBALL -C /tmp
chmod ugo+x /tmp/ddev /tmp/macos_ddev_nfs_setup.sh

printf "Download verified. Ready to place ddev in your /usr/local/bin.\n"

if [ -L /usr/local/bin/ddev ] ; then
    printf "${RED}ddev already exists as a link in /usr/local/bin. Was it installed with homebrew?${RESET}\n"
    printf "${RED}Cowardly refusing to install over existing symlink${RESET}\n"
    printf "${RED}Use 'brew unlink ddev' to remove the symlink. Or use 'brew upgrade ddev' to upgrade.${RESET}\n"
    exit 101
fi
if [[ "$BINOWNER" == "$USER" ]]; then
    mv /tmp/ddev /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/
else
    printf "${YELLOW}Running \"sudo mv /tmp/ddev /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/\" Please enter your password if prompted.${RESET}\n"
    sudo mv /tmp/ddev /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/
fi

if command -v brew >/dev/null ; then
    if [ -d "$(brew --prefix)/etc/bash_completion.d" ]; then
        bash_completion_dir=$(brew --prefix)/etc/bash_completion.d
        cp /tmp/ddev_bash_completion.sh $bash_completion_dir/ddev
        printf "${GREEN}Installed ddev bash completions in $bash_completion_dir${RESET}\n"
        rm /tmp/ddev_bash_completion.sh
    else
        printf "${YELLOW}Bash completion for ddev was not installed. You may manually install /tmp/ddev_bash_completion.sh in your bash_completion.d directory.${RESET}\n"
    fi

    if  [ -d "$(brew --prefix)/share/zsh-completions" ] && [ -f /tmp/ddev_zsh_completion.sh ]; then
        zsh_completion_dir=$(brew --prefix)/share/zsh-completions
        cp /tmp/ddev_zsh_completion.sh $zsh_completion_dir/_ddev
        printf "${GREEN}Installed ddev zsh completions in $zsh_completion_dir${RESET}\n"
        rm /tmp/ddev_zsh_completion.sh
    else
        printf "${YELLOW}zsh completion for ddev was not installed. You may manually install /tmp/ddev_zsh_completion.sh in your zsh-completions directory.${RESET}\n"
    fi
fi

rm /tmp/$TARBALL /tmp/$SHAFILE

printf "${GREEN}ddev is now installed. Run \"ddev\" to verify your installation and see usage.${RESET}\n"
if ! command -v mkcert >/dev/null ; then
    printf "${YELLOW}Please install mkcert from https://github.com/FiloSottile/mkcert/releases and then run 'mkcert -install'.${RESET}\n"
fi
