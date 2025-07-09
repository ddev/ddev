#!/bin/bash
set -eu -o pipefail

# Create temporary sudoers entry allowing all users passwordless sudo
# This is used during mkcert installation and removed afterward
echo "ALL ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/temp-mkcert-install
chmod 440 /etc/sudoers.d/temp-mkcert-install

echo "Temporary passwordless sudo created for all users"