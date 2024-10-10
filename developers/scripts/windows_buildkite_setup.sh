#!/bin/bash

set -eu -o pipefail
set -x

if [ -z ${BUILDKITE_AGENT_TOKEN:-""} ]; then
  echo "Please set environment variable BUILDKITE_AGENT_TOKEN"
  exit 101
fi

mkcert -install

# Set *global* line endings (not user) because the buildkite-agent may not be running as testbot user
perl -pi -e 's/autocrlf = true/autocrlf = false\n\teol = lf/' "/c/Program Files/Git/etc/gitconfig"

# Install Ubuntu from Microsoft store
# Then wsl --set-default Ubuntu

# Install buildkite-agent
LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/buildkite/agent/releases/latest)
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
NO_V_VERSION=${LATEST_VERSION#v}
URL="https://github.com/buildkite/agent/releases/download/$LATEST_VERSION/buildkite-agent-windows-amd64-${NO_V_VERSION}.zip"
mkdir -p /c/buildkite-agent/bin && cd /tmp && curl -L -O $URL
cd /c/buildkite-agent && unzip /tmp/buildkite-agent-windows-amd64-${NO_V_VERSION}.zip
perl -pi.bak -e 's/# tags="key1=val2,key2=val2"/tags="os=windows,architecture=amd64,osvariant=windows10pro,dockertype=dockerforwindows"/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e 's/^build-path=.*$/build-path=C:\\Users\\testbot\\tmp\\buildkite/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e 's/^build-path=.*$/build-path=C:\\Users\\testbot\\tmp\\buildkite/' /c/buildkite-agent/buildkite-agent.cfg
perl -pi.bak -e "s/^token=.*\$/token=${BUILDKITE_AGENT_TOKEN}/" /c/buildkite-agent/buildkite-agent.cfg

mv /c/buildkite-agent/buildkite-agent /c/buildkite-agent/bin

nssm.exe stop buildkite-agent || true
nssm.exe remove buildkite-agent confirm || true
nssm.exe install buildkite-agent "C:\buildkite-agent\bin\buildkite-agent.exe" "start" || true
nssm.exe set buildkite-agent AppStdout "C:\buildkite-agent\buildkite-agent.log"
nssm.exe set buildkite-agent AppStderr "C:\buildkite-agent\buildkite-agent.lwinpty docker run -it -p 80 busybox:stable log"
nssm.exe start buildkite-agent || true
nssm.exe status buildkite-agent || true

# Get firewall set up with a single run
winpty docker run -it --rm -p 80 busybox:stable ls

bash "/c/Program Files/ddev/windows_ddev_nfs_setup.sh"
