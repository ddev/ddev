#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

NFS_PROJECT_DIR=~/tmp/ddev-nfs-test
mkdir -p ${NFS_PROJECT_DIR} || true && cd ${NFS_PROJECT_DIR}

ddev config --auto
ddev debug nfsmount
ddev delete -Oy

echo "nfsd seems to be set up ok"
