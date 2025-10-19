#!/bin/bash
set -euo pipefail

# Script to extract default PHP configuration files from deb.sury.org
# This is used to reconcile DDEV's customizations with upstream PHP defaults
# when adding support for a new PHP major version.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-${SCRIPT_DIR}/php-configs}"
OLD_VERSION="${OLD_VERSION:-8.4}"
NEW_VERSION="${NEW_VERSION:-8.5}"
# Auto-detect DDEV root from script location (script is in ddev-php-base/ddev-php-files/)
DDEV_ROOT="${DDEV_ROOT:-$(cd "${SCRIPT_DIR}/../../.." && pwd)}"
PHP_PACKAGES_YAML="${PHP_PACKAGES_YAML:-${DDEV_ROOT}/containers/ddev-php-base/generic-files/etc/php-packages.yaml}"

echo "=== PHP Configuration Extractor ==="
echo "Old PHP Version: ${OLD_VERSION}"
echo "New PHP Version: ${NEW_VERSION}"
echo "Output Directory: ${OUTPUT_DIR}"
echo "PHP Packages YAML: ${PHP_PACKAGES_YAML}"
echo ""

# Verify php-packages.yaml exists
if [ ! -f "${PHP_PACKAGES_YAML}" ]; then
    echo "ERROR: php-packages.yaml not found at ${PHP_PACKAGES_YAML}"
    echo "Please set DDEV_ROOT or PHP_PACKAGES_YAML environment variable"
    exit 1
fi

# Create output directories
mkdir -p "${OUTPUT_DIR}/upstream-php${OLD_VERSION}"
mkdir -p "${OUTPUT_DIR}/upstream-php${NEW_VERSION}"
mkdir -p "${OUTPUT_DIR}/diffs"

# Function to parse PHP packages from YAML file
parse_php_packages() {
    local version=$1
    local php_version_key="php${version//./}"  # Convert 8.4 to php84

    # Parse YAML to extract amd64 package list for the given version
    # Uses yq to reliably parse YAML structure
    yq eval ".${php_version_key}.amd64[]" "${PHP_PACKAGES_YAML}" 2>/dev/null
}

# Get package lists for both versions
echo ">>> Parsing package lists from ${PHP_PACKAGES_YAML}..."
OLD_PACKAGES=$(parse_php_packages "${OLD_VERSION}")
NEW_PACKAGES=$(parse_php_packages "${NEW_VERSION}")

if [ -z "${OLD_PACKAGES}" ]; then
    echo "ERROR: Could not parse packages for PHP ${OLD_VERSION}"
    exit 1
fi

echo "    Found $(echo "${OLD_PACKAGES}" | wc -l | tr -d ' ') packages for PHP ${OLD_VERSION}"
if [ -n "${NEW_PACKAGES}" ]; then
    echo "    Found $(echo "${NEW_PACKAGES}" | wc -l | tr -d ' ') packages for PHP ${NEW_VERSION}"
else
    echo "    No specific package list for PHP ${NEW_VERSION}, will use PHP ${OLD_VERSION} list"
    NEW_PACKAGES="${OLD_PACKAGES}"
fi
echo ""

