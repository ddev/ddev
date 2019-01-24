#!/usr/bin/env bash

# Adapted from https://medium.com/@sean.handley/how-to-set-up-docker-for-mac-with-native-nfs-145151458adc

set -o errexit
set -o pipefail
set -o nounset

OS=$(uname -s)
variant=$(lsb_release -a 2>/dev/null | awk -F' *: *' '/Distributor ID/ { print $2 }')

if [ $OS != "Linux" ]; then
  echo "This script is Linux-only and tailored for Debian/Ubuntu. Please do not run it on any other system."
  exit 101
fi

if [ ${variant} != "Debian" ] && [ ${variant} != "Ubuntu" ] ; then
  echo "This script is tailored for Debian/Ubuntu. Please do not run it on any other system. "
  echo "It can easily be tailored for other systems and their nfs requirements"
  exit 102
fi

if [[ $EUID -eq 0 ]]; then
  echo "This script should NOT be run with sudo/root. Please re-run without sudo." 1>&2
  exit 103
fi

docker run --rm -t -v /$HOME:/tmp/junker99 busybox:latest ls //tmp/junker99 >/dev/null || ( echo "Docker does not seem to be running or functional, please check it for problems" && exit 103)

echo "
+--------------------------------------+
| Setup native NFS on Linux for Docker
| Allowing NFS access to /home has
| security implications you should
| consider. You'll want your firewall
| turned on.
+--------------------------------------+
"
echo "Stopping running ddev projects"

ddev rm -a

echo "Installing nfs-kernel-server"
sudo apt-get update -qq
sudo apt-get install -qq nfs-kernel-server

echo "== Setting up nfs..."
# Share /home folder. If the projects are elsewhere the /etc/exports will need
# to be adapted. This grants access to all unrouteable ("public") IP addresses
# (10.*, 172.16-172.28..., 192.168.*)
# You are welcome to edit and limit it to the addresses you prefer.
FILE=/etc/exports
LINE="/home 10.0.0.0/255.0.0.0(rw,sync,no_subtree_check) 172.16.0.0/255.240.0.0(rw,sync,no_subtree_check) 192.168.0.0/255.255.0.0(rw,sync,no_subtree_check)"
grep -qF -- "$LINE" "$FILE" || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

echo "== Restarting nfs-kernel-server..."
sudo systemctl restart nfs-kernel-server


