#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip nsis jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev

if [ ! -d /home/linuxbrew/.linuxbrew/bin ] ; then
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/Linuxbrew/install/master/install.sh)"
fi

echo "export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH" >>~/.bashrc

. ~/.bashrc

brew update
for item in osslsigncode golang mkcert; do
    brew install $item || /home/linuxbrew/.linuxbrew/bin/brew upgrade $item
done

mkcert -install

sudo bash -c "printf '/home 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)\n/tmp 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)' >>/etc/exports"
sudo service nfs-kernel-server restart

# gotestsum
GOTESTSUM_VERSION=0.3.2
curl -sSL "https://github.com/gotestyourself/gotestsum/releases/download/v$GOTESTSUM_VERSION/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz" | sudo tar -xz -C /usr/local/bin gotestsum && sudo chmod +x /usr/local/bin/gotestsum

# Install ghr
GHR_RELEASE="v0.12.0"
curl -fsL -o /tmp/ghr.tar.gz https://github.com/tcnksm/ghr/releases/download/${GHR_RELEASE}/ghr_${GHR_RELEASE}_linux_amd64.tar.gz
sudo tar -C /usr/local/bin --strip-components=1 -xzf /tmp/ghr.tar.gz ghr_${GHR_RELEASE}_linux_amd64/ghr
ghr -v