# Function to extract PHP configs for a specific version
extract_php_configs() {
    local version=$1
    local output_subdir=$2
    local packages=$3

    echo ">>> Extracting PHP ${version} default configurations..."

    # Create a temporary container name
    local container_name="ddev-php${version}-extract-$$"

    # Create and run container in detached mode
    docker run -d --name "${container_name}" -e PACKAGES="${packages}" debian:bookworm bash -c "
        set -e

        # Install prerequisites
        apt-get update -qq
        apt-get install -y -qq lsb-release ca-certificates curl wget gnupg2

        # Add deb.sury.org repository
        curl -sSLo /tmp/debsuryorg-archive-keyring.deb https://packages.sury.org/debsuryorg-archive-keyring.deb
        dpkg -i /tmp/debsuryorg-archive-keyring.deb
        echo 'deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ bookworm main' > /etc/apt/sources.list.d/php.list

        # Update package lists
        apt-get update -qq

        # Install PHP packages
        echo \"    Installing PHP ${version} packages...\"
        for pkg in \$PACKAGES; do
            DEBIAN_FRONTEND=noninteractive apt-get install -y -qq php${version}-\${pkg} 2>/dev/null && \
                echo \"      ✓ php${version}-\${pkg}\" || \
                echo \"      ✗ php${version}-\${pkg} (not available)\"
        done

        # Keep container running
        sleep 60
    " > /dev/null 2>&1

    # Wait for packages to install (check for actual config files)
    echo "    Waiting for PHP ${version} installation to complete..."
    local retries=60
    while [ $retries -gt 0 ]; do
        # Check if cli/php.ini and fpm/php.ini exist (core config files)
        if docker exec "${container_name}" test -f "/etc/php/${version}/cli/php.ini" 2>/dev/null && \
           docker exec "${container_name}" test -f "/etc/php/${version}/fpm/php.ini" 2>/dev/null; then
            break
        fi
        sleep 2
        retries=$((retries - 1))
    done

    if [ $retries -eq 0 ]; then
        echo "    ERROR: Timeout waiting for PHP ${version} installation"
        docker rm -f "${container_name}" > /dev/null 2>&1
        return 1
    fi

    # Copy the PHP config directory from container
    echo "    Copying PHP ${version} configs from container..."
    mkdir -p "${OUTPUT_DIR}/${output_subdir}"
    docker cp "${container_name}:/etc/php/${version}" "${OUTPUT_DIR}/${output_subdir}/php-${version}-tmp"

    # Move to expected location
    mkdir -p "${OUTPUT_DIR}/${output_subdir}/php"
    mv "${OUTPUT_DIR}/${output_subdir}/php-${version}-tmp" "${OUTPUT_DIR}/${output_subdir}/php/${version}"

    # Clean up container
    docker rm -f "${container_name}" > /dev/null 2>&1

    echo "    ✓ PHP ${version} configs extracted to ${OUTPUT_DIR}/${output_subdir}/php/${version}/"
    echo ""
}

# Function to compare configurations
compare_configs() {
    echo ">>> Generating comparison diffs..."

    local old_base="${OUTPUT_DIR}/upstream-php${OLD_VERSION}/php/${OLD_VERSION}"
    local new_base="${OUTPUT_DIR}/upstream-php${NEW_VERSION}/php/${NEW_VERSION}"
    local ddev_old_base="${SCRIPT_DIR}/etc/php/${OLD_VERSION}"

    # Check if DDEV configs exist
    if [ ! -d "${ddev_old_base}" ]; then
        echo "    Warning: DDEV PHP ${OLD_VERSION} configs not found at ${ddev_old_base}"
        echo "    Skipping DDEV customization comparison"
        ddev_old_base=""
    fi

    # Files to compare
    local files_to_compare=(
        "apache2/php.ini"
        "cli/php.ini"
        "fpm/php.ini"
        "fpm/php-fpm.conf"
        "fpm/pool.d/www.conf"
    )

    for config_file in "${files_to_compare[@]}"; do
        local safe_name=$(echo "$config_file" | tr '/' '_')

        # 1. Compare upstream OLD vs NEW (what changed in PHP itself)
        if [ -f "${old_base}/${config_file}" ] && [ -f "${new_base}/${config_file}" ]; then
            echo "    Comparing upstream PHP ${OLD_VERSION} vs ${NEW_VERSION}: ${config_file}"
            diff -u "${old_base}/${config_file}" "${new_base}/${config_file}" \
                > "${OUTPUT_DIR}/diffs/upstream-${OLD_VERSION}-to-${NEW_VERSION}-${safe_name}.diff" || true
        fi

        # 2. Compare DDEV OLD vs upstream OLD (what DDEV customized)
        if [ -n "${ddev_old_base}" ] && [ -f "${ddev_old_base}/${config_file}" ] && [ -f "${old_base}/${config_file}" ]; then
            echo "    Comparing upstream vs DDEV PHP ${OLD_VERSION}: ${config_file}"
            diff -u "${old_base}/${config_file}" "${ddev_old_base}/${config_file}" \
                > "${OUTPUT_DIR}/diffs/ddev-customizations-${OLD_VERSION}-${safe_name}.diff" || true
        fi
    done

    echo "    ✓ Diffs saved to ${OUTPUT_DIR}/diffs/"
    echo ""
}

