---
search:
  boost: 2
---

# Creating DDEV Add-ons

DDEV add-ons provide a powerful way to extend development environments. You can create add-ons using traditional bash actions or the new PHP-based actions for complex configuration processing.

## Quick Start

1. Use the [`ddev-addon-template`](https://github.com/ddev/ddev-addon-template) repository
2. Click "Use this template" to create your own repository
3. Customize the `install.yaml` file
4. Test with `tests.bats`
5. Create a release when ready
6. Add the `ddev-get` label to your GitHub repository

See this [screencast](https://www.youtube.com/watch?v=fPVGpKGr0f4) for a walkthrough.

## Add-on Structure

Every add-on requires an `install.yaml` file with these sections:

```yaml
name: my-addon
pre_install_actions: []
project_files: []
global_files: []
post_install_actions: []
removal_actions: []
```

### Core Sections

- **`name`**: The add-on name used in `ddev add-on` commands
- **`pre_install_actions`**: Scripts executed before files are copied
- **`project_files`**: Files copied to the project's `.ddev` directory
- **`global_files`**: Files copied to the global `~/.ddev/` directory
- **`post_install_actions`**: Scripts executed after files are copied
- **`removal_actions`**: Scripts executed when removing the add-on

### Advanced Sections

- **`ddev_version_constraint`**: Minimum DDEV version required
- **`dependencies`**: Other add-ons this add-on depends on
- **`yaml_read_files`**: YAML files to read for template processing

## Action Types: Bash vs PHP

### Traditional Bash Actions

Bash actions run directly on the host system and are suitable for:

- File permissions and system commands
- Environment setup and package installation
- Direct command execution
- Simple file operations

```yaml
name: bash-example

post_install_actions:
  - |
    #ddev-description: Configure project settings
    echo "Setting up project: $DDEV_PROJECT"
    chmod +x .ddev/commands/web/mycommand
```

### PHP-based Actions ✨ **NEW**

PHP actions provide powerful capabilities for:

- Complex data processing and YAML manipulation
- Conditional logic based on project configuration
- Cross-platform compatibility
- File content generation and template processing

#### Why Use PHP Actions?

- **Better YAML processing** with the built-in php-yaml extension
- **Cross-platform compatibility** (no shell scripting differences)
- **Rich string manipulation** and data processing capabilities
- **Access to DDEV project configuration** through environment variables
- **Familiar syntax** for developers working with PHP projects

#### Basic PHP Action

```yaml
name: php-example

post_install_actions:
  - |
    <?php
    #ddev-description: Process project configuration
    
    // Access DDEV environment variables
    $projectName = $_ENV['DDEV_PROJECT'];
    $projectType = $_ENV['DDEV_PROJECT_TYPE'];
    $docroot = $_ENV['DDEV_DOCROOT'];
    
    echo "Setting up $projectType project: $projectName\n";
    
    // Generate YAML configuration
    $config = [
        'services' => [
            'myservice' => [
                'image' => 'nginx:latest',
                'environment' => [
                    'PROJECT_TYPE' => $projectType
                ]
            ]
        ]
    ];
    
    file_put_contents('docker-compose.myservice.yaml',
        "#ddev-generated\n" . yaml_emit($config));
    ?>
```

#### Available Environment Variables

PHP actions have access to all standard DDEV environment variables:

```php
<?php
// Project Information
$_ENV['DDEV_PROJECT']        // Project name
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
$_ENV['DDEV_MUTAGEN_ENABLED'] // 'true' or 'false'
?>
```

#### PHP Action Execution Environment

- **Working directory**: `/var/www/html/.ddev` (your project's .ddev directory)
- **Project access**: Full read/write access to project repository at `/var/www/html/`
- **Error handling**: Automatic strict error handling (equivalent to bash `set -eu`)
- **Extensions**: php-yaml extension for robust YAML processing

#### Advanced PHP Example: Conditional Configuration

```yaml
name: conditional-config

pre_install_actions:
  - |
    <?php
    #ddev-description: Generate environment-specific configuration
    
    $projectType = $_ENV['DDEV_PROJECT_TYPE'];
    $services = [];
    
    // Different services based on project type
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
    
    $composeContent = ['services' => $services];
    file_put_contents('docker-compose.conditional.yaml',
        "#ddev-generated\n" . yaml_emit($composeContent));
        
    echo "Generated configuration for $projectType project\n";
    ?>
```

#### Separate PHP Script Files (Best Practice)

For complex logic, create separate PHP script files using your add-on's namespace:

**File structure:**

```
.ddev/
├── install.yaml
└── myservice/
    └── scripts/
        ├── setup.php
        └── configure.php
```

**Clean install.yaml:**

```yaml
name: myservice

project_files:
  - myservice/scripts/setup.php
  - myservice/scripts/configure.php

post_install_actions:
  - |
    <?php
    #ddev-description: Configure project
    require 'myservice/scripts/setup.php';
  - |
    <?php
    #ddev-description: Apply optimizations
    require 'myservice/scripts/configure.php';
```

**`myservice/scripts/setup.php`:**

```php
<?php
#ddev-generated

$projectType = $_ENV['DDEV_PROJECT_TYPE']; 
$docroot = $_ENV['DDEV_DOCROOT'];

// Exit early if not applicable
if ($projectType !== 'drupal') {
    echo "Not a Drupal project, skipping\n";
    exit(0);
}

// Perform Drupal-specific setup
$settingsFile = "/var/www/html/{$docroot}/sites/default/settings.ddev.php";
$settings = "<?php\n// DDEV-generated settings\n";
file_put_contents($settingsFile, $settings);

echo "Drupal settings configured\n";
```

#### Real-world example: ddev-redis structure

```
.ddev/
├── install.yaml
├── docker-compose.redis.yaml
└── redis/
    └── scripts/
        ├── setup-drupal-settings.php
        ├── setup-redis-optimized-config.php
        └── settings.ddev.redis.php
```

#### Mixed Bash and PHP Actions

You can combine both approaches in a single add-on:

```yaml
name: mixed-actions

pre_install_actions:
  - |
    #ddev-description: Set file permissions
    chmod +x .ddev/commands/web/mycommand
    
  - |
    <?php
    #ddev-description: Process configuration
    $projectName = $_ENV['DDEV_PROJECT'];
    echo "Processing config for: $projectName\n";
    ?>
```

## Advanced Features

### Version Constraints

Specify minimum DDEV version requirements:

```yaml
ddev_version_constraint: '>= v1.23.4'
```

### Dependencies

Declare add-on dependencies:

```yaml
dependencies:
  - redis
  - elasticsearch
```

### Template Replacements

Use environment variables in file names and content:

```yaml
project_files:
  - "settings.${DDEV_PROJECT}.php"
```

### YAML File Processing

Read project YAML files for advanced templating:

```yaml
yaml_read_files:
  config: "config.yaml"

post_install_actions:
  - |
    <?php
    // Access YAML data via templating: {{ .config.some_value }}
    ?>
```

### Error Handling

Use proper exit codes and error messages:

```php
<?php
#ddev-description: Validate requirements

if (empty($_ENV['DDEV_PROJECT'])) {
    echo "Error: DDEV environment not available\n";
    exit(1);
}

// Continue with setup...
echo "Requirements validated\n";
?>
```

## Special Directives

### Description Display

Add descriptions to your actions:

```bash
#ddev-description: Installing Redis configuration
```

```php
<?php
#ddev-description: Processing project settings
?>
```

### Warning Exit Codes

Treat specific exit codes as warnings instead of errors:

```yaml
post_install_actions:
  - |
    #ddev-warning-exit-code: 2
    #ddev-description: Optional configuration
    some-command-that-might-fail
```

## Testing Your Add-on

### Bats Testing Framework

The add-on template includes a `tests.bats` file for testing:

```bash
#!/usr/bin/env bats

@test "install add-on" {
  ddev add-on get . --project my-test
  cd my-test
  ddev restart
  # Add your tests here
}

@test "verify service is running" {
  cd my-test
  ddev exec "curl -s http://myservice:8080/health"
}
```

Run tests with:

```bash
bats tests.bats
```

### Manual Testing

1. Create a test DDEV project
2. Install your add-on locally:

   ```bash
   ddev add-on get /path/to/your/addon
   ```

3. Verify services start correctly
4. Test configuration options
5. Test removal process

## Publishing Your Add-on

### Repository Setup

1. **Test thoroughly** using the test framework
2. **Create proper releases** with semantic versioning
3. **Add the `ddev-get` label** to your GitHub repository
4. **Write clear documentation** in your README
5. **Include examples** and configuration options

### Making it Official

To become an officially supported add-on:

1. Open an issue in the [DDEV repository](https://github.com/ddev/ddev/issues/new)
2. Request upgrade to official status
3. Commit to maintaining the add-on
4. Subscribe to repository activity and be responsive

### Best Practices

- **Follow semantic versioning** for releases
- **Maintain backward compatibility** when possible
- **Test with different DDEV versions**
- **Update dependencies regularly**
- **Respond to user issues promptly**
- **Keep documentation up to date**
- **Use namespaced directories** (e.g., `myservice/scripts/` not just `scripts/`)

## Examples and References

- **Add-on Template**: [ddev-addon-template](https://github.com/ddev/ddev-addon-template)
- **Official Add-ons**: Browse examples at [addons.ddev.com](https://addons.ddev.com/)
- **Redis Add-on**: [ddev-redis](https://github.com/ddev/ddev-redis) - Good example of PHP actions with `redis/scripts/`
- **Community Examples**: [ddev-contrib](https://github.com/ddev/ddev-contrib)

## Getting Help

- **DDEV Discord**: Join [DDEV Discord](https://ddev.com/s/discord) for development support
- **GitHub Discussions**: Use [DDEV Discussions](https://github.com/ddev/ddev/discussions) for questions
- **Stack Overflow**: Tag questions with [ddev](https://stackoverflow.com/tags/ddev)

Creating DDEV add-ons is a powerful way to contribute to the DDEV ecosystem. Whether you use traditional bash actions or the new PHP-based actions, you can create sophisticated extensions that help developers worldwide.
