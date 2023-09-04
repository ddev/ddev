#!/bin/bash

# Set up nsis after it's installed
# $1 should be the target directory where nsis is
set -eu -o pipefail

NSIS_VERSION=3.06.1

if [ $1 = "" ]; then
  echo "first arg must be the NSIS_HOME"
  echo "For example nsis_setup.sh /usr/share/nsis"
  echo "For example nsis_setup.sh /usr/local/share/nsis"
  echo "For example, nsis_setup.sh /opt/homebrew/share/nsis"
  echo "For example, nsis_setup.sh /home/linuxbrew/.linuxbrew/Cellar/makensis/3.08/share/nsis"
  echo "For example, nsis_setup.sh /c/Program Files (x86)/NSIS"
  echo "For example, nsis_setup.sh /c/ProgramData/Chocolatey/lib/nsis"
  exit 1
fi
NSIS_HOME=$1
curl -sfL -o /tmp/nsis.zip https://sourceforge.net/projects/nsis/files/NSIS%203/${NSIS_VERSION}/nsis-${NSIS_VERSION}.zip/download && unzip -o -d /tmp/nsistemp /tmp/nsis.zip && sudo mv /tmp/nsistemp/*/* "${NSIS_HOME}" && rm -rf /tmp/nsistemp /tmp/nsis.zip
curl -sfL -o /tmp/EnVar-Plugin.zip https://github.com/GsNSIS/EnVar/releases/latest/download/EnVar-Plugin.zip && sudo unzip -o -d $"${NSIS_HOME}" /tmp/EnVar-Plugin.zip && rm /tmp/EnVar-Plugin.zip
curl -sfL -o /tmp/INetC.zip https://github.com/DigitalMediaServer/NSIS-INetC-plugin/releases/latest/download/INetC.zip && sudo unzip -o -d "${NSIS_HOME}/Plugins" /tmp/INetC.zip && rm /tmp/INetC.zip
curl -sfL -o /tmp/ExecDos.zip https://nsis.sourceforge.io/mediawiki/images/0/0f/ExecDos.zip && sudo unzip -o -d "${NSIS_HOME}" /tmp/ExecDos.zip && rm /tmp/ExecDos.zip
curl -sfL -o /tmp/ZipDLL.zip https://nsis.sourceforge.io/mediawiki/images/d/d9/ZipDLL.zip && sudo unzip -o -d "${NSIS_HOME}/Plugins/x86-unicode" /tmp/ZipDLL.zip && rm /tmp/ZipDLL.zip
curl -sfL -o /tmp/NsUnzip.zip https://nsis.sourceforge.io/mediawiki/images/8/88/NsUnzip.zip && sudo unzip -o -d "${NSIS_HOME}/Plugins/x86-unicode" /tmp/NsUnzip.zip && rm /tmp/NsUnzip.zip
curl -sfL -o /tmp/NSISunzU.zip https://nsis.sourceforge.io/mediawiki/images/5/5a/NSISunzU.zip && sudo unzip -o -d "${NSIS_HOME}/Plugins/x86-unicode" /tmp/NSISunzU.zip && rm /tmp/NSISunzU.zip
sudo cp "$NSIS_HOME/Plugins/x86-unicode/NSISunzU/Plugin unicode/nsisunz.dll" $NSIS_HOME/Plugins/x86-unicode