# Function to copy missing extension configs from old to new version
copy_missing_extension_configs() {
    echo ">>> Checking for missing extension configs in PHP ${NEW_VERSION}..."

    local old_mods="${OUTPUT_DIR}/upstream-php${OLD_VERSION}/php/${OLD_VERSION}/mods-available"
    local new_mods="${OUTPUT_DIR}/upstream-php${NEW_VERSION}/php/${NEW_VERSION}/mods-available"
    local missing_file="${OUTPUT_DIR}/MISSING_EXTENSIONS.txt"

    if [ ! -d "${old_mods}" ] || [ ! -d "${new_mods}" ]; then
        echo "    Warning: Could not find mods-available directories"
        return
    fi

    # Create list of missing extensions
    > "${missing_file}"

    # Get list of extensions that are typically installed as separate packages
    # (excluding core packages like cli, common, fpm)
    local ddev_extensions=$(echo "${OLD_PACKAGES}" | grep -v -E '^(cli|common|fpm)$')

    for ext in ${ddev_extensions}; do
        # Check if extension config exists in new version
        local found=false
        for ext_file in "${new_mods}/${ext}.ini" "${new_mods}/"*"${ext}"*.ini; do
            if [ -f "${ext_file}" ]; then
                found=true
                break
            fi
        done

        if [ "${found}" = false ]; then
            echo "    ✗ ${ext} - not available in PHP ${NEW_VERSION}"
            echo "${ext}" >> "${missing_file}"

            # Copy from old version if it exists
            for old_ext_file in "${old_mods}/${ext}.ini" "${old_mods}/"*"${ext}"*.ini; do
                if [ -f "${old_ext_file}" ]; then
                    local basename=$(basename "${old_ext_file}")
                    cp "${old_ext_file}" "${new_mods}/${basename}"
                    echo "      → Copied from PHP ${OLD_VERSION}: ${basename}"
                fi
            done
        else
            echo "    ✓ ${ext} - available in PHP ${NEW_VERSION}"
        fi
    done

    if [ -s "${missing_file}" ]; then
        echo ""
        echo "    Note: Extension configs for unavailable packages were copied from PHP ${OLD_VERSION}"
        echo "    See ${missing_file} for the list"
    fi
    echo ""
}

