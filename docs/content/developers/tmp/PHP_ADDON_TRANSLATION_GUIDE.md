# PHP Add-on Translation Guide

This guide documents the process, challenges, and solutions for translating bash-based DDEV add-ons to PHP. Based on the successful translation of ddev-redis from bash to PHP.

## Overview

PHP add-ons offer several advantages over bash:

- Better cross-platform compatibility
- Robust YAML parsing with php-yaml extension
- Cleaner conditional logic and error handling
- No shell scripting platform differences

However, the translation process reveals several challenges that need to be addressed systematically.

## Translation Process

### 1. Maintain Test Compatibility

**Critical**: Keep original tests unchanged to validate functional equivalence.

- Only change repository references in test files (`GITHUB_REPO` variable)
- All test assertions and expectations must remain identical
- Use tests as validation that PHP translation behaves exactly like bash original

### 2. Preserve Install.yaml Structure ✅ **BEST PRACTICE ESTABLISHED**

Maintain clean, readable install.yaml by using separate PHP script files:

```yaml
# BEFORE: Monolithic PHP code blocks (unreadable)
post_install_actions:
  - |
    <?php
    // 50+ lines of PHP code here
    ?>

# AFTER: Clean, modular approach using require (not include)
post_install_actions:
  - |
    <?php
    #ddev-description:Install redis settings for Drupal 9+ if applicable
    require 'redis/scripts/setup-drupal-settings.php';
  - |
    <?php
    #ddev-description:Using optimized config if applicable
    require 'redis/scripts/setup-redis-optimized-config.php';
```

**Key Points:**

- **Use `require` not `include`**: Scripts are mandatory dependencies
- **Separate script files**: Each with focused responsibility
- **Start scripts with `<?php`**: Required for proper execution  
- **Include `#ddev-generated`**: For safe cleanup during removal
- **Add to `project_files`**: Include all .php scripts in the file list

## Key Challenges and Solutions

### 1. Environment Variable and Context Support ✅ IMPLEMENTED

**Status**: ✅ **FULLY IMPLEMENTED** - PHP actions now have complete environment variable and context support.

**Available Features**:

- ✅ Access to all standard DDEV environment variables
- ✅ Consistent working directory execution in `/var/www/html/.ddev`
- ✅ Relative path usage for all file operations
- ✅ Processed configuration access via YAML files

**Current Implementation**:

```php
// ✅ AVAILABLE: Standard environment variables (identical to bash actions)
$_ENV['DDEV_APPROOT']     // '/var/www/html'  
$_ENV['DDEV_DOCROOT']     // 'web' or configured docroot
$_ENV['DDEV_PROJECT_TYPE'] // 'drupal', 'laravel', etc.
$_ENV['DDEV_SITENAME']    // Project name
$_ENV['DDEV_PROJECT']     // Project name (alias for SITENAME)
$_ENV['DDEV_PHP_VERSION'] // '8.1', '8.2', etc.
$_ENV['DDEV_WEBSERVER_TYPE'] // 'nginx-fpm', 'apache-fpm'
$_ENV['DDEV_DATABASE']    // 'mysql:8.0', 'postgres:16'
$_ENV['DDEV_DATABASE_FAMILY'] // 'mysql', 'postgres'
$_ENV['IS_DDEV_PROJECT']  // 'true'

// ✅ AVAILABLE: Consistent working directory execution
// All PHP actions execute in: /var/www/html/.ddev
// Enables relative path usage identical to bash actions

// ✅ AVAILABLE: Access to processed configuration
$projectConfig = yaml_parse_file('.ddev-config/project_config.yaml');
$globalConfig = yaml_parse_file('.ddev-config/global_config.yaml');
```

### 2. Container Environment Access ✅ IMPROVED

**Status**: ✅ **SIGNIFICANTLY IMPROVED** - PHP actions now have better configuration access.

**Available Solutions**:

- ✅ **Processed configuration access** - No need for `ddev debug configyaml`
- ✅ **Environment variable access** - Standard DDEV variables available
- ✅ **File system access** - Full project and `.ddev` directory access

