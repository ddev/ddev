#!/bin/bash

# This script is used to install a matching Node.js version
# using nodejs_version from .ddev/config.yaml in ddev-webserver
# It requires N_PREFIX and N_INSTALL_VERSION to be set (normally in a docker build phase)
# This script is intended to be run in /start.sh without root privileges

set -eu -o pipefail

if [ "${N_PREFIX:-}" = "" ]; then
  echo "This script requires N_PREFIX to be set" && exit 1
fi

if [ "${N_INSTALL_VERSION:-}" = "" ]; then
  echo "This script requires N_INSTALL_VERSION to be set" && exit 2
fi

if [ "${HOSTNAME:-}" = "" ]; then
  echo "This script requires HOSTNAME to be set" && exit 3
fi

if [ ! -d "/mnt/ddev-global-cache/n_prefix/${HOSTNAME}" ]; then
  echo "This script requires the directory /mnt/ddev-global-cache/n_prefix/${HOSTNAME}" && exit 4
fi

system_node_dir="$(dirname "$(which node)")"

if [ ! -w "${system_node_dir}" ]; then
  echo "This script cannot write to the directory ${system_node_dir}" && exit 5
fi

ln -sf "/mnt/ddev-global-cache/n_prefix/${HOSTNAME}" "${N_PREFIX}"

# try normal install that also uses cache
n_install_result=true
timeout 30 n install "${N_INSTALL_VERSION}" 2> >(tee /tmp/n-install-stderr.txt >&2) || n_install_result=false

# try offline install on fail, but only if there is no error for invalid ${N_INSTALL_VERSION}
if [ "${n_install_result}" = "false" ] && ! grep -q "invalid version" /tmp/n-install-stderr.txt; then
  timeout 30 n install "${N_INSTALL_VERSION}" --offline 2> >(tee -a /tmp/n-install-stderr.txt >&2) && n_install_result=true
fi

if [ "${n_install_result}" = "true" ]; then
  # create symlinks on success
  for node_binary in "${N_PREFIX}/bin/"*; do
    if [ -f "${node_binary}" ]; then
      ln -sf "${node_binary}" "${system_node_dir}"
    fi
  done
  ln -sf "${system_node_dir}/node" "${system_node_dir}/nodejs"
  # we don't need this file if everything is fine
  rm -f /tmp/n-install-stderr.txt
else
  # remove the symlink on error so that the system Node.js can be used
  rm -f "${N_PREFIX}"
fi

hash -r
