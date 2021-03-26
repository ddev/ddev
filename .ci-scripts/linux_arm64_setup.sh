#!/usr/bin/env bash

set -o errexit

# Basic tools

set -x
export GO_VERSION=1.16.2

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi

sudo apt-get -qq update
sudo rm -f /usr/local/bin/jq && sudo apt-get -qq install -y mysql-client zip jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev

curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-arm64.zip && sudo unzip -o -d /usr/local/bin /tmp/ngrok.zip

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

. ~/.bashrc

curl -sSL https://golang.org/dl/go${GO_VERSION}.linux-arm64.tar.gz -o /tmp/go.tgz && sudo rm -rf /usr/local/go && sudo tar -zxf /tmp/go.tgz -C /usr/local

git clone --branch v1.2.1 https://github.com/bats-core/bats-core.git /tmp/bats-core && pushd /tmp/bats-core >/dev/null && sudo ./install.sh /usr/local

# Install mkcert
sudo curl -sSL https://github.com/drud/mkcert/releases/download/v1.4.6/mkcert-v1.4.6-linux-arm64 -o /usr/local/bin/mkcert && sudo chmod +x /usr/local/bin/mkcert
mkcert -install

primary_ip=$(ip route get 1 | awk '{gsub("^.*src ",""); print $1; exit}')

sudo bash -c "cat <<EOF >/etc/exports
${HOME} ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
/tmp ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
EOF"

sudo service nfs-kernel-server restart

## Configure environment so changes are picked up when the Docker daemon is restarted after upgrading
echo '{"experimental":true}' | sudo tee /etc/docker/daemon.json
export DOCKER_CLI_EXPERIMENTAL=enabled

if [ ! -z "${TRAVIS_BUILD_NUMBER:-}" ]; then
  ## On Travis Upgrade to latest Docker CE BuildKit support
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
  sudo add-apt-repository "deb [arch=arm64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
  sudo apt-get update
  sudo apt-get -qq -y -o Dpkg::Options::="--force-confnew" install docker-ce

  # Copy docker-compose to /usr/local/bin because Travis' pre-installed version leads to exec format error
  sudo apt-get -qq install -y docker-compose
  sudo cp /usr/bin/docker-compose /usr/local/bin/docker-compose
fi

# Show info to simplify debugging and create a builder
docker info
docker version
docker-compose version
lsb_release -a
docker buildx create --name ddev-builder-multi --use  
