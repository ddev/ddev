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
sudo apt-get install -qq mysql-client realpath zip jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev

curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-arm64.tgz && sudo unzip -o -d /usr/local/bin /tmp/ngrok.zip

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

. ~/.bashrc

npm install -g bats

# Install mkcert
sudo curl -sSL https://github.com/drud/mkcert/releases/download/v1.4.6/mkcert-v1.4.6-linux-arm64 -o /usr/local/bin/mkcert && sudo chmod +x /usr/local/bin/mkcert
mkcert -install

primary_ip=$(ip route get 1 | awk '{gsub("^.*src ",""); print $1; exit}')

sudo bash -c "cat <<EOF >/etc/exports
${HOME} ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
/tmp ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
EOF"

sudo service nfs-kernel-server restart
