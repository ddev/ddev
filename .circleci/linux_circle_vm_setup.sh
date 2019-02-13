#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip nsis jq expect nfs-kernel-server build-essential curl git

if [ ! -d /home/linuxbrew/.linuxbrew/bin ] ; then
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/Linuxbrew/install/master/install.sh)"
    export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin
    echo "export PATH=/home/linuxbrew/linuxbrew/.linuxbrew/bin:$PATH" >>~/.bashrc
fi
export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin
/home/linuxbrew/.linuxbrew/bin/brew update
/home/linuxbrew/.linuxbrew/bin/brew install osslsigncode golang docker-compose

sudo bash -c "printf '/home 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)\n/tmp 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)' >>/etc/exports"
sudo service nfs-kernel-server restart

# Remove existing docker and install from their apt package
sudo apt-get remove docker docker-engine docker.io
sudo apt-get install \
    apt-transport-https \
    ca-certificates \
    curl \
    software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"
sudo apt-get update -qq
sudo apt-get install -qq docker-ce

# gotestsum
GOTESTSUM_VERSION=0.3.2
curl -sSL "https://github.com/gotestyourself/gotestsum/releases/download/v$GOTESTSUM_VERSION/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz" | sudo tar -xz -C /usr/local/bin gotestsum && sudo chmod +x /usr/local/bin/gotestsum

# Install ghr
GHR_RELEASE="v0.12.0"
curl -fsL -o ~/${GHR_RELEASE}.tar.gz https://github.com/tcnksm/ghr/releases/download/${GHR_RELEASE}/ghr_${GHR_RELEASE}_linux_amd64.tar.gz
tar -xzf ~/${GHR_RELEASE}.tar.gz -C ~/
ln -s ~/${GHR_RELEASE}/ghr ~/ghr
~/ghr -v
