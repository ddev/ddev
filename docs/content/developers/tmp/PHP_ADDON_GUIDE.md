# DDEV PHP-based Add-ons Guide

DDEV add-ons now support PHP-based actions alongside traditional bash actions, opening up new possibilities for complex configuration processing, YAML manipulation, and cross-platform compatibility.

## Overview

PHP-based add-ons allow you to write installation and configuration logic in PHP instead of bash. This provides:

- **Better YAML processing** with the built-in php-yaml extension
- **Cross-platform compatibility** (no shell scripting differences)
- **Rich string manipulation** and data processing capabilities
- **Access to DDEV project configuration** through mounted directories
- **Familiar syntax** for developers working with PHP projects

## How PHP Actions Work

DDEV automatically detects PHP actions by looking for scripts that start with `<?php`. When found, these actions are executed in a PHP container using docker-compose with access to:

- **Working directory:** `/var/www/html/.ddev` (your project's .ddev directory)
- **Environment variables:** All standard DDEV environment variables (identical to bash actions)
- **Processed configuration:** Access to fully processed project and global configuration via YAML files
- **Full project access:** Complete read/write access to your project repository
- **php-yaml extension:** For robust YAML parsing and generation
- **Standard PHP functionality:** File manipulation, string processing, etc.
- **DDEV Integration:** Proper container labels and host.docker.internal setup for debugging support

## Basic Syntax Comparison

### Traditional Bash Action

```yaml
pre_install_actions:
  - |
    #ddev-description: Process project configuration
    PROJECT_NAME=$(grep "^name:" .ddev/config.yaml | cut -d: -f2 | tr -d ' ')
    echo "Setting up project: $PROJECT_NAME"
    
    # Create configuration file
    cat > .ddev/my-addon-config.yaml << EOF
    name: $PROJECT_NAME
    type: addon-config
    EOF
```

### Equivalent PHP Action

```yaml
pre_install_actions:
  - |
    <?php
    #ddev-description: Process project configuration
    
    // ✅ RECOMMENDED: Use environment variables for common values
    $projectName = $_ENV['DDEV_PROJECT'];
    echo "Setting up project: $projectName\n";
    
    // ✅ ALTERNATIVE: Access processed configuration when needed
    // $config = yaml_parse_file('.ddev-config/project_config.yaml');
    // $projectName = $config['name'] ?? 'unknown';
    
    // Create configuration file
    $addonConfig = [
        'name' => $projectName,
        'type' => 'addon-config'
    ];
    file_put_contents('my-addon-config.yaml', 
        yaml_emit($addonConfig));
    ?>
```

## System-Level Error Handling ✅ **NEW FEATURE**

PHP actions now include automatic strict error handling equivalent to bash `set -eu -o pipefail`:

```php
// Automatically applied to all PHP actions:
<?php
// PHP strict error handling equivalent to bash 'set -eu -o pipefail'
error_reporting(E_ALL);
ini_set('display_errors', 1);
set_error_handler(function($severity, $message, $file, $line) {
    throw new ErrorException($message, 0, $severity, $file, $line);
});
?>
```

This ensures PHP actions fail fast on warnings and errors, providing reliable error detection similar to bash actions.

## Available Environment Variables ✅ **NEW FEATURE**

PHP actions now have access to all standard DDEV environment variables, making them functionally equivalent to bash actions:

```php
<?php
// Project Information
$_ENV['DDEV_PROJECT']        // Project name
$_ENV['DDEV_SITENAME']       // Same as DDEV_PROJECT
$_ENV['DDEV_PROJECT_TYPE']   // 'drupal', 'wordpress', 'laravel', etc.
$_ENV['DDEV_APPROOT']        // '/var/www/html' (project root)
$_ENV['DDEV_DOCROOT']        // 'web', 'public', or configured docroot
$_ENV['DDEV_TLD']            // 'ddev.site' or configured TLD

// Technology Stack
$_ENV['DDEV_PHP_VERSION']    // '8.1', '8.2', '8.3', etc.
$_ENV['DDEV_WEBSERVER_TYPE'] // 'nginx-fpm', 'apache-fpm'
$_ENV['DDEV_DATABASE']       // 'mysql:8.0', 'postgres:16', etc.
$_ENV['DDEV_DATABASE_FAMILY'] // 'mysql', 'postgres'

// System Information
$_ENV['DDEV_VERSION']        // Current DDEV version
$_ENV['DDEV_FILES_DIRS']     // Upload directories (comma-separated)
$_ENV['DDEV_MUTAGEN_ENABLED'] // 'true' or 'false'
$_ENV['IS_DDEV_PROJECT']     // Always 'true' in DDEV context
?>
```

**Migration Benefit:** This eliminates the need for manual config parsing in most cases!

```php
// ❌ OLD APPROACH (no longer needed)
$config = yaml_parse_file('config.yaml');
$projectType = $config['type'] ?? 'php';
$docroot = $config['docroot'] ?? 'web';

// ✅ NEW APPROACH (recommended)
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$docroot = $_ENV['DDEV_DOCROOT'];
```

## Processed Configuration Access ✅ **NEW FEATURE**

For complex configuration needs, PHP actions can access fully processed DDEV configuration:

```php
<?php
// Access processed project configuration (equivalent to 'ddev debug configyaml')
$projectConfig = yaml_parse_file('.ddev-config/project_config.yaml');

// Access global DDEV configuration
$globalConfig = yaml_parse_file('.ddev-config/global_config.yaml');

// These files contain all merged config.*.yaml files with computed values
// including resolved hostnames, ports, and complete service definitions
?>
```

## Key Differences from Bash Actions

### 1. Execution Environment

**Bash Actions:**

- Run directly on the host system
- Have access to all host tools and environment
- Current directory is the project root

**PHP Actions:**

- Run inside a PHP container using docker-compose
- Limited to PHP and basic container tools
- Working directory is `/var/www/html/.ddev` (the project's .ddev directory)
- **Full project repository mounted at `/var/www/html/`** (read/write access)
- Proper DDEV container labels and host.docker.internal setup for debugging

### 2. File Access

**Bash Actions:**

```bash
# Direct access to project and .ddev directory
cat .ddev/config.yaml
echo "data" > .ddev/output.txt
echo "content" > web/sites/default/settings.php
```

**PHP Actions:**

```php
<?php
// ✅ RECOMMENDED: Use environment variables (working directory: /var/www/html/.ddev)
$projectName = $_ENV['DDEV_PROJECT'];
$docroot = $_ENV['DDEV_DOCROOT'];
file_put_contents('output.txt', 'data');

// ✅ RECOMMENDED: Use processed configuration when needed
$config = yaml_parse_file('.ddev-config/project_config.yaml');

// Access full project repository 
$projectFiles = scandir('/var/www/html');
file_put_contents("/var/www/html/{$docroot}/sites/default/settings.php", 'content');
?>
```

### 3. Error Handling

**Bash Actions:**

```bash
#ddev-description: Check if file exists
if [ ! -f ".ddev/config.yaml" ]; then
    echo "Config file not found!"
    exit 1
fi
```

**PHP Actions:**

```php
<?php
#ddev-description: Check if file exists
if (!file_exists('config.yaml')) {
    echo "Config file not found!\n";
    exit(1);
}
?>
```

### 4. YAML Processing

**Bash Actions (limited):**

```bash
# Basic grep-based parsing
DB_VERSION=$(grep "database:" -A 2 .ddev/config.yaml | grep "version:" | cut -d: -f2 | tr -d ' ')
```

**PHP Actions (robust):**

```php
<?php
// ✅ RECOMMENDED: Use environment variables for common values
$databaseInfo = $_ENV['DDEV_DATABASE']; // e.g., 'mysql:8.0' or 'postgres:16'
$dbVersion = explode(':', $databaseInfo)[1] ?? 'default';

// ✅ ALTERNATIVE: Use processed configuration for complex data
$config = yaml_parse_file('.ddev-config/project_config.yaml');
$dbVersion = $config['database']['version'] ?? 'default';

// Generate complex YAML structures
$newConfig = [
    'services' => [
        'myservice' => [
            'image' => 'nginx:latest',
            'environment' => [
                'DB_VERSION' => $dbVersion
            ]
        ]
    ]
];
file_put_contents('docker-compose.myservice.yaml', 
    "#ddev-generated\n" . yaml_emit($newConfig));
?>
```

## Separate PHP Script Files ✅ **BEST PRACTICE**

For complex logic, create separate PHP script files to keep install.yaml clean and readable:

### File Structure

```
.ddev/
├── install.yaml
└── scripts/
    ├── setup-drupal.php
    ├── optimize-config.php
    └── cleanup.php
```

### Clean install.yaml Pattern

```yaml
name: my-addon

post_install_actions:
  - |
    <?php
    #ddev-description:Configure Drupal settings
    require 'scripts/setup-drupal.php';
  - |
    <?php  
    #ddev-description:Apply optimized configuration
    require 'scripts/optimize-config.php';
  - |
    <?php
    #ddev-description:Clean up temporary files
    require 'scripts/cleanup.php';
```

### Example Script File: `scripts/setup-drupal.php`

```php
<?php
#ddev-generated

// ✅ Use environment variables for project information
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$docroot = $_ENV['DDEV_DOCROOT'];
$appRoot = $_ENV['DDEV_APPROOT'];

// Exit early if not applicable
if (strpos($projectType, 'drupal') !== 0) {
    echo "Not a Drupal project, skipping Drupal setup\n";
    exit(0);
}

// ✅ Use processed configuration when needed
$config = yaml_parse_file('.ddev-config/project_config.yaml');
if (isset($config['disable_settings_management']) && $config['disable_settings_management'] === true) {
    echo "Settings management disabled, skipping\n";
    exit(0);
}

// Perform Drupal-specific setup
$targetDir = "{$appRoot}/{$docroot}/sites/default";
$targetFile = "{$targetDir}/settings.ddev.php";

// Copy settings file
if (!copy('scripts/settings.ddev.php', $targetFile)) {
    echo "Error: Failed to copy settings file\n";
    exit(1);
}

echo "Drupal settings configured successfully\n";
```

### Benefits of This Approach

1. **Clean install.yaml**: Easy to read and understand the add-on workflow
2. **Modular logic**: Each script has a focused responsibility  
3. **Reusable components**: Scripts can be shared between different actions
4. **Better error handling**: Individual script failures don't affect other operations
5. **Easier testing**: Scripts can be tested independently

### Key Points for Script Files

- **Start with `<?php`**: Required for proper PHP execution
- **Include `#ddev-generated`**: For safe file cleanup during removal
- **Use `require` not `include`**: Scripts are mandatory dependencies
- **Exit codes matter**: Use `exit(0)` for success, `exit(1)` for errors
- **Add to `project_files`**: Include script files in install.yaml file list

```yaml
project_files:
  - scripts/setup-drupal.php
  - scripts/optimize-config.php
  - scripts/cleanup.php
```

## Practical Examples

### Example 1: Environment-based Configuration

```yaml
name: conditional-config
image: ddev/ddev-webserver:latest

pre_install_actions:
  - |
    <?php
    #ddev-description: Generate environment-specific configuration
    
    // ✅ RECOMMENDED: Use environment variable
    $projectType = $_ENV['DDEV_PROJECT_TYPE'];
    
    // ✅ ALTERNATIVE: Use processed configuration if needed
    // $config = yaml_parse_file('.ddev-config/project_config.yaml');
    // $projectType = $config['type'] ?? 'php';
    
    // Generate different configs based on project type
    $services = [];
    
    switch($projectType) {
        case 'drupal':
            $services['redis'] = [
                'image' => 'redis:7-alpine',
                'ports' => ['6379:6379']
            ];
            break;
        case 'wordpress':
            $services['memcached'] = [
                'image' => 'memcached:alpine',
                'ports' => ['11211:11211']
            ];
            break;
        default:
            $services['cache'] = [
                'image' => 'nginx:alpine'
            ];
    }
    
    $composeContent = [
        'services' => $services
    ];
    
    file_put_contents('docker-compose.conditional.yaml',
        "#ddev-generated\n" . yaml_emit($composeContent));
        
    echo "Generated configuration for $projectType project\n";
    ?>
```

### Example 2: Complex Data Transformation

```yaml
name: data-transformer
image: ddev/ddev-webserver:latest

yaml_read_files:
  platform_config: ".platform.app.yaml"

pre_install_actions:
  - |
    <?php
    #ddev-description: Transform Platform.sh config to DDEV format
    
    // This would be populated by yaml_read_files
    $platformConfig = '.platform.app.yaml';
    
    if (file_exists($platformConfig)) {
        $platform = yaml_parse_file($platformConfig);
        
        // Extract PHP version
        $phpVersion = '8.1';
        if (isset($platform['type']) && strpos($platform['type'], 'php:') === 0) {
            $phpVersion = str_replace('php:', '', $platform['type']);
        }
        
        // Transform build commands
        $hooks = [];
        if (isset($platform['hooks']['build'])) {
            $buildCommands = explode("\n", trim($platform['hooks']['build']));
            $hooks['post-start'] = array_map(function($cmd) {
                return ['exec' => trim($cmd)];
            }, $buildCommands);
        }
        
        // Generate DDEV config
        $ddevConfig = [
            'php_version' => $phpVersion,
            'hooks' => $hooks
        ];
        
        file_put_contents('config.platform.yaml',
            "#ddev-generated\n" . yaml_emit($ddevConfig));
            
        echo "Transformed Platform.sh config (PHP $phpVersion)\n";
    }
    ?>
```

### Example 3: Repository File Management

```yaml
name: settings-manager
image: ddev/ddev-webserver:latest

pre_install_actions:
  - |
    <?php
    #ddev-description: Configure project settings files
    
    // ✅ RECOMMENDED: Use environment variable
    $projectType = $_ENV['DDEV_PROJECT_TYPE'];
    $docroot = $_ENV['DDEV_DOCROOT'];
    
    // ✅ ALTERNATIVE: Use processed configuration for complex decisions
    // $config = yaml_parse_file('.ddev-config/project_config.yaml');
    
    // Create appropriate settings files based on project type
    switch($projectType) {
        case 'drupal':
            // ✅ RECOMMENDED: Use environment variable for docroot
            $docroot = $_ENV['DDEV_DOCROOT'];
            $settingsDir = "/var/www/html/{$docroot}/sites/default";
            if (!is_dir($settingsDir)) {
                mkdir($settingsDir, 0755, true);
            }
            
            $settingsFile = $settingsDir . '/settings.ddev.php';
            $settingsContent = "<?php\n// DDEV settings file\n";
            $settingsContent .= "\$databases['default']['default'] = [\n";
            $settingsContent .= "  'database' => 'db',\n";
            $settingsContent .= "  'username' => 'db',\n";
            $settingsContent .= "  'password' => 'db',\n";
            $settingsContent .= "  'host' => 'db',\n";
            $settingsContent .= "  'driver' => 'mysql',\n";
            $settingsContent .= "];\n";
            
            file_put_contents($settingsFile, $settingsContent);
            echo "Created Drupal settings file\n";
            break;
            
        case 'wordpress':
            $configFile = '/var/www/html/wp-config-ddev.php';
            $configContent = "<?php\n// DDEV WordPress configuration\n";
            $configContent .= "define('DB_NAME', 'db');\n";
            $configContent .= "define('DB_USER', 'db');\n";
            $configContent .= "define('DB_PASSWORD', 'db');\n";
            $configContent .= "define('DB_HOST', 'db');\n";
            
            file_put_contents($configFile, $configContent);
            echo "Created WordPress config file\n";
            break;
    }
    ?>

post_install_actions:
  - |
    <?php
    #ddev-description: Verify settings files are accessible
    
    // Check that files were created and are readable from project
    $possibleFiles = [
        '/var/www/html/web/sites/default/settings.ddev.php',
        '/var/www/html/wp-config-ddev.php'
    ];
    
    foreach ($possibleFiles as $file) {
        if (file_exists($file)) {
            $size = filesize($file);
            echo "Settings file $file exists ($size bytes)\n";
            
            // Verify it's valid PHP
            $content = file_get_contents($file);
            if (strpos($content, '<?php') === 0) {
                echo "File has valid PHP syntax\n";
            }
        }
    }
    ?>
```

### Example 4: Mixed Bash and PHP Actions

```yaml
name: mixed-actions-addon

pre_install_actions:
  - |
    #ddev-description: Prepare system dependencies
    echo "Installing system dependencies..."
    # Bash is better for system-level tasks
    
  - |
    <?php
    #ddev-description: Process configuration files  
    // ✅ RECOMMENDED: PHP is better for data processing
    $projectName = $_ENV['DDEV_PROJECT'];
    // Alternative: $config = yaml_parse_file('.ddev-config/project_config.yaml');
    echo "Processing config for: $projectName\n";
    ?>
    
  - |
    #ddev-description: Set file permissions
    chmod +x .ddev/commands/web/mycommand
    # Back to bash for file system operations
```

## Best Practices

### 1. Use Appropriate Tool for Each Task

- **Use PHP for:** Complex data processing, YAML manipulation, conditional logic
- **Use Bash for:** File permissions, system commands, environment setup

### 2. Proper Description Comments

```php
<?php
#ddev-description: Generate service configuration based on project type
// PHP comment explaining the logic
?>
```

### 3. Proper Error Handling

```php
<?php
// ✅ RECOMMENDED: Check environment variables
if (empty($_ENV['DDEV_PROJECT'])) {
    echo "Error: DDEV environment not available\n";
    exit(1);
}

// ✅ ALTERNATIVE: Check processed configuration files
if (!file_exists('.ddev-config/project_config.yaml')) {
    echo "Error: DDEV config file not found\n";
    exit(1);
}
?>
```

### 4. Clean Output

```php
<?php
#ddev-description: Configure database settings
// Keep output informative but concise
echo "Database configured for project\n";
// Avoid verbose debugging output unless debugging
?>
```

## Real-World Example: ddev-redis PHP Translation

The ddev-redis add-on has been successfully translated from bash to PHP, demonstrating production-ready PHP add-on patterns:

### Original Structure (Bash)

```
redis/scripts/
├── setup-drupal-settings.sh       # Drupal configuration
├── setup-redis-optimized-config.sh # Optimization handling  
└── settings.ddev.redis.php         # Settings template
```

### Translated Structure (PHP)

```
redis/scripts/
├── setup-drupal-settings.php      # PHP version
├── setup-redis-optimized-config.php # PHP version
└── settings.ddev.redis.php        # Unchanged template
```

### Clean install.yaml

```yaml
name: redis

post_install_actions:
  - |
    <?php
    #ddev-description:Install redis settings for Drupal 9+ if applicable
    require 'redis/scripts/setup-drupal-settings.php';
  - |
    <?php
    #ddev-description:Using optimized config if --redis-optimized=true
    require 'redis/scripts/setup-redis-optimized-config.php';
  - |
    <?php
    #ddev-description:Remove redis/scripts if there are no files
    $scriptsDir = 'redis/scripts';
    if (is_dir($scriptsDir) && count(scandir($scriptsDir)) <= 2) {
        rmdir($scriptsDir);
    }
```

### Key Implementation Details

**Environment File Detection:**

```php
// Direct file access instead of ddev dotenv command
$envFile = '.env.redis';
if (file_exists($envFile)) {
    $envContent = file_get_contents($envFile);
    $isOptimized = strpos($envContent, 'REDIS_OPTIMIZED=true') !== false;
}
```

**YAML Generation with php-yaml:**

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

**Environment Variable Usage:**

```php
// ✅ Use environment variables instead of manual config parsing
$projectType = $_ENV['DDEV_PROJECT_TYPE'];
$docroot = $_ENV['DDEV_DOCROOT']; 
$appRoot = $_ENV['DDEV_APPROOT'];
$siteName = $_ENV['DDEV_SITENAME'];
```

**Results:**

- ✅ All 10 test scenarios pass identically to bash version
- ✅ Cleaner, more maintainable code structure
- ✅ Better error handling with system-level strict mode
- ✅ Cross-platform compatibility
- ✅ Robust YAML processing

## Available Test Examples

The DDEV repository includes several test add-ons demonstrating PHP functionality:

### Basic PHP add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/basic-php-addon/`

Shows fundamental PHP action usage:

- Reading DDEV configuration
- File creation and manipulation
- Mixed PHP and bash actions

### Complex PHP add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/complex-php-addon/`

Demonstrates advanced features:

- YAML file parsing with php-yaml extension
- Complex data structure manipulation
- Docker compose generation

### Mixed Actions add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/mixed-addon/`

Shows best practices for combining bash and PHP:

- Sequential bash and PHP actions
- Proper description usage
- Action coordination

### Varnish PHP add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/varnish-php-addon/`

Real-world example converting a bash add-on to PHP:

- Configuration file processing
- `HEREDOC` usage for clean YAML generation
- Error handling and validation

### Custom Image add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/custom-image-addon/`

Demonstrates using custom PHP images:

- Specifying alternative PHP versions
- Image compatibility testing

### Repository Access add-on

**Location:** `cmd/ddev/cmd/testdata/TestCmdAddonPHP/repo-access-addon/`

Shows full project repository access capabilities:

- Reading and scanning project files
- Creating files in project root and subdirectories
- Managing settings files like ddev-redis does
- Directory creation and file management

## Migration from Bash

When migrating existing bash actions to PHP, consider:

1. **File paths:** Change `.ddev/file` to relative paths like `file` (working directory is `/var/www/html/.ddev`)
2. **Project files:** Change `./file` to `/var/www/html/file`
3. **YAML parsing:** Replace grep/sed with `yaml_parse_file()`
4. **Variables:** ✅ **NOW AVAILABLE** - Use `$_ENV['DDEV_PROJECT']` and other environment variables directly
5. **Output:** Add `\n` to echo statements for proper line breaks

## Limitations

- No direct access to host system (by design)
- Limited to tools available in the PHP container
- File operations restricted to mounted directories (`.ddev` and project root)
- Cannot execute host-specific commands like `ddev` itself

## Container Image

PHP actions by default use the default `ddev-webserver` image `ddev/ddev-webserver` which includes:

- PHP with php-yaml extension
- Basic container utilities
- Access to mounted project `.ddev` directory

However, an `image` may be specified in `install.yaml`; the specified image must contain the php-yaml extension.

## Getting Started

1. Start with the [`ddev-addon-template`](https://github.com/ddev/ddev-addon-template)
2. Replace bash actions with PHP equivalents where beneficial
3. Test thoroughly with the test add-ons as references
4. Follow the [Add-on Maintenance Guide](https://ddev.com/blog/ddev-add-on-maintenance-guide/) for ongoing updates

PHP-based add-ons provide a powerful complement to traditional bash actions, enabling more sophisticated configuration processing while maintaining the simplicity and reliability DDEV users expect.
