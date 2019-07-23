#!/usr/bin/env bash

set -o errexit
set -x


# Basic tools
brew update >/dev/null 2>/dev/null

# Get docker in first so we can install it and work on other things
brew cask install docker ngrok
sudo /Applications/Docker.app/Contents/MacOS/Docker --quit-after-install --unattended
nohup /Applications/Docker.app/Contents/MacOS/Docker --unattended &

brew tap drud/ddev

for item in mysql-client zip nsis jq expect coreutils golang ddev mkcert osslsigncode; do
    brew install $item || brew upgrade $item
done

brew link --force mysql-client

# homebrew sometimes removes /usr/local/etc/my.cnf.d
mkdir -p /usr/local/etc/my.cnf.d

mkcert -install

curl -fsSL -o /tmp/gotestsum.tgz https://github.com/gotestyourself/gotestsum/releases/download/v0.3.2/gotestsum_0.3.2_darwin_amd64.tar.gz && tar -C /usr/local/bin -zxf /tmp/gotestsum.tgz gotestsum

# gotestsum
GOTESTSUM_VERSION=0.3.2
curl -fsSL -o /tmp/gotestsum.tgz https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_darwin_amd64.tar.gz && tar -C /usr/local/bin -zxf /tmp/gotestsum.tgz gotestsum

sudo bash -c "cat <<EOF >/etc/exports
${HOME} -alldirs -mapall=$(id -u):$(id -g) localhost
/private/var -alldirs -mapall=$(id -u):$(id -g) localhost
EOF"

LINE="nfs.server.mount.require_resv_port = 0"
FILE=/etc/nfs.conf
grep -qF -- "$LINE" "$FILE" || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

sudo nfsd enable && sudo nfsd restart


while ! docker ps 2>/dev/null ; do
  sleep 5
  echo "Waiting for docker to come up: $(date)"
done
