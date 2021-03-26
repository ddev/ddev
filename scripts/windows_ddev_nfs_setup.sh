#!/usr/bin/env bash

# Set up NFS on Windows for ddev.

set -o errexit
set -o pipefail
set -o nounset

# Currently Windows ddev containers run as UID 1000
# We want the NFS mount to do the same.
DDEV_WINDOWS_UID=1000
DDEV_WINDOWS_GID=1000

nfs_addr=127.0.0.1

mkdir -p ~/.ddev
docker run --rm -t -v "/$HOME/.ddev:/tmp/junker99" busybox:latest ls //tmp/junker99 >/dev/null || ( echo "Docker does not seem to be running or functional, please check it for problems" && exit 101)


status=uninstalled
if nssm status nfsd 2>/dev/null ; then
    status=$(nssm status nfsd)
fi
if [ "$status" = "SERVICE_RUNNING" ] ; then
    echo "nfsd is already running. You can configure its behavior with 'nssm edit nfsd'."
    exit 0
fi

if [ "$status" = "SERVICE_STOPPED" ] ; then
    echo "nfsd was already installed, starting it. You can configure its behavior with 'nssm edit nfsd'"
    sudo nssm start nfsd
    exit 0
fi

if ! command -v winnfsd.exe >/dev/null; then
    echo "winnfsd.exe does not seem to be installed or is not in the PATH"
    exit 102
fi
winnfsd=$(command -v winnfsd.exe)

if [ -f "$HOME/.ddev/nfs_exports.txt" ]; then
    printf "$HOME/.ddev/nfs_exports.txt already exists, not overwriting it, you will be responsible for its exports.\n"
else
    echo "
# Exports for winnfsd for ddev
# You can edit these yourself to match your workflow
# But nfs must share your project directory
# Additional lines can be added for additional directories or drives.
${HOMEDRIVE}${HOMEPATH} > ${HOME}" >"$HOME/.ddev/nfs_exports.txt"
fi
sudo nssm install nfsd "${winnfsd}" -id ${DDEV_WINDOWS_UID} ${DDEV_WINDOWS_GID} -addr $nfs_addr -log off -pathFile "\"$HOMEDRIVE$HOMEPATH\.ddev\nfs_exports.txt\""
sudo nssm start nfsd || true
sleep 2
nssm status nfsd

echo "winnfsd has been installed as service nfsd serving the mounts in ~/.ddev/nfs_exports.txt"
echo "You can edit that file and restart the nfsd service"
echo "with 'sudo nssm restart nfsd'"
echo "Or you can edit its behavior with 'sudo nssm edit nfsd'"
