#!/bin/bash

set -eu -o pipefail
set -x

if [ -z ${BUILDKITE_AGENT_TOKEN:-""} ]; then
  echo "Please set environment variable BUILDKITE_AGENT_TOKEN"
  exit 101
fi

# Update kernel for WSL2
cd /tmp && curl -O -sSL https://wslstorestorage.blob.core.windows.net/wslblob/wsl_update_x64.msi && start wsl_update_x64.msi

wsl --set-default-version 2

# Install required items using chocolatey
choco upgrade -y git mysql-cli golang make docker-desktop nssm GoogleChrome zip jq composer cmder netcat ddev mkcert

mkcert -install
git config --global core.autocrlf false
git config --global core.eol lf

# Install Ubuntu from Microsoft store
# Then wsl --set-default Ubuntu

# install bats
cd /tmp && curl -L -O https://github.com/bats-core/bats-core/archive/v1.2.0.tar.gz && tar -zxf v1.2.0.tar.gz && cd bats-core-1.2.0 && ./install.sh /usr/local

# Install buildkite-agent
LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/buildkite/agent/releases/latest)
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
NO_V_VERSION=${LATEST_VERSION#v}
URL="https://github.com/buildkite/agent/releases/download/$LATEST_VERSION/buildkite-agent-windows-amd64-${NO_V_VERSION}.zip"
mkdir -p /c/buildkite-agent/bin && cd /tmp && curl -L -O $URL
cd /c/buildkite-agent && unzip /tmp/buildkite-agent-windows-amd64-${NO_V_VERSION}.zip
perl -pi.bak -e 's/# tags="key1=val2,key2=val2"/tags="os=windows,osvariant=windows10pro,dockertype=dockerforwindows"/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e 's/^build-path=.*$/build-path=C:\\Users\\testbot\\tmp\\buildkite/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e 's/^build-path=.*$/build-path=C:\\Users\\testbot\\tmp\\buildkite/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e "s/^token=.*\$/token=${BUILDKITE_AGENT_TOKEN}/" /c/buildkite-agent/buildkite-agent.cfg

mv /c/buildkite-agent/buildkite-agent /c/buildkite-agent/bin

nssm.exe stop buildkite-agent || true
nssm.exe remove buildkite-agent confirm || true
nssm.exe install buildkite-agent "C:\buildkite-agent\bin\buildkite-agent.exe" "start" || true
nssm.exe set buildkite-agent AppStdout "C:\buildkite-agent\buildkite-agent.log"
nssm.exe set buildkite-agent AppStderr "C:\buildkite-agent\buildkite-agent.lwinpty docker run -it -p 80 busybox lsog"
nssm.exe start buildkite-agent || true
nssm.exe status buildkite-agent || true

# Get firewall set up with a single run
winpty docker run -it --rm -p 80 busybox ls

bash /c/Program\ Files/ddev/windows_ddev_nfs_setup.sh
