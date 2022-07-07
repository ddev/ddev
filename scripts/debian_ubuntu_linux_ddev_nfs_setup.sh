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

mkdir -p ~/.ddev
docker run --rm -t -v /$HOME/.ddev:/tmp/junker99 busybox:stable ls //tmp/junker99 >/dev/null || ( echo "Docker does not seem to be running or functional, please check it for problems" && exit 103)

echo "
+-------------------------------------------------------------------------+
| Setup native NFS on Linux for Docker
| Only the primary IP of machine is allowed client access;
| Your home directory is shared by default.
| But, of course, pay attention to security.
|
| Please note that ddev NFS support on Linux is not a particular
| performance gain, since docker on Linux is already very fast.
| You may not need/want to use it.
| If you do use it, this script sets of very restrictive /etc/exports that
| you will probably need to modify to suit your needs, especially if you're
| using DHCP and your IP address changes periodically.
| Please see https://ddev.readthedocs.io/en/stable/users/performance/#debianubuntu-linux-nfs-setup
| for more information.
+-------------------------------------------------------------------------+
"
echo "Stopping running ddev projects"

ddev stop -a || true

echo "Installing nfs-kernel-server"
sudo apt-get update -qq
sudo apt-get install -qq nfs-kernel-server

primary_ip=$(ip route get 1 | awk '{gsub("^.*src ",""); print $1; exit}')
echo "== Setting up nfs..."
# Share /home folder. If the projects are elsewhere the /etc/exports will need
# to be adapted. This grants access ONLY to the host machine's primary IP address
# (at the time the script was run)
# You may need to adapt it to be less restrictive in your environment,
# or of course just use the firewall to restrict access.
# You are welcome to edit and limit it to the addresses you prefer.
# Please see https://ddev.readthedocs.io/en/stable/users/install/performance/
# for more information.
FILE=/etc/exports
LINE="${HOME} ${primary_ip}(rw,sync,no_subtree_check)"
grep -qF -- "$LINE" "$FILE" 2>/dev/null || ( sudo echo "$LINE" | sudo tee -a $FILE > /dev/null )

echo "== Restarting nfs-kernel-server..."
sudo systemctl restart nfs-kernel-server


