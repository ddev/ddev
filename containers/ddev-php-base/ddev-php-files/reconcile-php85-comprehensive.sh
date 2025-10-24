#!/usr/bin/env bash
set -euo pipefail

# Comprehensive PHP 8.5 Reconciliation Script
#
# Strategy:
# 1. Keep current DDEV 8.5 configs (all values are correct!)
# 2. Update PHP 8.4→8.5 documentation/comment changes
# 3. Add new PHP 8.5 directives

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_BASE="${SCRIPT_DIR}/etc/php/8.5"

echo "=== PHP 8.5 Comprehensive Reconciliation ==="
echo "Target: $TARGET_BASE"
echo ""

# Backup
BACKUP_DIR="${SCRIPT_DIR}/php85-backup-comprehensive-$(date +%Y%m%d-%H%M%S)"
echo "Creating backup at: $BACKUP_DIR"
cp -r "$TARGET_BASE" "$BACKUP_DIR"

#
# Function to apply PHP 8.4→8.5 changes to php.ini files
#
update_php_ini_for_85() {
    local file=$1
    local is_cli=$2

    echo "  Processing: $file"

    # 1. Remove register_argc_argv from Quick Reference section (lines ~127-130)
    sed -i.bak '/^; register_argc_argv$/,/^;   Production Value: Off$/d' "$file"

    # 2. Add mysqlnd.collect_memory_statistics to Quick Reference (after max_input_time section)
    sed -i.bak '/^;   Production Value: 60 (60 seconds)$/a\
\
; mysqlnd.collect_memory_statistics\
;   Default Value: Off\
;   Development Value: On\
;   Production Value: Off
' "$file"

    # 3. Remove disable_classes directive and comments (removed in PHP 8.5)
    sed -i.bak '/^; This directive allows you to disable certain classes\.$/,/^disable_classes =$/d' "$file"

    # 4. Add max_memory_limit directive (after memory_limit)
    # Check if it already exists
    if ! grep -q "^max_memory_limit" "$file"; then
        sed -i.bak '/^memory_limit = /a\
max_memory_limit = -1
' "$file"
    fi

    # 5. Remove report_memleaks directive and comments (removed in PHP 8.5)
    sed -i.bak '/^; If this parameter is set to Off, then memory leaks/,/^report_memleaks = On$/d' "$file"

    # 6. Add fatal_error_backtraces comment (new in 8.5, before Data Handling section)
    if ! grep -q "fatal_error_backtraces" "$file"; then
        sed -i.bak '/^; Production value: 0$/a\
\
; This directive controls whether PHP will output the backtrace of fatal errors.\
; Default Value: On\
; Development Value: On\
; Production Value: On\
;fatal_error_backtraces = On
' "$file"
    fi

    # 7. Update register_argc_argv comments (deprecated in 8.5)
    sed -i.bak 's/; a script is executed\. For performance reasons, this feature should be disabled/; a script is executed. For security reasons, this feature should be disabled/' "$file"
    sed -i.bak 's/; on production servers\./; for non-CLI SAPIs./' "$file"
    sed -i.bak 's/; Note: This directive is hardcoded to On for the CLI SAPI/; Note: This directive is ignored for the CLI SAPI/' "$file"
    sed -i.bak '/^; Note: This directive is ignored for the CLI SAPI$/a\
; This directive is deprecated.
' "$file"

    # 8. Add fastcgi.script_path_encoded comment (new in 8.5)
    if ! grep -q "fastcgi.script_path_encoded" "$file"; then
        sed -i.bak '/^;fastcgi.impersonate = 1$/a\
\
; Prevent decoding of SCRIPT_FILENAME when using Apache ProxyPass or\
; ProxyPassMatch. This should only be used if script file paths are already\
; stored in an encoded format on the file system.\
; Default is 0.\
;fastcgi.script_path_encoded = 1
' "$file"
    fi

    rm -f "$file.bak"
}

echo ""
echo ">>> Updating CLI php.ini..."
update_php_ini_for_85 "$TARGET_BASE/cli/php.ini" "cli"

echo ""
echo ">>> Updating FPM php.ini..."
update_php_ini_for_85 "$TARGET_BASE/fpm/php.ini" "fpm"

echo ""
echo ">>> FPM php-fpm.conf and www.conf..."
echo "  No changes needed - already correct for PHP 8.5"

echo ""
echo "=== Verification ==="
echo "Verifying critical DDEV settings preserved:"
echo ""

verify_setting() {
    local file=$1
    local pattern=$2
    local description=$3

    if grep -q "$pattern" "$file"; then
        echo "  ✓ $description"
    else
        echo "  ✗ MISSING: $description"
        return 1
    fi
}

echo "php-fpm.conf:"
verify_setting "$TARGET_BASE/fpm/php-fpm.conf" "^daemonize = no$" "daemonize = no"
verify_setting "$TARGET_BASE/fpm/php-fpm.conf" "^error_log = /dev/stdout$" "error_log = /dev/stdout"

echo ""
echo "FPM php.ini:"
verify_setting "$TARGET_BASE/fpm/php.ini" "^memory_limit = 1024M$" "memory_limit = 1024M"
verify_setting "$TARGET_BASE/fpm/php.ini" "^max_execution_time = 600$" "max_execution_time = 600"
verify_setting "$TARGET_BASE/fpm/php.ini" "^date.timezone = UTC$" "date.timezone = UTC"
verify_setting "$TARGET_BASE/fpm/php.ini" "^mail.add_x_header = On$" "mail.add_x_header = On"
verify_setting "$TARGET_BASE/fpm/php.ini" "^mysqli.reconnect = Off$" "mysqli.reconnect = Off"
verify_setting "$TARGET_BASE/fpm/php.ini" "^cgi.fix_pathinfo=1$" "cgi.fix_pathinfo=1 (FPM)"
verify_setting "$TARGET_BASE/fpm/php.ini" "^opcache.validate_timestamps=1$" "opcache.validate_timestamps=1"
verify_setting "$TARGET_BASE/fpm/php.ini" "^opcache.revalidate_freq=0$" "opcache.revalidate_freq=0"

echo ""
echo "CLI php.ini:"
verify_setting "$TARGET_BASE/cli/php.ini" "^cgi.fix_pathinfo=0$" "cgi.fix_pathinfo=0 (CLI)"

echo ""
echo "New PHP 8.5 additions:"
verify_setting "$TARGET_BASE/cli/php.ini" "^max_memory_limit = -1$" "max_memory_limit = -1 (CLI)"
verify_setting "$TARGET_BASE/fpm/php.ini" "^max_memory_limit = -1$" "max_memory_limit = -1 (FPM)"

echo ""
echo "=== Reconciliation Complete ==="
echo ""
echo "Changes made:"
echo "- Updated PHP 8.4→8.5 documentation/comments"
echo "- Removed deprecated directives (disable_classes, report_memleaks)"
echo "- Added new PHP 8.5 directives (max_memory_limit)"
echo "- Updated Quick Reference section"
echo "- ALL DDEV customizations preserved"
echo ""
echo "Backup: $BACKUP_DIR"
echo ""
echo "Next: git diff etc/php/8.5/ | less"