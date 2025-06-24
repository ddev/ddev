#!/usr/bin/env bash

#ddev-generated
# Script to download and install DDEV, https://github.com/ddev/ddev
# Usage: install_ddev.sh or install_ddev.sh <version>

set -o errexit
set -o pipefail
set -o nounset

if [ ! -d /usr/local/bin ]; then echo 'using sudo to mkdir missing /usr/local/bin' && sudo mkdir -p /usr/local/bin; fi

GITHUB_OWNER=${GITHUB_OWNER:-ddev}
ARTIFACTS="ddev ddev_hostname mkcert"

TMPDIR=/tmp

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
    [ "$pr_a" \> "$pr_b" ] && [ -n "$pr_b" ] && echo 1 && return 0

    ####
    # Return -1 when A is lower than B
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

LATEST_RELEASE=$(curl -fsSL -H 'Accept: application/json' https://github.com/${GITHUB_OWNER}/ddev/releases/latest || (printf "${RED}Failed to get find latest release${RESET}\n" >/dev/stderr && exit 107))
# The releases are returned in the format {"id":3622206,"tag_name":"hello-1.0.0.11",...}, we have to extract the tag_name.
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')

VERSION=$LATEST_VERSION
if [ $# -ge 1 ]; then
  VERSION=$1
fi
RELEASE_BASE_URL="https://github.com/${GITHUB_OWNER}/ddev/releases/download/$VERSION"

if [[ "$OS" == "Darwin" ]]; then
    SHACMD="shasum -a 256 --ignore-missing"
    FILEBASE="ddev_macos"
elif [[ "$OS" == "Linux" ]]; then
    SHACMD="sha256sum --ignore-missing"
    FILEBASE="ddev_linux"
else
    printf "${RED}Sorry, this installer does not support your platform at this time.${RESET}\n"
    exit 1
fi


FILEBASE="${FILEBASE}-${ARCH}"

if ! docker --version >/dev/null 2>&1; then
    printf "${YELLOW}A docker provider is required for ddev. Please see https://ddev.readthedocs.io/en/stable/users/install/docker-installation/.${RESET}\n"
fi

TARBALL="$FILEBASE.$VERSION.tar.gz"
OLD_CHECKSUM=$(semver_compare "${VERSION}" "v1.19.3")
SHAFILE=checksums.txt
if [ ${OLD_CHECKSUM} != 1 ]; then
  SHAFILE="$TARBALL.sha256.txt"
fi

curl -fsSL "$RELEASE_BASE_URL/$TARBALL" -o "${TMPDIR}/${TARBALL}" || (printf "${RED}Failed downloading $RELEASE_BASE_URL/$TARBALL${RESET}\n" && exit 108)
curl -fsSL "$RELEASE_BASE_URL/$SHAFILE" -o "${TMPDIR}/${SHAFILE}" || (printf "${RED}Failed downloading $RELEASE_BASE_URL/$SHAFILE${RESET}\n" && exit 109)

cd $TMPDIR
$SHACMD -c "$SHAFILE"
tar -xzf $TARBALL


printf "${GREEN}Download verified. Ready to place ddev, ddev_hostname, and mkcert in your /usr/local/bin.${RESET}\n"

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
    printf "${YELLOW}Running \"sudo mv  -f ${ARTIFACTS} /usr/local/bin/\" Please enter your password if prompted.${RESET}\n"
fi
for item in ${ARTIFACTS}; do
  if [ -f ${item} ]; then
    chmod +x ${item}
    ${SUDO} mv ${item} /usr/local/bin/
  fi
done

if command -v brew >/dev/null ; then
    if [ -d "$(brew --prefix)/etc/bash_completion.d" ]; then
        bash_completion_dir=$(brew --prefix)/etc/bash_completion.d
        cp ddev_bash_completion.sh $bash_completion_dir/ddev
        printf "${GREEN}Installed ddev bash completions in $bash_completion_dir${RESET}\n"
        rm ddev_bash_completion.sh
    else
        printf "${YELLOW}Bash completion for ddev was not installed. You may manually install /tmp/ddev_bash_completion.sh in your bash_completion.d directory.${RESET}\n"
    fi

    if  [ -d "$(brew --prefix)/share/zsh-completions" ] && [ -f ddev_zsh_completion.sh ]; then
        zsh_completion_dir=$(brew --prefix)/share/zsh-completions
        cp ddev_zsh_completion.sh $zsh_completion_dir/_ddev
        printf "${GREEN}Installed ddev zsh completions in $zsh_completion_dir${RESET}\n"
        rm ddev_zsh_completion.sh
    else
        printf "${YELLOW}zsh completion for ddev was not installed. You may manually install ${TMPDIR}/ddev_zsh_completion.sh in your zsh-completions directory.${RESET}\n"
    fi
fi

rm -f ${TMPDIR}$TARBALL ${TMPDIR}/$SHAFILE

if command -v mkcert >/dev/null; then
  printf "${YELLOW}Running mkcert -install, which may request your sudo password.'.${RESET}\n"
  mkcert -install
fi

hash -r

printf "${GREEN}ddev is now installed. Run \"ddev\" and \"ddev --version\" to verify your installation and see usage.${RESET}\n"
