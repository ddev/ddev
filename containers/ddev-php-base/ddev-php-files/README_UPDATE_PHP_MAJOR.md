# How to Update PHP Configuration Files for a New Major Version

This document describes the process for adding and reconciling PHP configuration files when DDEV adds support for a new PHP major version (e.g., 8.5, 8.6, etc.).

## Overview

DDEV maintains customized PHP configuration files that differ from upstream defaults in several ways:

- **Increased resource limits** for local development (memory, execution time, upload sizes)
- **Development-friendly settings** (error display, assertions enabled)
- **Mailpit integration** for email testing
- **Custom extension configurations** for debugging tools (xdebug, xhprof, blackfire)

When a new PHP version is released, we need to:

1. Get the upstream default configurations from deb.sury.org
2. Identify what DDEV customized in the previous version
3. Identify what changed in PHP itself between versions
4. Apply DDEV's customizations to the new version while preserving PHP's changes

## Prerequisites

- Docker installed and running
- Write access to the DDEV repository
- Basic understanding of diff/patch operations

## Step-by-Step Process

### 1. Extract Upstream Default Configurations

Use the provided extraction script to get pristine configuration files:

```bash
# From the DDEV repository root
cd containers/ddev-php-base/ddev-php-files

# Set the PHP versions (adjust as needed)
export OLD_VERSION=8.4
export NEW_VERSION=8.5

# Run the extraction script
./extract-php-configs.sh

# Or with custom output directory
OUTPUT_DIR=~/tmp/php-reconciliation ./extract-php-configs.sh
```

The script will:
- Extract default configs for both OLD and NEW PHP versions
- Generate diff files showing changes
- Create a reconciliation report

**Output structure:**
```
php-configs/
├── upstream-php8.4/        # Upstream PHP 8.4 defaults
├── upstream-php8.5/        # Upstream PHP 8.5 defaults
├── diffs/                  # Comparison diffs
│   ├── upstream-8.4-to-8.5-*.diff          # PHP version changes
│   └── ddev-customizations-8.4-*.diff      # DDEV customizations
└── RECONCILIATION_REPORT.txt
```

### 2. Review DDEV Customizations

Check `diffs/ddev-customizations-*.diff` files to identify all DDEV customizations from the previous version.

**Common DDEV customizations** (update this list as changes are made):

**php.ini files (apache2, cli, fpm):**
```ini
max_execution_time = 600           # Default: 30
request_terminate_timeout = 0      # Allow long-running processes
memory_limit = 1024M               # Default: 128M
display_errors = On                # Default: Off (production)
display_startup_errors = On        # Default: Off
post_max_size = 100M              # Default: 8M
upload_max_filesize = 100M        # Default: 2M
max_input_vars = 5000             # Default: 1000
expose_php = Off                   # Security: don't advertise PHP version
error_reporting = E_ALL           # Show all errors for development
sendmail_path = "/usr/local/bin/mailpit sendmail -t --smtp-addr 127.0.0.1:1025"
date.timezone = UTC                # Set consistent timezone
```

**fpm/pool.d/www.conf:**
```ini
decorate_workers_output = no       # Prevent PHP-FPM from decorating output
```

**fpm/php-fpm.conf:**
```ini
; Usually matches upstream, verify any custom settings
```

### 3. Review PHP Version Changes

Check `diffs/upstream-8.4-to-8.5-*.diff` files to see what PHP upstream changed between versions.

Common types of changes:
- New directives or deprecated directives
- Changed default values
- New sections or comments
- Version number updates in comments

### 4. Create New Version Configuration Files

There are two approaches:

**Approach A: Copy and Modify** (Recommended for minor changes)
```bash
# Copy the old version's DDEV configs
cp -r etc/php/8.4 etc/php/8.5

# Update version-specific comments
find etc/php/8.5 -type f -name "*.ini" -o -name "*.conf" | \
    xargs sed -i '' 's/8\.4/8.5/g'

# Manually review and apply upstream changes from diffs/
```

**Approach B: Start Fresh** (Recommended for major changes)
```bash
# Copy upstream defaults
cp -r ~/tmp/php-configs/upstream-php8.5/php/8.5/* etc/php/8.5/

# Apply each DDEV customization manually
# Use diffs/ddev-customizations-8.4-*.diff as a guide
```

### 5. Handle Extension Configurations

DDEV maintains custom configurations for debugging/profiling extensions in `mods-available/`:

**Extensions with DDEV customizations:**
- `assert.ini` - Assertion configuration for development
- `blackfire.ini` - Blackfire profiler configuration
- `xdebug.ini` - Xdebug debugger configuration
- `xhprof.ini` - XHProf profiler configuration

**Process:**
```bash
# Copy from previous version
cp etc/php/8.4/mods-available/{assert,blackfire,xdebug,xhprof}.ini \
   etc/php/8.5/mods-available/

# Verify each file's settings are compatible with the new PHP version
# Check extension documentation if needed
```

**Important:** All other extension `.ini` files should come from the Debian packages and are installed automatically by the package manager. Do NOT manually copy other extension configs.

### 6. Verify Configuration Files