# Function to generate report
generate_report() {
    local report_file="${OUTPUT_DIR}/RECONCILIATION_REPORT.txt"
    local missing_file="${OUTPUT_DIR}/MISSING_EXTENSIONS.txt"

    echo ">>> Generating reconciliation report..."

    cat > "${report_file}" << 'EOF'
================================================================================
PHP Configuration Reconciliation Report
================================================================================

This report helps reconcile DDEV's PHP configurations when adding a new major
PHP version. Follow these steps:

STEP 1: Review Upstream Changes (PHP version differences)
--------------------------------------------------------------------------------
Check the diffs/upstream-*.diff files to see what changed in PHP itself
between versions. These are changes made by the PHP project.

Example:
    less diffs/upstream-8.4-to-8.5-cli_php.ini.diff

STEP 2: Review DDEV Customizations
--------------------------------------------------------------------------------
Check the diffs/ddev-customizations-*.diff files to see what DDEV changed
from the upstream defaults. These customizations need to be preserved.

Common DDEV customizations:
- max_execution_time = 600 (increased from default 30)
- memory_limit = 1024M (increased from default 128M)
- display_errors = On (enabled for development)
- post_max_size = 100M (increased from default 8M)
- upload_max_filesize = 100M (increased from default 2M)
- sendmail_path = mailpit configuration
- expose_php = Off (security)
- max_input_vars = 5000 (increased from default 1000)
- decorate_workers_output = no (in www.conf)

STEP 3: Apply Reconciliation
--------------------------------------------------------------------------------
For the new PHP version configs:
1. Start with the upstream NEW version defaults
2. Apply all DDEV customizations from STEP 2
3. Verify that upstream NEW changes from STEP 1 are preserved

STEP 4: Verify Extension Configs
--------------------------------------------------------------------------------
DDEV maintains custom configs for these extensions in mods-available/:
- assert.ini
- blackfire.ini
- xdebug.ini
- xhprof.ini

Copy these from the previous version and verify they work with the new version.
All other extension configs are managed by the package manager.

STEP 5: Test
--------------------------------------------------------------------------------
Build the container and test:
1. Verify PHP version: php -v
2. Check loaded extensions: php -m
3. Verify configuration: php -i | grep -E '(max_execution_time|memory_limit)'
4. Test with a DDEV project

FILES EXTRACTED:
================================================================================
EOF

    echo "Upstream PHP ${OLD_VERSION}: ${OUTPUT_DIR}/upstream-php${OLD_VERSION}/php/${OLD_VERSION}/" >> "${report_file}"
    echo "Upstream PHP ${NEW_VERSION}: ${OUTPUT_DIR}/upstream-php${NEW_VERSION}/php/${NEW_VERSION}/" >> "${report_file}"
    echo "Comparison diffs: ${OUTPUT_DIR}/diffs/" >> "${report_file}"
    echo "" >> "${report_file}"

    # Add missing extensions section if applicable
    if [ -f "${missing_file}" ] && [ -s "${missing_file}" ]; then
        echo "MISSING EXTENSIONS IN PHP ${NEW_VERSION}:" >> "${report_file}"
        echo "================================================================================\n" >> "${report_file}"
        echo "The following extensions are not yet available for PHP ${NEW_VERSION} on" >> "${report_file}"
        echo "deb.sury.org. Their configuration files were copied from PHP ${OLD_VERSION}:" >> "${report_file}"
        echo "" >> "${report_file}"
        cat "${missing_file}" | while read ext; do
            echo "  - ${ext}" >> "${report_file}"
        done
        echo "" >> "${report_file}"
        echo "Note: When these extensions become available, you should:" >> "${report_file}"
        echo "1. Install the package (e.g., apt install php${NEW_VERSION}-xdebug)" >> "${report_file}"
        echo "2. Get the new default .ini file from /etc/php/${NEW_VERSION}/mods-available/" >> "${report_file}"
        echo "3. Apply DDEV's customizations if needed (check assert.ini, xdebug.ini, etc.)" >> "${report_file}"
        echo "" >> "${report_file}"
    fi

    echo "    ✓ Report saved to ${report_file}"
    echo ""

    # Display the report
    cat "${report_file}"
}

# Main execution
main() {
    echo "Starting extraction process..."
    echo ""

    # Extract PHP configs for both versions
    extract_php_configs "${OLD_VERSION}" "upstream-php${OLD_VERSION}" "${OLD_PACKAGES}"
    extract_php_configs "${NEW_VERSION}" "upstream-php${NEW_VERSION}" "${NEW_PACKAGES}"

    # Copy missing extension configs from old to new
    copy_missing_extension_configs

    # Generate comparisons
    compare_configs

    # Generate report
    generate_report

    echo "=== Extraction Complete ==="
    echo ""
    echo "Next steps:"
    echo "1. Review the report: ${OUTPUT_DIR}/RECONCILIATION_REPORT.txt"
    echo "2. Check the diffs in: ${OUTPUT_DIR}/diffs/"
    echo "3. Apply necessary changes to: containers/ddev-php-base/ddev-php-files/etc/php/${NEW_VERSION}/"
    echo ""
}

# Run main function
main