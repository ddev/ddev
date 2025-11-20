#!/usr/bin/env bash

# This script installs MySQL compatibility wrappers for MariaDB commands.
# Two packages provide these commands:
# - mariadb-server-compat: server commands (mysqld, mysqld_multi, etc.)
# - mariadb-client-compat: client commands (mysql, mysqladmin, etc.)
# These packages show deprecation warnings.
#
# The script creates wrappers for all MariaDB commands and points them to the
# correct binaries, avoiding deprecation warnings such as:
# "mysql: Deprecated program name. It will be removed in a future release, use '/usr/bin/mariadb' instead."

set -eu -o pipefail

DDEV_DATABASE_FAMILY=${DDEV_DATABASE%:*}
ADD_WRAPPER=true
# Don't add wrappers if using MySQL
if [ "${DDEV_DATABASE_FAMILY}" = "mysql" ]; then
  ADD_WRAPPER=false
fi

add_or_remove_mariadb_wrapper() {
  local mysql_binary="$1"
  local mariadb_binary="$2"
  local script_path="/usr/local/bin/$mysql_binary"

  # Remove mode: delete wrapper if it exists and is our script
  if [ "${ADD_WRAPPER}" = "false" ]; then
    # Check if it's our wrapper by looking for #ddev-generated marker
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # Install mode: create wrapper if needed

  # Only proceed if target command exists (e.g., mariadb, mariadbd)
  if ! command -v "$mariadb_binary" >/dev/null 2>&1; then
    # If target doesn't exist, remove our wrapper if it exists
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # If the mysql binary exists natively elsewhere, remove our wrapper
  if [ "$(which -a "$mysql_binary" 2>/dev/null | wc -l)" -gt 1 ]; then
    if [ -x "$script_path" ] && head -n 3 "$script_path" 2>/dev/null | grep -q "#ddev-generated"; then
      rm -f "$script_path"
    fi
    return
  fi

  # Don't overwrite if executable file already exists
  if [ -x "$script_path" ]; then
    return
  fi

  cat >"$script_path" <<EOF
#!/usr/bin/env bash
#ddev-generated
exec -a $mariadb_binary $mariadb_binary "\$@"
EOF

  chmod +x "$script_path"
}

add_or_remove_mariadb_wrapper mariabackup mariadb-backup
add_or_remove_mariadb_wrapper mysql mariadb
add_or_remove_mariadb_wrapper mysql_convert_table_format mariadb-convert-table-format
add_or_remove_mariadb_wrapper mysql_find_rows mariadb-find-rows
add_or_remove_mariadb_wrapper mysql_fix_extensions mariadb-fix-extensions
add_or_remove_mariadb_wrapper mysql_install_db mariadb-install-db
add_or_remove_mariadb_wrapper mysql_plugin mariadb-plugin
add_or_remove_mariadb_wrapper mysql_secure_installation mariadb-secure-installation
add_or_remove_mariadb_wrapper mysql_setpermission mariadb-setpermission
add_or_remove_mariadb_wrapper mysql_tzinfo_to_sql mariadb-tzinfo-to-sql
add_or_remove_mariadb_wrapper mysql_upgrade mariadb-upgrade
add_or_remove_mariadb_wrapper mysql_waitpid mariadb-waitpid
add_or_remove_mariadb_wrapper mysqlaccess mariadb-access
add_or_remove_mariadb_wrapper mysqladmin mariadb-admin
add_or_remove_mariadb_wrapper mysqlanalyze mariadb-check
add_or_remove_mariadb_wrapper mysqlbinlog mariadb-binlog
add_or_remove_mariadb_wrapper mysqlcheck mariadb-check
add_or_remove_mariadb_wrapper mysqld mariadbd
add_or_remove_mariadb_wrapper mysqld_multi mariadbd-multi
add_or_remove_mariadb_wrapper mysqld_safe mariadbd-safe
add_or_remove_mariadb_wrapper mysqld_safe_helper mariadbd-safe-helper
add_or_remove_mariadb_wrapper mysqldump mariadb-dump
add_or_remove_mariadb_wrapper mysqldumpslow mariadb-dumpslow
add_or_remove_mariadb_wrapper mysqlhotcopy mariadb-hotcopy
add_or_remove_mariadb_wrapper mysqlimport mariadb-import
add_or_remove_mariadb_wrapper mysqloptimize mariadb-check
add_or_remove_mariadb_wrapper mysqlreport mariadb-report
add_or_remove_mariadb_wrapper mysqlrepair mariadb-check
add_or_remove_mariadb_wrapper mysqlshow mariadb-show
add_or_remove_mariadb_wrapper mysqlslap mariadb-slap

if [ "${ADD_WRAPPER}" = "true" ]; then
  echo "MariaDB compatibility wrappers installed, using ${DDEV_DATABASE}"
else
  echo "MariaDB compatibility wrappers removed, using ${DDEV_DATABASE}"
fi
