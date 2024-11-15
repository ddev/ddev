#!/usr/bin/env bash

set -eu -o pipefail
#set -x

mkdir -p ~/.ssh
cp -r /tmp/sshtmp/* ~/.ssh
chmod -R go-rwx ~/.ssh
cd ~/.ssh
grep -l '^-----BEGIN .* PRIVATE KEY-----' * | xargs ssh-add
