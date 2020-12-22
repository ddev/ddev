#!/usr/bin/env bash

set -eu -o pipefail

# Needed for the build jobs, e.g. for Windows makensis

export HOMEBREW_NO_AUTO_UPDATE=1
export HOMEBREW_NO_INSTALL_CLEANUP=1

for item in osslsigncode makensis; do
    brew install $item >/dev/null || /home/linuxbrew/.linuxbrew/bin/brew upgrade $item >/dev/null
done

# Get the Stubs and Plugins for makensis; the linux makensis build doesn't do this.
wget https://sourceforge.net/projects/nsis/files/NSIS%203/3.06.1/nsis-3.06.1.zip/download && sudo unzip -o -d /usr/local/share download && sudo mv /usr/local/share/nsis-3.06.1 /usr/local/share/nsis
wget https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -o -d /usr/local/share/nsis EnVar-Plugin.zip
wget https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -o -d /usr/local/share/nsis/Plugins INetC.zip

set +eu
