#!/usr/bin/env bash

# Adapted from https://medium.com/@sean.handley/how-to-set-up-docker-for-mac-with-native-nfs-145151458adc

set -o errexit
set -o pipefail
set -o nounset

OS=$(uname -s)

if [ $OS != "Darwin" ]; then
  echo "This script is OSX-only. Please do not run it on any other Unix."
  exit 101
fi

if [[ $EUID -eq 0 ]]; then
  echo "This script must NOT be run with sudo/root. Please re-run without sudo." 1>&2
  exit 102
fi

mkdir -p ~/.ddev
docker run --rm -t -v /$HOME/.ddev:/tmp/junker99 busybox:latest ls //tmp/junker99 >/dev/null || ( echo "Docker does not seem to be running or functional, please check it for problems" && exit 103)

echo "
+-------------------------------------------+
| Setup native NFS on macOS for Docker
| Only localhost is allowed access;
| Your home directory is shared by default.
| But, of course, pay attention to security.
+-------------------------------------------+
"
echo "Stopping running ddev projects"
echo ""

ddev poweroff || true

echo "== Setting up nfs..."
# Share /Users folder. If the projects are elsewhere the /etc/exports will need
# to be adapted.
# If Catalina or later, the share directory has to be /System/Volumes/Data/...
SHAREDIR=${HOME}
if [ -d /System/Volumes/Data${HOME} ] ; then
    SHAREDIR=/System/Volumes/Data${HOME}
fi
LINE="${SHAREDIR} -alldirs -mapall=$(id -u):$(id -g) localhost"
FILE=/etc/exports
sudo bash -c "echo >> $FILE" || ( echo "Unable to edit /etc/exports, need Full Disk Access on Mojave and later" && exit 103 )
grep -qF -- "$LINE" "$FILE" || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

LINE="nfs.server.mount.require_resv_port = 0"
FILE=/etc/nfs.conf
grep -qF -- "$LINE" "$FILE" || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

echo "== Restarting nfsd..."
sudo nfsd enable && sudo nfsd restart

