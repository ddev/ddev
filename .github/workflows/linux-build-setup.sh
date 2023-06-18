#!/usr/bin/env bash

set -eu -o pipefail

# Needed for the build jobs, e.g. for Windows makensis

sudo apt-get update -qq
sudo apt-get install -y -qq osslsigncode nsis

# Get the Stubs and Plugins for makensis; the linux makensis build doesn't do this.
./.ci-scripts/nsis_setup.sh /usr/share/nsis

set +eu
