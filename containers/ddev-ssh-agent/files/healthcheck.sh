#!/bin/bash

# ddev-ssh-agent healthcheck

set -eo pipefail

# Make sure that both socat and ssh-agent are running
killall -0 socat && killall -0 ssh-agent

