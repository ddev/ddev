# Command name changes for MariaDB 11+

if command -v mariadbd >/dev/null 2>&1; then
    mysqld() { mariadbd "$@"; }
fi

if command -v mariadb-convert-table-format >/dev/null 2>&1; then
    mysql_convert_table_format() { mariadb-convert-table-format "$@"; }
fi

if command -v mariadb-find-rows >/dev/null 2>&1; then
    mysql_find_rows() { mariadb-find-rows "$@"; }
fi

if command -v mariadb-fix-extensions >/dev/null 2>&1; then
    mysql_fix_extensions() { mariadb-fix-extensions "$@"; }
fi

if command -v mariadb-install-db >/dev/null 2>&1; then
    mysql_install_db() { mariadb-install-db "$@"; }
fi

if command -v mariadb-plugin >/dev/null 2>&1; then
    mysql_plugin() { mariadb-plugin "$@"; }
fi

if command -v mariadb-secure-installation >/dev/null 2>&1; then
    mysql_secure_installation() { mariadb-secure-installation "$@"; }
fi

if command -v mariadb-setpermission >/dev/null 2>&1; then
    mysql_setpermission() { mariadb-setpermission "$@"; }
fi

if command -v mariadb-tzinfo-to-sql >/dev/null 2>&1; then
    mysql_tzinfo_to_sql() { mariadb-tzinfo-to-sql "$@"; }
fi

if command -v mariadb-upgrade >/dev/null 2>&1; then
    mysql_upgrade() { mariadb-upgrade "$@"; }
fi

if command -v mariadb-waitpid >/dev/null 2>&1; then
    mysql_waitpid() { mariadb-waitpid "$@"; }
fi

if command -v mariadb-access >/dev/null 2>&1; then
    mysqlaccess() { mariadb-access "$@"; }
fi

if command -v mariadb-admin >/dev/null 2>&1; then
    mysqladmin() { mariadb-admin "$@"; }
fi

if command -v mariadb-check >/dev/null 2>&1; then
    mysqlanalyze() { mariadb-check "$@"; }
    mysqlcheck() { mariadb-check "$@"; }
    mysqloptimize() { mariadb-check "$@"; }
    mysqlrepair() { mariadb-check "$@"; }
fi

if command -v mariadb-binlog >/dev/null 2>&1; then
    mysqlbinlog() { mariadb-binlog "$@"; }
fi

if command -v mariadbd-multi >/dev/null 2>&1; then
    mysqld_multi() { mariadbd-multi "$@"; }
fi

if command -v mariadbd-safe >/dev/null 2>&1; then
    mysqld_safe() { mariadbd-safe "$@"; }
fi

if command -v mariadbd-safe-helper >/dev/null 2>&1; then
    mysqld_safe_helper() { mariadbd-safe-helper "$@"; }
fi

if command -v mariadb-dump >/dev/null 2>&1; then
    mysqldump() { mariadb-dump "$@"; }
fi

if command -v mariadb-dumpslow >/dev/null 2>&1; then
    mysqldumpslow() { mariadb-dumpslow "$@"; }
fi

if command -v mariadb-hotcopy >/dev/null 2>&1; then
    mysqlhotcopy() { mariadb-hotcopy "$@"; }
fi

if command -v mariadb-import >/dev/null 2>&1; then
    mysqlimport() { mariadb-import "$@"; }
fi

if command -v mariadb-report >/dev/null 2>&1; then
    mysqlreport() { mariadb-report "$@"; }
fi

if command -v mariadb-show >/dev/null 2>&1; then
    mysqlshow() { mariadb-show "$@"; }
fi

if command -v mariadb-slap >/dev/null 2>&1; then
    mysqlslap() { mariadb-slap "$@"; }
fi
