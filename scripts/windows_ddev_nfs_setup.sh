#!/usr/bin/env bash

# Set up NFS on Windows for ddev.

set -o errexit
set -o pipefail
set -o nounset

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

sudo nssm install nfsd "C:\Program Files\ddev\winnfsd.exe" -log off "C:\\Users"
sudo nssm start nfsd

echo "winnfsd has been installed as service nfsd serving C:\Users"
echo "You can edit its behavior with 'sudo nssm edit nfsd'"
