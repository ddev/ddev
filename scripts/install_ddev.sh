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

# semver_compare from https://gist.github.com/Ariel-Rodriguez/9e3c2163f4644d7a389759b224bfe7f3
semver_compare() {
  local version_a version_b pr_a pr_b
  # strip word "v" and extract first subset version (x.y.z from x.y.z-foo.n)
  version_a=$(echo "${1//v/}" | awk -F'-' '{print $1}')
  version_b=$(echo "${2//v/}" | awk -F'-' '{print $1}')

  if [ "$version_a" \= "$version_b" ]
  then
    # check for pre-release
    # extract pre-release (-foo.n from x.y.z-foo.n)
    pr_a=$(echo "$1" | awk -F'-' '{print $2}')
    pr_b=$(echo "$2" | awk -F'-' '{print $2}')

    ####
    # Return 0 when A is equal to B
    [ "$pr_a" \= "$pr_b" ] && echo 0 && return 0

    ####
    # Return 1

    # Case when A is not pre-release
    if [ -z "$pr_a" ]
    then
      echo 1 && return 0
    fi

    ####
    # Case when pre-release A exists and is greater than B's pre-release

    # extract numbers -rc.x --> x
    number_a=$(echo ${pr_a//[!0-9]/})
    number_b=$(echo ${pr_b//[!0-9]/})
    [ -z "${number_a}" ] && number_a=0
    [ -z "${number_b}" ] && number_b=0

    [ "$pr_a" \> "$pr_b" ] && [ -n "$pr_b" ] && [ "$number_a" -gt "$number_b" ] && echo 1 && return 0

    ####
    # Retrun -1 when A is lower than B
    echo -1 && return 0
  fi
  arr_version_a=(${version_a//./ })
  arr_version_b=(${version_b//./ })
  cursor=0
  # Iterate arrays from left to right and find the first difference
  while [ "$([ "${arr_version_a[$cursor]}" -eq "${arr_version_b[$cursor]}" ] && [ $cursor -lt ${#arr_version_a[@]} ] && echo true)" == true ]
  do
    cursor=$((cursor+1))
  done
  [ "${arr_version_a[$cursor]}" -gt "${arr_version_b[$cursor]}" ] && echo 1 || echo -1
}

unamearch=$(uname -m)
case ${unamearch} in
  x86_64) ARCH="amd64";
  ;;
  aarch64) ARCH="arm64";
  ;;
  *) printf "${RED}Sorry, your machine architecture ${unamearch} is not currently supported.\n${RESET}" && exit 106
  ;;
esac

LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/drud/ddev/releases/latest)
# The releases are returned in the format {"id":3622206,"tag_name":"hello-1.0.0.11",...}, we have to extract the tag_name.
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')

VERSION=$LATEST_VERSION
if [ $# -ge 1 ]; then
  VERSION=$1
fi
RELEASE_BASE_URL="https://github.com/drud/ddev/releases/download/$VERSION"

rv=$(semver_compare "${VERSION}" "v1.10.0")
if [[ ${rv} -lt 0 ]]; then
  printf "${RED}Sorry, this installer does not support specifying versions of ddev prior to v1.10.0${RESET}\n"
  exit 1
fi

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

USE_ARCH=$(semver_compare "${VERSION}" "v1.16.0-alpha4")
# Versions after v1.16.0-alpha4 need the architecture in the filename
if [ "${USE_ARCH}" == 1 ]; then
  FILEBASE="${FILEBASE}.${ARCH}"
fi


if ! docker --version >/dev/null 2>&1; then
    printf "${YELLOW}Docker is required for ddev. Download and install docker at https://www.docker.com/community-edition#/download before attempting to use ddev.${RESET}\n"
fi

if ! docker-compose --version >/dev/null 2>&1; then
    printf "${YELLOW}Docker Compose is required for ddev. Download and install docker-compose at https://www.docker.com/community-edition#/download before attempting to use ddev.${RESET}\n"
fi

TARBALL="$FILEBASE.$VERSION.tar.gz"
SHAFILE="$TARBALL.sha256.txt"
NFS_INSTALLER=macos_ddev_nfs_setup.sh

curl -sSL "$RELEASE_BASE_URL/$TARBALL" -o "/tmp/$TARBALL"
curl -sSL "$RELEASE_BASE_URL/$SHAFILE" -o "/tmp/$SHAFILE"
curl -sSL "$RELEASE_BASE_URL/macos_ddev_nfs_setup.sh" -o /tmp/macos_ddev_nfs_setup.sh

cd /tmp; $SHACMD -c "$SHAFILE"
tar -xzf $TARBALL -C /tmp
chmod ugo+x /tmp/ddev /tmp/macos_ddev_nfs_setup.sh

printf "Download verified. Ready to place ddev and mkcert in your /usr/local/bin.\n"

if [ -L /usr/local/bin/ddev ] ; then
    printf "${RED}ddev already exists as a link in /usr/local/bin. Was it installed with homebrew?${RESET}\n"
    printf "${RED}Cowardly refusing to install over existing symlink${RESET}\n"
    printf "${RED}Use 'brew unlink ddev' to remove the symlink. Or use 'brew upgrade ddev' to upgrade.${RESET}\n"
    exit 101
fi
if [[ "$BINOWNER" == "$USER" ]]; then
    mv /tmp/ddev /tmp/mkcert /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/
else
    printf "${YELLOW}Running \"sudo mv /tmp/ddev /tmp/mkcert /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/\" Please enter your password if prompted.${RESET}\n"
    sudo mv /tmp/ddev /tmp/mkcert /tmp/macos_ddev_nfs_setup.sh /usr/local/bin/
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

printf "${YELLOW}Running mkcert -install, which may request your sudo password.'.${RESET}\n"
mkcert -install

