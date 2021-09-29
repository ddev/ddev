#!/bin/bash

# Set up nsis after it's installed
# $1 should be the target directory where nsis is
set -eu -o pipefail

if [ $1 = "" ]; then
  echo "first arg must be the NSIS_HOME"
  echo "For example nsis_setup.sh /usr/share/nsis"
  echo "For example nsis_setup.sh /usr/local/share/nsis"
  echo "For example, nsis_setup.sh /opt/homebrew/Cellar/makensis/3.07/share/nsis"
  exit 1
fi
NSIS_HOME=$1
wget https://sourceforge.net/projects/nsis/files/NSIS%203/3.06.1/nsis-3.06.1.zip/download && sudo unzip -o -d /usr/share download && sudo mv $NSIS_HOME-3.06.1 $NSIS_HOME
wget https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -o -d $NSIS_HOME EnVar-Plugin.zip
wget https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -o -d $NSIS_HOME/Plugins INetC.zip
