#!/bin/bash

# Requires $DDEV_GITHUB_TOKEN, a token with public repo power
# TODO: Handle both stable and edge releases

set -o errexit
set -o pipefail
set -o nounset

if [ $# != 4 ]; then
    echo "Arguments: GITHUB_REPO (homebrew repo like drud/homebrew-ddev) PROJECT_NAME (like ddev) VERSION_NUMBER (like v1.14.2) ARTIFACTS_DIR (like /home/circleci/artifacts)" && exit 101
fi

GITHUB_USERNAME=rfay
GITHUB_REPO=$1
PROJECT_NAME=$2
VERSION_NUMBER=$3
ARTIFACTS_DIR=$4
NO_V_VERSION=$(echo ${VERSION_NUMBER} | awk  -F"-" '{ OFS="-"; sub(/^./, "", $1); printf $0; }')
SOURCE_URL="https://github.com/drud/ddev/archive/${VERSION_NUMBER}.tar.gz"
SOURCE_SHA=$(curl -sSL ${SOURCE_URL} | shasum -a 256 | awk '{print $1}')

LINUX_BOTTLE_SHA=$(awk '{print $1}' <"${ARTIFACTS_DIR}/${PROJECT_NAME}-${NO_V_VERSION}.x86_64_linux.bottle.tar.gz.sha256.txt")
MACOS_BOTTLE_SHA=$(awk '{print $1}' <${ARTIFACTS_DIR}/${PROJECT_NAME}-${NO_V_VERSION}.sierra.bottle.tar.gz.sha256.txt)


TMPDIR=$(mktemp -d)
cd ${TMPDIR} && git clone https://github.com/${GITHUB_REPO} && cd $(basename ${GITHUB_REPO})

cat >Formula/${PROJECT_NAME}.rb <<END
class Ddev < Formula
  desc "ddev: a local development environment management system"
  homepage "https://ddev.readthedocs.io/en/stable/"
  url "${SOURCE_URL}"
  sha256 "${SOURCE_SHA}"

  # depends_on "docker" => :run
  # depends_on "docker-compose" => :run
  depends_on "docker" => :build
  depends_on "go" => :build
  depends_on "mkcert" => :run
  depends_on "nss" => :run

  bottle do
    root_url "https://github.com/drud/ddev/releases/download/${VERSION_NUMBER}/"
    cellar :any_skip_relocation
    sha256 "${LINUX_BOTTLE_SHA}" => :x86_64_linux
    sha256 "${MACOS_BOTTLE_SHA}" => :sierra
  end
  def install
    system "make", "VERSION=v#{version}", "COMMIT=v#{version}"
    system "mkdir", "-p", "#{bin}"
    if OS.mac?
      system "cp", ".gotmp/bin/darwin_amd64/ddev", "#{bin}/ddev"
      system ".gotmp/bin/darwin_amd64/ddev_gen_autocomplete"
    else
      system "cp", ".gotmp/bin/ddev", "#{bin}/ddev"
      system ".gotmp/bin/ddev_gen_autocomplete"
    end
    bash_completion.install ".gotmp/bin/ddev_bash_completion.sh" => "ddev"
    zsh_completion.install ".gotmp/bin/ddev_zsh_completion.sh" => "ddev"
  end

  test do
    system "#{bin}/ddev", "--version"
  end

  def caveats
  <<~EOS
Make sure to do a 'mkcert -install' if you haven't done it before, it may require your sudo password.

ddev requires docker and docker-compose.
Docker installation instructions at https://ddev.readthedocs.io/en/stable/users/docker_installation/
  EOS
  end
end
END

git config user.email "randy@randyfay.com"
git config user.name "rfay"
git add -u
git commit -m "Homebrew bump to ${VERSION_NUMBER}"

git push -q https://${GITHUB_USERNAME}:${DDEV_GITHUB_TOKEN}@github.com/${GITHUB_REPO}.git master