**Current Best Practices**:

```php
// ✅ RECOMMENDED: Use processed configuration files
$projectConfig = yaml_parse_file('.ddev-config/project_config.yaml');
$globalConfig = yaml_parse_file('.ddev-config/global_config.yaml');

// ✅ RECOMMENDED: Use environment variables instead of parsing
$projectName = $_ENV['DDEV_PROJECT'];
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$phpVersion = $_ENV['DDEV_PHP_VERSION'];

// ✅ STILL VALID: Direct file access for addon-specific files  
if (file_exists('.env.redis')) {
    $envContent = file_get_contents('.env.redis');
    $isOptimized = strpos($envContent, 'REDIS_OPTIMIZED="true"') !== false;
}
```

**Note**: Host `ddev` commands remain unavailable in containers, but this is now rarely needed due to improved configuration access.

### 3. File Path Translation ✅ SIMPLIFIED

**Status**: ✅ **FULLY RESOLVED** - PHP actions now use identical paths to bash actions.

**Current Path Mappings** (Working directory: `/var/www/html/.ddev`):

```php
// ✅ IDENTICAL to bash actions - no translation needed!
.ddev/config.yaml -> config.yaml (relative path from working directory)
.ddev/redis/file.conf -> redis/file.conf (relative path from working directory)
./project/file -> ../project/file (relative to project root)

// ✅ ENVIRONMENT VARIABLES: Use instead of path construction
$_ENV['DDEV_APPROOT']  // '/var/www/html' (project root)
// Working directory is always: $_ENV['DDEV_APPROOT'] . '/.ddev'
```

**Best Practice**: Use relative paths identical to bash actions - no special handling needed.

### 4. Error Handling and Exit Code Reporting ✅ **FULLY IMPLEMENTED**

**Status**: ✅ **System-level strict error handling implemented** - PHP actions now include automatic error handling equivalent to bash `set -eu -o pipefail`.

**System-Level Implementation**:

```php
// Automatically applied to all PHP actions by the system:
<?php
// PHP strict error handling equivalent to bash 'set -eu -o pipefail'
error_reporting(E_ALL);
ini_set('display_errors', 1);
set_error_handler(function($severity, $message, $file, $line) {
    throw new ErrorException($message, 0, $severity, $file, $line);
});
?>
```

**Manual Error Handling**:

```php
// Standard PHP error handling works correctly
if (!file_exists($configFile)) {
    echo "Error: Configuration file not found: $configFile\n";
    exit(1);
}

// PHP exceptions are caught and reported by system-level handler
if (!$success) {
    throw new Exception("Failed to configure Redis: $errorMessage");
}
```

**Key Benefits**:

- ✅ **Fail-fast behavior**: Warnings and errors cause immediate script termination
- ✅ **Consistent error reporting**: All PHP errors are caught and displayed
- ✅ **Bash equivalence**: Same reliability as bash `set -eu -o pipefail`
- ✅ **Automatic application**: No need to add error handling to individual scripts

### 5. Interactive User Input Support ⚠️ LIMITATIONS

**Challenge**: Interactive user input in containerized PHP actions.

**Current Status**: ⚠️ **COMPLEX LIMITATIONS** - Container environment restricts interactive input.

**Known Limitations**:

- Standard PHP `readline()` functions don't work in container context
- TTY forwarding requires additional container setup
- Interactive prompts may block indefinitely

**Alternative Approaches**:

```php
// RECOMMENDED: Use environment variables for user input
$version = $_ENV['USER_PHP_VERSION'] ?? '8.2';
$confirmed = $_ENV['USER_CONFIRMED'] ?? 'yes';

// ALTERNATIVE: Pre-execution prompts (handled by DDEV host)
// User is prompted before PHP action execution begins
// Results passed via environment variables
```

### 6. Advanced Configuration Access ✅ IMPLEMENTED

**Status**: ✅ **FULLY IMPLEMENTED** - Complete configuration access now available.

**Available Features**:

