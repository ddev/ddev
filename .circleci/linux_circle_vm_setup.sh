#!/usr/bin/env bash

set -o errexit

# Basic tools

set -x

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi

sudo apt-get update -qq
sudo apt-get install -qq mysql-client coreutils zip jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev mkdocs postgresql-client

curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && sudo unzip -o -d /usr/local/bin /tmp/ngrok.zip

if [ ! -d /home/linuxbrew/.linuxbrew/bin ] ; then
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi

echo "export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH" >>~/.bashrc

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

. ~/.bashrc

for item in ddev/ddev/ddev golang makensis mingw-w64 mkcert; do
    brew install $item >/dev/null || brew upgrade $item >/dev/null
done

git clone --branch v1.2.1 https://github.com/bats-core/bats-core.git /tmp/bats-core && pushd /tmp/bats-core >/dev/null && sudo ./install.sh /usr/local && popd >/dev/null

npm install --global markdownlint-cli
markdownlint --version
# readthedocs has ancient version of mkdocs in it.
pyenv global 3.9.7 # added to make CircleCi give us pip3
pip3 install -q mkdocs==0.17.5

# Get the Stubs and Plugins for makensis; the linux makensis build doesn't do this.
pwd && ./.ci-scripts/nsis_setup.sh /usr/local/share/nsis

mkcert -install

primary_ip=$(ip route get 1 | awk '{gsub("^.*src ",""); print $1; exit}')

sudo bash -c "cat <<EOF >/etc/exports
${HOME} ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
/tmp ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
EOF"

sudo service nfs-kernel-server restart

# Install ghr
GHR_RELEASE="v0.14.0"
curl -fsL -o /tmp/ghr.tar.gz https://github.com/tcnksm/ghr/releases/download/${GHR_RELEASE}/ghr_${GHR_RELEASE}_linux_amd64.tar.gz
sudo tar -C /usr/local/bin --strip-components=1 -xzf /tmp/ghr.tar.gz ghr_${GHR_RELEASE}_linux_amd64/ghr
