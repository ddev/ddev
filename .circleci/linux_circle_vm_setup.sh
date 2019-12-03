#!/usr/bin/env bash

set -o errexit
set -x

# Basic tools

sudo apt-get update -qq
sudo apt-get install -qq mysql-client realpath zip jq expect nfs-kernel-server build-essential curl git libnss3-tools libcurl4-gnutls-dev

curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && sudo unzip -d /usr/local/bin /tmp/ngrok.zip

if [ ! -d /home/linuxbrew/.linuxbrew/bin ] ; then
    sh -c "$(curl -fsSL https://raw.githubusercontent.com/Linuxbrew/install/master/install.sh)"
fi

echo "export PATH=/home/linuxbrew/.linuxbrew/bin:$PATH" >>~/.bashrc

. ~/.bashrc

brew update && brew tap drud/ddev
for item in osslsigncode golang mkcert ddev makensis; do
    brew install $item || /home/linuxbrew/.linuxbrew/bin/brew upgrade $item
done

# Get the Stubs and Plugins for makensis; the linux makensis build doesn't do this.
wget https://sourceforge.net/projects/nsis/files/NSIS%203/3.04/nsis-3.04.zip/download && sudo unzip -d /usr/local/share download && sudo mv /usr/local/share/nsis-3.04 /usr/local/share/nsis
wget https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -d /usr/local/share/nsis EnVar-Plugin.zip
wget https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -d /usr/local/share/nsis/Plugins INetC.zip

mkcert -install

primary_ip=$(ip route get 1 | awk '{gsub("^.*src ",""); print $1; exit}')

sudo bash -c "cat <<EOF >/etc/exports
${HOME} ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
/tmp ${primary_ip}/255.255.255.255(rw,sync,no_subtree_check)
EOF"

sudo service nfs-kernel-server restart

# gotestsum
GOTESTSUM_VERSION=0.3.2
curl -sSL "https://github.com/gotestyourself/gotestsum/releases/download/v$GOTESTSUM_VERSION/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz" | sudo tar -xz -C /usr/local/bin gotestsum && sudo chmod +x /usr/local/bin/gotestsum

# Install ghr
GHR_RELEASE="v0.12.0"
curl -fsL -o /tmp/ghr.tar.gz https://github.com/tcnksm/ghr/releases/download/${GHR_RELEASE}/ghr_${GHR_RELEASE}_linux_amd64.tar.gz
sudo tar -C /usr/local/bin --strip-components=1 -xzf /tmp/ghr.tar.gz ghr_${GHR_RELEASE}_linux_amd64/ghr
ghr -v