```php
// ✅ AVAILABLE: Processed project configuration
$projectConfig = yaml_parse_file('.ddev-config/project_config.yaml');
// Contains: fully merged config.*.yaml files, computed values, resolved hostnames

// ✅ AVAILABLE: Global DDEV configuration
$globalConfig = yaml_parse_file('.ddev-config/global_config.yaml');
// Contains: global settings, router settings, performance options

// ✅ AVAILABLE: Environment variables for common values
$appRoot = $_ENV['DDEV_APPROOT'];     // '/var/www/html'
$docroot = $_ENV['DDEV_DOCROOT'];     // 'web' or configured
$projectType = $_ENV['DDEV_PROJECT_TYPE']; // 'drupal', 'laravel', etc.
$databaseType = $_ENV['DDEV_DATABASE']; // 'mysql:8.0', 'postgres:16'

// ✅ AVAILABLE: All project files and directories
// Working directory: /var/www/html/.ddev (enables relative paths)
```

## Best Practices

### 1. Modular Script Organization

Create focused PHP scripts for each responsibility:

```
redis/scripts/
├── setup-drupal-settings.php    # Drupal-specific configuration
├── setup-redis-optimized-config.php    # Optimization handling
└── cleanup-legacy-files.php     # File cleanup operations
```

### 2. Error Handling and Validation

```php
// Always validate file existence and permissions
if (!file_exists($configFile)) {
    echo "Error: Configuration file not found: $configFile\n";
    exit(1);
}

// Check for #ddev-generated before modifying files
if (file_exists($file)) {
    $content = file_get_contents($file);
    if (strpos($content, '#ddev-generated') === false) {
        echo "Warning: File lacks #ddev-generated marker, skipping: $file\n";
        continue;
    }
}
```

### 3. Cross-Platform Compatibility

```php
// Use PHP file functions instead of shell commands
mkdir($directory, 0755, true);  // Instead of: mkdir -p
unlink($file);                  // Instead of: rm -f
copy($source, $dest);           // Instead of: cp
```

### 4. YAML Processing

```php
// Leverage php-yaml for robust parsing
$config = yaml_parse_file('config.yaml');

// Generate clean YAML output
$dockerConfig = [
    'services' => [
        'redis' => [
            'image' => 'redis:7',
            'ports' => ['6379:6379']
        ]
    ]
];
file_put_contents('docker-compose.redis.yaml', 
    "#ddev-generated\n" . yaml_emit($dockerConfig));
```

## Translation Checklist

### Pre-Translation Analysis

- [ ] Map all bash environment variables to PHP equivalents
- [ ] Identify `ddev` command usage requiring alternative approaches  
- [ ] List all file operations and their container path mappings
- [ ] Document external dependencies and command requirements

### Implementation

- [ ] Create modular PHP script structure
- [ ] Update project_files list with .php extensions
- [ ] Convert all bash actions to PHP with proper paths
- [ ] Handle environment variables through config parsing
- [ ] Implement proper error handling and validation

### Validation

- [ ] All original tests pass unchanged
- [ ] File operations produce identical results
- [ ] Error messages maintain user-friendly format
- [ ] Performance comparable to bash implementation

### Documentation

- [ ] Document any behavior differences
- [ ] Note limitations compared to bash version
- [ ] Provide troubleshooting guidance
- [ ] Update README with PHP-specific notes

## Identified System Improvements Needed

Based on the translation experience, several enhancements would improve PHP add-on development:

### 6. Removal Actions Support ✅ **FULLY IMPLEMENTED**

**Status**: ✅ **FULLY IMPLEMENTED** - PHP removal actions now work without requiring running project.

**Available Features**:

- ✅ PHP removal actions execute properly during add-on removal
- ✅ Work correctly when project is stopped/not running
- ✅ Have access to all environment variables and configuration files
- ✅ Proper error handling and output display

**Implementation Benefits**:

```php
<?php
#ddev-description: Remove generated configuration files
$extrasFile = 'docker-compose.redis-extras.yaml';
if (file_exists($extrasFile)) {
    $content = file_get_contents($extrasFile);
    if (strpos($content, '#ddev-generated') !== false) {
        unlink($extrasFile);
        echo "PHP: Removed Redis extras file\n";
    }
}
?>
```

