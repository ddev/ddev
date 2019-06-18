#!/bin/bash

set -eu

os=$(go env GOOS)

if ! command -v ngrok >/dev/null; then
    case $os in
    darwin)
        brew cask install ngrok
        ;;
    windows)
        choco install -y ngrok
        ;;
    linux)
        curl -sSL --fail -o /tmp/ngrok.zip https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-linux-amd64.zip && sudo unzip -d /usr/local/bin /tmp/ngrok.zip
        ;;
    esac
fi
