#!/usr/bin/env bash

set -o errexit
set -x

# Docker desktop after 31259 refuses to install using root
DOCKER_URL=https://download.docker.com/mac/stable/31259/Docker.dmg
curl -O -sSL $DOCKER_URL
open -W Docker.dmg && cp -r /Volumes/Docker/Docker.app /Applications

# Get docker in first so we can install it and work on other things
sudo /Applications/Docker.app/Contents/MacOS/Docker --quit-after-install --unattended
nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &

brew install homebrew/cask/ngrok >/dev/null
brew install drud/ddev/ddev >/dev/null
brew unlink python@2 >/dev/null || true

brew install mysql-client zip makensis jq expect coreutils golang ddev mkcert osslsigncode ghr gnu-getopt libgsf glib pcre >/dev/null || true
brew link mysql-client zip makensis jq expect coreutils golang ddev mkcert osslsigncode ghr gnu-getopt libgsf glib pcre >/dev/null

brew link --force mysql-client >/dev/null

# Get the Plugins for NSIS
curl -fsSL -o /tmp/EnVar-Plugin.zip https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -o -d /usr/local/share/nsis /tmp/EnVar-Plugin.zip
curl -fsSL -o /tmp/INetC.zip https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -o -d /usr/local/share/nsis/Plugins /tmp/INetC.zip

# homebrew sometimes removes /usr/local/etc/my.cnf.d
mkdir -p "$(brew --prefix)/etc/my.cnf.d"

mkcert -install

pip3 install -q yq

sudo bash -c "cat <<EOF >/etc/exports
${HOME} -alldirs -mapall=$(id -u):$(id -g) localhost
/private/var -alldirs -mapall=$(id -u):$(id -g) localhost
EOF"

LINE="nfs.server.mount.require_resv_port = 0"
FILE=/etc/nfs.conf
grep -qF -- "$LINE" "$FILE" || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

sudo nfsd enable && sudo nfsd restart

timeout -v 10m bash -c 'while ! docker ps 2>/dev/null ; do
  sleep 5
  echo "Waiting for docker to come up: $(date)"
done'

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD:-}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi
