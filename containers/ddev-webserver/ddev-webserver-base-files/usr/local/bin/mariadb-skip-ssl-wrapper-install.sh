#!/usr/bin/env bash

# This script creates wrappers for MariaDB commands that add the
# --skip-ssl-verify-server-cert flag to disable SSL certificate verification.
# Each wrapper is only created if the command supports the flag.

# Enabling One-Way TLS for MariaDB Clients:
# One-way TLS means that only the server provides a private key and an X509 certificate.
# Starting from MariaDB 11.4 (Connector/C version 3.4) this mode is enabled by default.
# See https://mariadb.com/docs/server/security/securing-mariadb/encryption/data-in-transit-encryption/securing-connections-for-client-and-server#enabling-tls-for-mariadb-clients

set -eu -o pipefail

DDEV_DATABASE_FAMILY=${DDEV_DATABASE%:*}
ADD_WRAPPER=true

# Don't add wrappers if not using MariaDB
if [ "${DDEV_DATABASE_FAMILY}" != "mariadb" ]; then
  ADD_WRAPPER=false
fi

# Don't add wrappers if using MariaDB below 11.x (10.11 client is installed instead, which doesn't enforce SSL)
MARIADB_VERSION=${DDEV_DATABASE#*:}
if [ "${MARIADB_VERSION%%.*}" -lt 11 ]; then
  ADD_WRAPPER=false
fi

add_or_remove_ssl_wrapper() {
  local mariadb_binary="$1"
  local script_path="/usr/local/bin/${mariadb_binary}"

  # Remove mode: delete wrapper if it exists and is our script
  if [ "${ADD_WRAPPER}" = "false" ]; then
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # Install mode: create wrapper if needed

  # Find the real binary path (not in /usr/local/bin to avoid wrapping ourselves)
  local real_binary_path=""
  while IFS= read -r path; do
    if [[ "$path" != "/usr/local/bin/${mariadb_binary}" ]]; then
      real_binary_path="$path"
      break
    fi
  done < <(which -a "$mariadb_binary" 2>/dev/null)

  # If no real binary found, clean up our wrapper if it exists
  if [ -z "$real_binary_path" ]; then
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # Check if the command supports --skip-ssl-verify-server-cert flag
  if ! "$real_binary_path" --help 2>&1 | grep -qw -- "--skip-ssl-verify-server-cert"; then
    # Flag not supported, remove wrapper if it exists
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # Don't overwrite if executable file already exists and is not our wrapper
  if [ -x "$script_path" ] && ! head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
    return
  fi

  cat >"$script_path" <<EOF
#!/usr/bin/env bash
#ddev-generated
exec -a $mariadb_binary "$real_binary_path" "\$@" --skip-ssl-verify-server-cert
EOF

  chmod +x "$script_path"
  echo "Created SSL wrapper for $mariadb_binary at $script_path"
}

# MariaDB client commands that typically connect to databases and may support SSL options
add_or_remove_ssl_wrapper mariadb
add_or_remove_ssl_wrapper mariadb-admin
add_or_remove_ssl_wrapper mariadb-analyze
add_or_remove_ssl_wrapper mariadb-binlog
add_or_remove_ssl_wrapper mariadb-check
add_or_remove_ssl_wrapper mariadb-dump
add_or_remove_ssl_wrapper mariadb-import
add_or_remove_ssl_wrapper mariadb-optimize
add_or_remove_ssl_wrapper mariadb-repair
add_or_remove_ssl_wrapper mariadb-show
add_or_remove_ssl_wrapper mariadb-slap
add_or_remove_ssl_wrapper mariadbcheck

if [ "${ADD_WRAPPER}" = "true" ]; then
  echo "MariaDB skip SSL verification wrappers installed for ${DDEV_DATABASE}"
else
  echo "MariaDB skip SSL verification wrappers removed for ${DDEV_DATABASE}"
fi
