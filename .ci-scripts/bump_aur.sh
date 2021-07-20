#!/bin/bash

# Requires $AUR_SSH_PRIVATE_KEY, a private key in environment variable
# This environment variable must be a single line, with \n replaced by "<SPLIT>"

set -o errexit
set -o pipefail
set -o nounset

if [ -z "${AUR_SSH_PRIVATE_KEY:-}" ]; then
    printf "\$AUR_SSH_PRIVATE_KEY must be set in the environment. It should be a single line with \n replaced by <SPLIT>" && exit 101
fi
if [ "$#" != "3" ]; then
    printf "Arguments: AUR_REPO (AUR repo ddev-bin or ddev-edge-bin)  \nVERSION_NUMBER (like v1.14.2) \nARTIFACTS_DIR (like /home/circleci/artifacts)\n" && exit 102
fi

# For testing, you can change GITHUB_USERNAME to, for example, rfay so releases can be tested
# without bothering people.
GITHUB_USERNAME=drud
AUR_USERNAME=ddev-releaser
AUR_REPO=$1
VERSION_NUMBER=$2
ARTIFACTS_DIR=$3
NO_V_VERSION=$(echo ${VERSION_NUMBER} | awk  -F"-" '{ OFS="-"; sub(/^./, "", $1); printf $0; }')
LINUX_HASH=$(cat $ARTIFACTS_DIR/ddev_linux-amd64.${VERSION_NUMBER}.tar.gz.sha256.txt | awk '{print $1;}' )
LINUX_TARBALL_URL=https://github.com/${GITHUB_USERNAME}/ddev/releases/download/${VERSION_NUMBER}/ddev_linux-amd64.${VERSION_NUMBER}.tar.gz
if [ ! -z "${LINUX_TARBALL_OVERRIDE:-}" ]; then
    LINUX_TARBALL_URL=${LINUX_TARBALL_OVERRIDE}
    LINUX_HASH=$(curl -sSL "${LINUX_TARBALL_URL}.sha256.txt" | awk '{print $1}')
fi

EDGE_DESCRIPTION=""
if [ ${AUR_REPO} = "ddev-edge-bin" ] ; then EDGE_DESCRIPTION="  (edge channel)"; fi

# Add temporary private key provided by CircleCI context
echo $AUR_SSH_PRIVATE_KEY | perl -p -e 's/<SPLIT>/\n/g' >/tmp/id_rsa_aur && chmod 600 /tmp/id_rsa_aur

eval $(ssh-agent)
ssh-add /tmp/id_rsa_aur
rm -rf ${AUR_REPO}
git config --global core.sshCommand 'ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no'
git clone ssh://aur@aur.archlinux.org/${AUR_REPO}.git && pushd ${AUR_REPO} && touch .BUILDINFO && chmod -R ugo+w .

_name="ddev"
cat >PKGBUILD <<END
pkgname="${AUR_REPO}"
pkgver=$(echo ${VERSION_NUMBER} | tr '-' '_')
pkgrel=1
pkgdesc='DDEV-Local: a local PHP development environment system${EDGE_DESCRIPTION}'
arch=('x86_64')
url='https://github.com/drud/ddev'
license=('Apache')
provides=("$_name")
conflicts=("$_name")
depends=('docker' 'docker-compose')
optdepends=('bash-completion: subcommand completion support')
source=("${LINUX_TARBALL_URL}")
sha256sums=("${LINUX_HASH}")

package() {
	install -D -m 0755 ddev "\$pkgdir/usr/bin/ddev"
	install -D -m 0755 ddev_bash_completion.sh "\$pkgdir/usr/share/bash-completion/completions/ddev"
}
END

docker run --rm --mount type=bind,source=$(pwd),target=/tmp/ddev-bin --workdir=/tmp/ddev-bin drud/arch-aur-builder bash -c "makepkg --printsrcinfo > .SRCINFO && makepkg -s"

git config user.email "randy+ddev-releaser@randyfay.com"
git config user.name "ddev-releaser"

git commit -am "AUR bump to ${VERSION_NUMBER}"

git push
popd