**Check php.ini files:**
```bash
# Each SAPI should have the same DDEV customizations
diff etc/php/8.5/apache2/php.ini etc/php/8.5/cli/php.ini
diff etc/php/8.5/apache2/php.ini etc/php/8.5/fpm/php.ini

# Should only differ in SAPI-specific sections
```

**Verify FPM configuration:**
```bash
# Check pool configuration
grep -A2 "decorate_workers_output" etc/php/8.5/fpm/pool.d/www.conf

# Check FPM global config
less etc/php/8.5/fpm/php-fpm.conf
```

### 7. Update Container Build Files

Update `containers/ddev-php-base/Dockerfile`:
```dockerfile
# Add the new PHP version to the build logic
# Update version constraints
# Add any new extension installations needed
```

Update `containers/ddev-webserver/Dockerfile`:
```dockerfile
# Add apache2 configuration for the new PHP version if needed
```

### 8. Test the Configuration

**Build and test:**
```bash
# Build the new image
cd containers/ddev-php-base
docker build -t test-php8.5 .

# Run a test container
docker run --rm -it test-php8.5 bash

# Inside container, verify:
php8.5 -v                                    # Version
php8.5 -m                                    # Loaded modules
php8.5 -i | grep -E 'memory_limit|max_execution_time'  # Settings
php8.5 -c /etc/php/8.5/cli -r 'phpinfo();'  # Full config
```

**Test with DDEV:**
```bash
# Create test project
mkdir ~/tmp/test-php8.5 && cd ~/tmp/test-php8.5
ddev config --project-type=php --php-version=8.5
ddev start

# Verify configuration
ddev exec php -v
ddev exec php -i | grep memory_limit
```

### 9. Update Tests and Documentation

**Update tests:**
- `pkg/ddevapp/config_test.go` - Add PHP version to tests
- `pkg/nodeps/php_values.go` - Add to version constants
- Container tests in `containers/*/tests/`

**Update documentation:**
- `docs/content/users/configuration/config.md` - List new PHP version
- Release notes/changelog

## Troubleshooting

### Extension Installation Issues

If extensions fail to install:
```bash
# Check available extensions for new version
docker run --rm debian:bookworm bash -c "
    curl -sSL https://packages.sury.org/debsuryorg-archive-keyring.deb -o /tmp/key.deb
    dpkg -i /tmp/key.deb
    echo 'deb [signed-by=/usr/share/keyrings/deb.sury.org-php.gpg] https://packages.sury.org/php/ bookworm main' > /etc/apt/sources.list.d/php.list
    apt-get update
    apt-cache search php8.5
"
```

### Configuration Not Loading

Check FPM configuration parsing:
```bash
php-fpm8.5 -t  # Test configuration
php-fpm8.5 -tt # Test and show configuration
```

### Differences in SAPIs

If apache2/cli/fpm configs diverge:
- Review the upstream diffs for SAPI-specific sections
- DDEV customizations should generally be the same across all SAPIs
- Some directives are SAPI-specific (e.g., CLI-specific settings)

## Reference: Historical Changes

**PHP 8.4 (July 2024 - Commit 05adf8d72):**
- Initial PHP 8.4 support added
- All extension configs copied from deb.sury.org packages

**PHP 8.4 Cleanup (November 2024 - Commit a1ea2fc7f):**
- Removed package-managed extension .ini files
- Kept only DDEV-customized extensions (assert, blackfire, xdebug, xhprof)
- Reasoning: Package manager handles standard extensions automatically

**PHP 8.4 FPM Update (February 2025 - Commit 0662a3cd3):**
- Added `decorate_workers_output = no` to www.conf
- Prevents PHP-FPM from adding decoration to worker output

## Quick Reference Commands

```bash
# Extract configs for version X.Y
OLD_VERSION=8.4 NEW_VERSION=8.5 bash extract-php-configs.sh

# Find all DDEV customizations in a config
diff -u <(docker run --rm debian:bookworm-with-php bash -c "cat /etc/php/8.4/cli/php.ini") \
        etc/php/8.4/cli/php.ini

# Check what changed in PHP itself
diff -u upstream-8.4/php.ini upstream-8.5/php.ini

# Verify extension is available
apt-cache show php8.5-xdebug

# Test PHP config syntax
php8.5 -c /etc/php/8.5/cli/php.ini -m
```

## Checklist

When adding a new PHP version, ensure you:

- [ ] Extract upstream configs for old and new versions
- [ ] Review all DDEV customizations from old version
- [ ] Review all PHP upstream changes between versions
- [ ] Create new version config files with both DDEV and PHP changes
- [ ] Copy and verify extension configs (assert, blackfire, xdebug, xhprof)
- [ ] Update Dockerfiles for both ddev-php-base and ddev-webserver
- [ ] Test configuration loading in container
- [ ] Test with actual DDEV project
- [ ] Update tests and documentation
- [ ] Update version constants in Go code
- [ ] Update this README if process changed

## Additional Resources

- [PHP Configuration Directives](https://www.php.net/manual/en/ini.php)
- [PHP-FPM Configuration](https://www.php.net/manual/en/install.fpm.configuration.php)
- [deb.sury.org PHP packages](https://packages.sury.org/php/)
- [DDEV PHP Documentation](https://docs.ddev.com/en/stable/users/extend/customization-extendibility/#providing-custom-php-configuration-phpini)