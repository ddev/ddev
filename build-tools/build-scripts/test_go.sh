#!/bin/sh
#
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

if [ -z "${OS}" ]; then
    echo "OS must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOOS="${OS}"

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

echo "Running tests:"
go test -i -installsuffix "static" ${TARGETS}
go test -installsuffix "static" ${TARGETS}
echo

