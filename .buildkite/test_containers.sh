#!/bin/bash

# Runs "make test" in each container directory

set -o errexit
set -o pipefail
set -o nounset
set -x

for dir in containers/*
    do pushd $dir
    time make test
    popd
done
