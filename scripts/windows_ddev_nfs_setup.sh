#!/usr/bin/env bash

# Set up NFS on Windows for ddev.

set -o errexit
set -o pipefail
set -o nounset

if [ $(nssm status nfsd) = "SERVICE_RUNNING" ] ; then
    echo "nfsd is already running. You can configure its behavior with 'nssm edit nfsd'." && exit 0
fi

if [ $(nssm status nfsd) = "SERVICE_STOPPED" ] ; then
    sudo nssm start nfsd && ( echo "nfsd was already installed, starting it. You can configure its behavior with 'nssm edit nfsd'" && exit 0)
fi

sudo nssm install nfsd "C:\Program Files\ddev\winnfsd.exe" -log off "C:\\Users"
sudo nssm start nfsd

echo "winnfsd has been installed serving C:\Users"
echo "You can edit its behavior with 'nssm edit nfsd'"