**Key Points**:

- ✅ **No running project required**: Removal actions work when project is stopped
- ✅ **Environment variable access**: All DDEV variables available during removal
- ✅ **Configuration access**: Processed configuration files accessible
- ✅ **Error handling**: System-level strict mode applies to removal actions
- ✅ **Test coverage**: Comprehensive tests validate removal action functionality

### 7. Syntax Validation ✅ **FULLY IMPLEMENTED**

**Status**: ✅ **FULLY IMPLEMENTED** - Comprehensive PHP syntax validation for all PHP actions.

**Available Features**:

- ✅ **Container-based validation**: Uses `php -l` in proper container environment
- ✅ **Include/require validation**: Validates syntax of included/required PHP files
- ✅ **Single-container execution**: Validates and executes in same container for performance
- ✅ **Early failure detection**: Syntax errors prevent execution with clear error messages

**Implementation Details**:

```bash
# System automatically validates original PHP syntax first
php -l /tmp/original-script.php
# Then applies strict error handling and executes
php /tmp/addon-script.php
```

**Validation Scope**:

- ✅ **Main action syntax**: All PHP actions validated before execution
- ✅ **Include/require files**: Recursively validates included PHP files
- ✅ **Error reporting**: Clear syntax error messages with line numbers
- ✅ **Container consistency**: Uses same image specified in install.yaml

### 0. Version Constraint Challenges

**Challenge**: Development and testing of PHP add-ons is complicated by version constraints.

**Issue**: When testing PHP add-ons with development builds (like `v1.23.5-477-gd1efc5064`), version constraints in `install.yaml` (e.g., `ddev_version_constraint: '>= v1.24.3'`) prevent installation, even when the development build contains the required PHP add-on functionality.

**Impact on Development**:

- Cannot test PHP add-ons with development builds without commenting out version constraints
- CI/CD testing requires manual constraint removal  
- Version constraints become a barrier during development rather than a helpful guard

**Potential Solutions**:

- Allow development builds to bypass version constraints with a flag
- Implement more flexible version matching for development builds
- Provide a way to specify "development build compatible" in constraints

### GitHub Actions Testing Challenges

**Challenge**: Testing PHP add-ons requires installing custom DDEV builds that aren't available through standard distribution channels.

**Current Solution**: We implemented a custom build step in `.github/workflows/tests.yml` that:

1. **Dynamically fetches artifacts** from the PHP add-on development branch
2. **Downloads the correct binary** using GitHub's nightly.link service
3. **Replaces the standard DDEV** installed by `ddev/github-action-add-on-test@v2`
4. **Handles API failures** with fallback to known working artifact IDs

```yaml
- name: Install PHP addon DDEV binary
  run: |
    # Get latest successful workflow run with artifacts
    RUN_ID=""
    WORKFLOW_RUNS=$(curl -s --fail "https://api.github.com/repos/rfay/ddev/actions/runs?branch=20250806_rfay_php_addon&per_page=5" || echo '{"workflow_runs":[]}')
    
    for run_id in $(echo "$WORKFLOW_RUNS" | jq -r '.workflow_runs[] | select(.conclusion=="success") | .id'); do
      ARTIFACT_COUNT=$(curl -s --fail "https://api.github.com/repos/rfay/ddev/actions/runs/$run_id/artifacts" | jq '.total_count')
      if [ "$ARTIFACT_COUNT" -gt 0 ]; then
        RUN_ID=$run_id
        break
      fi
    done
    
    # Fallback to known working run if API fails
    if [ -z "$RUN_ID" ]; then
      RUN_ID="16806923996"  # Known working build
    fi
    
    # Download and install PHP addon DDEV binary  
    ARTIFACT_ID=$(curl -s --fail "https://api.github.com/repos/rfay/ddev/actions/runs/$RUN_ID/artifacts" | jq -r '.artifacts[] | select(.name=="ddev-linux-amd64") | .id')
    curl -sSL --fail "https://nightly.link/rfay/ddev/actions/artifacts/$ARTIFACT_ID.zip" -o ddev-php-addon.zip
    unzip -q ddev-php-addon.zip
    sudo cp ddev /usr/local/bin/ddev
    sudo chmod +x /usr/local/bin/ddev
```

