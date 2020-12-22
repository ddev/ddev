#!/usr/bin/env bash

set -eu -o pipefail

# Needed for the build jobs, e.g. for Windows makensis

sudo apt-get update -qq
sudo apt-get install -qq osslsigncode nsis

# Get the Stubs and Plugins for makensis; the linux makensis build doesn't do this.
wget https://sourceforge.net/projects/nsis/files/NSIS%203/3.06.1/nsis-3.06.1.zip/download && sudo unzip -o -d /usr/share download && sudo mv /usr/share/nsis-3.06.1 /usr/share/nsis
wget https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -o -d /usr/share/nsis EnVar-Plugin.zip
wget https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -o -d /usr/share/nsis/Plugins INetC.zip

set +eu
