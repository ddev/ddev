#!/usr/bin/env bash

# Requires $DDEV_GITHUB_TOKEN, a token with public repo power

set -o errexit
set -o pipefail
set -o nounset

if [ $# != 4 ] && [ $# != 5 ]; then
    printf "Incorrect args provided. You gave '$*'; Correct Arguments: HOMEBREW_REPO (homebrew repo like ddev/homebrew-ddev) \nPROJECT_NAME (like ddev) \nVERSION_NUMBER (like v1.16.5) \nARTIFACTS_DIR (like /home/circleci/artifacts)\nGITHUB_USERNAME (defaults to ddev)" && exit 101
fi

# For testing, you can change GITHUB_USERNAME to, for example, rfay so releases can be tested
# without bothering people.
HOMEBREW_REPO=$1
PROJECT_NAME=$2
VERSION_NUMBER=$3
ARTIFACTS_DIR=$4
GITHUB_USERNAME=${5:-ddev}

NO_V_VERSION=${VERSION_NUMBER#v}
SOURCE_URL="https://github.com/${GITHUB_USERNAME}/${PROJECT_NAME}/archive/${VERSION_NUMBER}.tar.gz"
echo "HOMEBREW_REPO=${HOMEBREW_REPO} PROJECT_NAME=${PROJECT_NAME} VERSION_NUMBER=${VERSION_NUMBER} ARTIFACTS_DIR=${ARTIFACTS_DIR} GITHUB_USERNAME=${GITHUB_USERNAME} NO_V_VERSION=${NO_V_VERSION} SOURCE_URL=${SOURCE_URL}"
SOURCE_SHA=$(curl -sSL ${SOURCE_URL} | shasum -a 256 | awk '{print $1}')

LINUX_BOTTLE_SHA=$(awk '{print $1}' <"${ARTIFACTS_DIR}/${PROJECT_NAME}-${NO_V_VERSION}.x86_64_linux.bottle.tar.gz.sha256.txt")
MACOS_AMD64_BOTTLE_SHA=$(awk '{print $1}' <${ARTIFACTS_DIR}/${PROJECT_NAME}-${NO_V_VERSION}.high_sierra.bottle.tar.gz.sha256.txt)
MACOS_ARM64_BOTTLE_SHA=$(awk '{print $1}' <${ARTIFACTS_DIR}/${PROJECT_NAME}-${NO_V_VERSION}.arm64_big_sur.bottle.tar.gz.sha256.txt)


TMPDIR=$(mktemp -d)
cd ${TMPDIR} && git clone https://github.com/${HOMEBREW_REPO} && cd "$(basename ${HOMEBREW_REPO})"

cat >Formula/${PROJECT_NAME}.rb <<END
class Ddev < Formula
  desc "Local web development system"
  homepage "https://ddev.readthedocs.io/"
  url "${SOURCE_URL}"
  sha256 "${SOURCE_SHA}"
  license "apache-2.0"
  head "https://github.com/ddev/ddev.git", branch: "main"

  depends_on "mkcert" => :run
  depends_on "nss" => :run
  depends_on "go" => :build
  depends_on "make" => :build

  bottle do
    root_url "https://github.com/${GITHUB_USERNAME}/${PROJECT_NAME}/releases/download/${VERSION_NUMBER}/"
    sha256 cellar: :any_skip_relocation, x86_64_linux: "${LINUX_BOTTLE_SHA}"
    sha256 cellar: :any_skip_relocation, high_sierra: "${MACOS_AMD64_BOTTLE_SHA}"
    sha256 cellar: :any_skip_relocation, arm64_big_sur: "${MACOS_ARM64_BOTTLE_SHA}"
  end
  def install
    system "make"
    system "mkdir", "-p", "#{bin}"
    if OS.mac?
        if Hardware::CPU.arm?
            system "cp", ".gotmp/bin/darwin_arm64/ddev", "#{bin}/ddev"
            system ".gotmp/bin/darwin_arm64/ddev_gen_autocomplete"
        else
            system "cp", ".gotmp/bin/darwin_amd64/ddev", "#{bin}/ddev"
            system ".gotmp/bin/darwin_amd64/ddev_gen_autocomplete"
        end
    end
    if OS.linux?
      system "cp", ".gotmp/bin/linux_amd64/ddev", "#{bin}/ddev"
      system ".gotmp/bin/linux_amd64/ddev_gen_autocomplete"
    end
    bash_completion.install ".gotmp/bin/ddev_bash_completion.sh" => "ddev"
    zsh_completion.install ".gotmp/bin/ddev_zsh_completion.sh" => "_ddev"
    fish_completion.install ".gotmp/bin/ddev_fish_completion.sh" => "ddev.fisn"
  end

  def caveats
    <<~EOS
            Make sure to do a 'mkcert -install' if you haven't done it before, it may require your sudo password.
      #{"      "}
            ddev requires docker or colima.
            See https://ddev.readthedocs.io/en/stable/users/install/docker-installation/
    EOS
  end

  test do
    system "#{bin}/ddev", "--version"
  end
end
END

git config user.email "randy@randyfay.com"
git config user.name "rfay"
git add -u
git commit -m "Homebrew bump to ${VERSION_NUMBER}"

git push -q https://${GITHUB_USERNAME}:${DDEV_GITHUB_TOKEN}@github.com/${HOMEBREW_REPO}.git master