**Why This Approach**:

- `github-action-add-on-test@v2` only supports `"stable"` or `"HEAD"` versions, not custom builds
- PHP add-on functionality requires specific development build with container runtime support
- Dynamic artifact fetching ensures tests use latest compatible build
- Fallback mechanism prevents failures due to GitHub API issues

**Future Options for Improvement**:

1. **Enhanced github-action-add-on-test**: Extend the action to support:
   - Custom binary URLs or artifact references
   - Skip DDEV installation when custom binary provided
   - Direct integration with development branches

2. **Separate Test Step**: Move test execution outside the action:
   - Install custom DDEV in separate step
   - Run bats tests directly without using the action
   - More control over test environment setup

3. **Development Distribution**: Create temporary distribution channel:
   - Publish development builds to test registry
   - Allow version constraints like `>= v1.24.0-dev`
   - Enable seamless testing of experimental features

4. **Docker-based Testing**: Containerize the entire test environment:
   - Build custom DDEV container images with PHP add-on support
   - Test add-ons within controlled container environment
   - Eliminate host-level binary installation complexity

**Current Status**: The dynamic artifact approach successfully enables comprehensive testing of PHP add-ons, with all 10 test scenarios consistently using the correct DDEV version (`v1.23.5-478-ga611e2155`) and passing validation.

## Implementation Status Summary

### ✅ COMPLETED FEATURES

1. **Standard Environment Variables** ✅ **FULLY IMPLEMENTED**
   - All DDEV environment variables available in PHP actions
   - Consistent with bash action behavior
   - Working directory set to `/var/www/html/.ddev`

2. **Processed Configuration Access** ✅ **FULLY IMPLEMENTED**
   - Project configuration: `.ddev-config/project_config.yaml`
   - Global configuration: `.ddev-config/global_config.yaml`
   - Automatic creation and cleanup of configuration files

3. **Container Execution Context** ✅ **FULLY IMPLEMENTED**
   - Proper bind mounts for project and configuration access
   - Environment variable injection
   - Consistent working directory execution

4. **System-Level Error Handling** ✅ **FULLY IMPLEMENTED**
   - PHP strict error handling equivalent to bash `set -eu -o pipefail`
   - Automatic application to all PHP actions
   - Fail-fast behavior on warnings and errors

5. **PHP Syntax Validation** ✅ **FULLY IMPLEMENTED**
   - Container-based syntax validation using `php -l`
   - Include/require file validation
   - Single-container optimization for performance

6. **Removal Actions Support** ✅ **FULLY IMPLEMENTED**
   - PHP removal actions work without running project
   - Full environment variable and configuration access
   - Comprehensive test coverage

7. **Single-Container Optimization** ✅ **FULLY IMPLEMENTED**
   - Validation and execution in same container instance
   - Uses correct image specified in install.yaml
   - Improved performance and resource efficiency

### ⚠️ KNOWN LIMITATIONS

1. **Interactive User Input** ⚠️ **COMPLEX LIMITATIONS**
   - Container environment restricts interactive input
   - TTY forwarding requires additional setup
   - **Recommendation**: Use environment variables for user input

2. **Output Control** ✅ **INHERITED FROM BASH ACTIONS**
   - `#ddev-nodisplay` directive works correctly (inherited from bash implementation)
   - Standard PHP error handling works correctly
   - Exit code handling works and is tested

### Current Translation Benefits

With the fully implemented features, PHP translations now offer comprehensive advantages:

**Before (Manual Configuration Parsing and No Error Handling)**:

