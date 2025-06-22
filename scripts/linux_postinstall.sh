#!/bin/sh
if ! grep -qEi "(Microsoft|WSL)" /proc/version; then
  rm -f /usr/bin/ddev_hostname.exe
fi
