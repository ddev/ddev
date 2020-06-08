#!/bin/bash

# Requires $AUR_SSH_PRIVATE_KEY, a private key in environment variable
# This environment variable must be a single line, with \n replaced by "<SPLIT>"

set -o errexit
set -o pipefail
set -o nounset

if [ -z ${AUR_SSH_PRIVATE_KEY} ] ; then
    printf "\$AUR_SSH_PRIVATE_KEY must be set in the environment. It should be a single line with \n replaced by <SPLIT>" && exit 101
fi
if [ $# != 3 ]; then
    printf "Arguments: AUR_REPO (AUR repo ddev-bin or ddev-edge-bin)  \nVERSION_NUMBER (like v1.14.2) \nARTIFACTS_DIR (like /home/circleci/artifacts)\n" && exit 102
fi

AUR_USERNAME=ddev-releaser
AUR_REPO=$1
VERSION_NUMBER=$2
ARTIFACTS_DIR=$3
#NO_V_VERSION=$(echo ${VERSION_NUMBER} | awk  -F"-" '{ OFS="-"; sub(/^./, "", $1); printf $0; }')
EDGE_DESCRIPTION=""
if [ ${AUR_REPO} = "ddev-edge-bin" ] ; then EDGE_DESCRIPTION="  (edge channel)"; fi

# Add temporary private key provided by CircleCI context
echo $AUR_SSH_PRIVATE_KEY | perl -p -e 's/<SPLIT>/\n/g' >/tmp/id_rsa_aur && chmod 600 /tmp/id_rsa_aur
TMPDIR=$(mktemp -d)
ssh-add /tmp/id_rsa_aur
pushd ${TMPDIR} && git clone ssh://aur@aur.archlinux.org/${AUR_REPO}.git && cd ${AUR_REPO}

cat >PKGBUILD <<END
_name="ddev"
pkgname="${AUR_REPO}"
pkgver="${VERSION_NUMBER}"
pkgrel=1
pkgdesc='DDEV-Local: a local PHP development environment system${EDGE_DESCRIPTION}'
arch=('x86_64')
url='https://github.com/drud/ddev'
license=('Apache')
provides=("$_name")
conflicts=("$_name")
depends=('docker' 'docker-compose')
optdepends=('bash-completion: subcommand completion support')
source=("https://github.com/drud/ddev/releases/download/v$pkgver/ddev_linux.$pkgver.tar.gz")
sha256sums=("4d1082b5ac67829347bb8029582ef4719f2e916946cff7ad4b48042ada710e37")

package() {
	install -D -m 0755 ddev "$pkgdir/usr/bin/ddev"
	install -D -m 0755 ddev_bash_completion.sh "$pkgdir/usr/share/bash-completion/completions/ddev"
}

END

git config user.email "randy+ddev-releaser@randyfay.com"
git config user.name "ddev-releaser"
git add -u
git commit -m "AUR bump to ${VERSION_NUMBER}"

git push
popd