```php
// Complex manual configuration parsing
$config = yaml_parse_file('config.yaml');
$targetDir = '../' . ($config['docroot'] ?? 'web') . '/sites/default';
$extraDockerFile = 'docker-compose.redis_extra.yaml';

// Manual environment variable extraction
$projectType = $config['type'] ?? 'php';
$siteName = $config['name'] ?? 'default';

// No error handling - warnings/errors could be ignored
```

**After (With All Implemented Features)** ✅:

```php
// ✅ AUTOMATIC: System-level strict error handling (equivalent to bash 'set -eu -o pipefail')
// ✅ AUTOMATIC: PHP syntax validation before execution

// ✅ AVAILABLE: Direct environment access (working directory: /var/www/html/.ddev)
$targetDir = '../' . $_ENV['DDEV_DOCROOT'] . '/sites/default';
$extraDockerFile = 'docker-compose.redis_extra.yaml';

// ✅ AVAILABLE: Direct environment variable access
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$siteName = $_ENV['DDEV_SITENAME'];

// ✅ AVAILABLE: Access processed configuration when needed
$projectConfig = yaml_parse_file('.ddev-config/project_config.yaml');
$globalConfig = yaml_parse_file('.ddev-config/global_config.yaml');

// ✅ AVAILABLE: Removal actions work without running project
// ✅ AVAILABLE: Single-container execution for performance
```

**Key Benefits Achieved** ✅:

1. **Environment Variables**: All standard DDEV variables available via `$_ENV`
2. **Working Directory**: Consistent `/var/www/html/.ddev` execution context
3. **Configuration Access**: Full processed configuration available
4. **Path Consistency**: Relative paths work identically to bash actions
5. **Error Handling**: Automatic strict mode catches warnings/errors
6. **Syntax Validation**: PHP syntax validated before execution
7. **Removal Actions**: Work reliably without running project
8. **Performance**: Single-container optimization reduces resource usage

## Migration Strategy

### For Simple Add-ons ✅ READY

1. Convert bash actions to inline PHP
2. Use environment variables instead of config parsing: `$_ENV['DDEV_DOCROOT']`  
3. Leverage relative paths from working directory: `/var/www/html/.ddev`
4. Test all scenarios thoroughly

### For Complex Add-ons ✅ READY

1. Break down into modular PHP scripts
2. Use processed configuration access: `yaml_parse_file('.ddev-config/project_config.yaml')`
3. Leverage environment variables for common values
4. Maintain comprehensive test coverage

### For Add-ons with Host Dependencies ⚠️ EVALUATE

1. **Interactive Input**: Consider environment variable approach
2. **Host Commands**: Convert to container-based equivalents where possible
3. **Complex Workflows**: Hybrid bash/PHP approach may be optimal
4. **Document Limitations**: Be clear about container environment constraints

## Example: ddev-redis Translation Results ✅ **PRODUCTION-READY**

**Full Test Suite Results**: ✅ All 10 test scenarios passing

- Default installation
- Default with optimized config  
- Drupal 8+ installation  
- Drupal 7 installation (settings skipped)
- Drupal with disabled settings management
- Laravel with Redis backend variants
- Laravel with Valkey backend variants
- Multiple Redis versions (6, 7)
- Alpine and standard image variants

**Translation Highlights**:

### Clean Modular Structure

```yaml
# install.yaml - Clean and readable
post_install_actions:
  - |
    <?php
    #ddev-description:Install redis settings for Drupal 9+ if applicable
    require 'redis/scripts/setup-drupal-settings.php';
  - |
    <?php
    #ddev-description:Using optimized config if --redis-optimized=true  
    require 'redis/scripts/setup-redis-optimized-config.php';
```

### Environment Variable Usage

```php
// redis/scripts/setup-drupal-settings.php
<?php
#ddev-generated

// ✅ Use environment variables instead of manual config parsing
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$docroot = $_ENV['DDEV_DOCROOT'];
$appRoot = $_ENV['DDEV_APPROOT'];

// ✅ Use processed configuration when needed
$config = yaml_parse_file('.ddev-config/project_config.yaml');
if (isset($config['disable_settings_management']) && $config['disable_settings_management'] === true) {
    exit(0);
}
```

### YAML Processing with php-yaml

