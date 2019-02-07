#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip nsis jq expect nfs-kernel-server build-essential curl git

if [ ! -d ~linuxbrew/.linuxbrew/bin ] ; then
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/Linuxbrew/install/master/install.sh)"
    export PATH=$PATH:~linuxbrew/.linuxbrew/bin
    echo "export PATH=$PATH:~linuxbrew/.linuxbrew/bin" >~/.bashrc
fi
brew update >/dev/null 2>/dev/null

sudo bash -c "printf '/home 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)\n/tmp 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)' >>/etc/exports"
sudo service nfs-kernel-server restart

# golang of the version we want
sudo apt-get remove -qq golang && sudo rm -rf /usr/local/go &&
wget -q -O /tmp/golang.tgz https://dl.google.com/go/go1.11.4.linux-amd64.tar.gz &&
sudo tar -C /usr/local -xzf /tmp/golang.tgz

# docker-compose
sudo rm -f /usr/local/bin/docker-compose
sudo curl -s -L "https://github.com/docker/compose/releases/download/1.23.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

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
GHR_RELEASE="ghr_v0.12.0_linux_amd64"
curl -sL https://github.com/tcnksm/ghr/releases/download/v0.12.0/${GHR_RELEASE}.tar.gz > /home/circleci/${GHR_RELEASE}.tar.gz
tar -xzf /home/circleci/${GHR_RELEASE}.tar.gz -C /home/circleci
ln -s /home/circleci/${GHR_RELEASE}/ghr /home/circleci/ghr
/home/circleci/ghr -v 