```php
// Generate docker-compose extra file using yaml_emit instead of heredoc
$dockerConfig = [
    'services' => [
        'redis' => [
            'deploy' => [
                'resources' => [
                    'limits' => ['cpus' => '2.5', 'memory' => '768M'],
                    'reservations' => ['cpus' => '1.5', 'memory' => '512M']
                ]
            ]
        ]
    ]
];
$yamlContent = "#ddev-generated\n" . yaml_emit($dockerConfig);
file_put_contents($extraDockerFile, $yamlContent);
```

**Results**:

- ✅ **Performance**: Comparable to bash implementation
- ✅ **Reliability**: Identical behavior validated through unchanged tests  
- ✅ **Maintainability**: Improved code organization and error handling
- ✅ **Cross-platform**: Better compatibility than bash scripts
- ✅ **Error handling**: System-level strict mode catches issues early

## Technical Implementation: Docker Compose Architecture ✅ **IMPLEMENTED**

**Status**: ✅ **FULLY IMPLEMENTED** - PHP actions now use docker-compose for improved container lifecycle management.

### Container Execution Evolution

**Previous Implementation (`RunSimpleContainer`)**:

- Direct Docker API container creation
- Manual container lifecycle management  
- Basic cleanup on completion
- Limited integration with DDEV networking

**Current Implementation (Docker Compose)**:

- In-memory docker-compose project creation
- Automatic container lifecycle management
- Proper DDEV container labeling
- Integrated host.docker.internal setup
- Better resource cleanup

### Architecture Details

PHP actions are executed using DDEV's standard docker-compose infrastructure:

```go
// Create in-memory compose project
phpProject, err := dockerutil.CreateComposeProject("name: ddev-php-action")

// Configure service with proper DDEV integration
serviceConfig := composeTypes.ServiceConfig{
    Name:        "php-runner", 
    Image:       image,
    Command:     cmd,
    WorkingDir:  "/var/www/html/.ddev",
    Environment: buildEnvironmentMap(env),
    Volumes:     buildVolumeConfigs(binds),
    Labels: composeTypes.Labels{
        "com.ddev.site-name": app.Name,
        "com.ddev.approot":   app.AppRoot,
    },
}

// Execute with docker-compose
dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
    ComposeYaml: phpProject,
    Action:      []string{"run", "--rm", "--no-deps", serviceName},
})
```

### DDEV Integration Features

**Container Labels**:

- `com.ddev.site-name`: Project identification
- `com.ddev.approot`: Project root path

**Host.docker.internal Setup**:

- Uses `dockerutil.GetHostDockerInternalIP()` for proper IP resolution
- Handles WSL2, Docker Desktop, and Linux Docker Engine scenarios
- Enables debugging support (Xdebug connections to host IDE)

**Network Integration**:

- Containers properly isolated (no network access needed for file operations)
- Future debugging features can leverage host.docker.internal setup

### Benefits of Docker Compose Approach

1. **Consistency with DDEV**: Uses same container management as DDEV core
2. **Better Lifecycle Management**: Automatic cleanup and resource management  
3. **Proper Integration**: Standard DDEV labels and networking setup
4. **Future-Proof**: Supports debugging and advanced networking features
5. **No Temporary Files**: All configuration handled in memory

### Migration Impact

- ✅ **Backward Compatible**: All existing PHP add-ons continue working unchanged
- ✅ **Performance**: Comparable or better container startup times
- ✅ **Reliability**: More robust error handling and cleanup
- ✅ **Debugging Ready**: Host.docker.internal properly configured for future debugging features

## Conclusion

PHP add-on translation is viable and provides significant benefits for complex configuration processing. However, it requires careful handling of container environment limitations and systematic approach to configuration access.

The most significant challenge is the lack of access to processed DDEV configuration and global settings from within PHP containers. Addressing this limitation would significantly improve the PHP add-on development experience.

For add-ons that primarily handle file operations, YAML processing, and conditional logic, PHP translation offers substantial advantages in maintainability, cross-platform compatibility, and robustness.
