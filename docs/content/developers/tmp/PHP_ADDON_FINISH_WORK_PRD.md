# PHP Add-on Finish Work - Product Requirements Document

## Overview

This PRD outlines the remaining implementation work needed to complete PHP add-on functionality in DDEV. The current PHP add-on implementation has achieved feature parity with bash actions for basic operations but requires additional work to match the full bash action experience.

## Background

The PHP add-on implementation successfully completed Phase 1 with:

- Container-based PHP script execution
- Repository and .ddev directory access via bind mounts  
- Description display system matching bash actions
- YAML parsing with php-yaml extension
- Complete ddev-redis translation passing all 10 test scenarios

However, several gaps remain that prevent PHP add-ons from being production-ready.

## Goals

### Primary Goals

1. **Environment Variable Parity**: PHP actions should have access to the same environment variables as bash actions
2. **Consistent Execution Context**: PHP actions should execute in predictable working directories with relative path support
3. **Configuration Access**: PHP actions should access processed DDEV configuration data
4. **Output Control**: PHP actions should support #ddev-nodisplay and proper error handling
5. **Failure Handling**: PHP actions should report failures consistently with bash actions

### Secondary Goals

1. **Interactive Input Support**: Enable user interaction for PHP actions (research phase)

## Detailed Requirements

### 1. Standard Environment Variables (HIGH PRIORITY)

#### Problem Statement

PHP actions currently lack access to standard DDEV environment variables that bash actions receive, forcing manual configuration file parsing and absolute path usage.

#### Functional Requirements

- All PHP actions must receive the same environment variables as bash actions
- Environment variables must be available via `$_ENV` superglobal
- Variables must reflect current project configuration state

#### Required Variables

- `DDEV_APPROOT`: Project root directory (`/var/www/html`)
- `DDEV_DOCROOT`: Document root relative to project root (`web`, `public`, etc.)
- `DDEV_PROJECT_TYPE`: Project type (`drupal`, `laravel`, `php`, etc.)
- `DDEV_SITENAME`: Project name from config
- `DDEV_HOSTNAME`: Primary hostname for the project
- `DDEV_PHP_VERSION`: Current PHP version
- `DDEV_WEBSERVER_TYPE`: Webserver type (`nginx-fpm`, `apache-fpm`)
- `DDEV_DATABASE_TYPE`: Database type (`mysql`, `postgres`, `mariadb`)

#### Technical Implementation

- Modify `processPHPAction()` in `pkg/ddevapp/addons.go`
- Pass environment variables to container execution context
- Ensure variables are set before PHP script execution

#### Success Criteria

- PHP scripts can access `$_ENV['DDEV_APPROOT']` and all other standard variables
- No more manual `yaml_parse_file('config.yaml')` calls needed (using relative paths from /var/www/html/.ddev)
- Environment variables match values available to bash actions

### 2. Consistent Working Directory (COMPLETED ✅)

#### Working Directory Problem

PHP actions execute in unpredictable working directories, requiring absolute paths for all file operations and preventing relative path usage that bash actions support.

#### Working Directory Requirements

- All PHP actions must execute with working directory set to `/var/www/html/.ddev`
- Relative paths must work identically to bash actions
- File operations must use the same path patterns as bash actions

#### Working Directory Implementation

- Set working directory in `processPHPAction()` before script execution
- Use `docker exec -w /var/www/html/.ddev` or equivalent
- Ensure all PHP scripts start execution from consistent location

#### Working Directory Success Criteria

- PHP scripts can use `file_put_contents('docker-compose.redis.yaml', $content)` instead of absolute paths
- Relative paths like `../composer.json` work consistently
- File operations match bash action behavior exactly

### 3. Processed Configuration Access (MEDIUM PRIORITY)

#### Configuration Access Problem

PHP actions can only access raw config.yaml files, not the processed configuration that includes computed values, global settings, and runtime state that bash actions access via `ddev debug configyaml`.

#### Configuration Access Requirements

- PHP actions must access processed configuration data
- Global DDEV configuration must be available
- Computed values (like resolved hostnames, ports) must be accessible

#### Technical Implementation Options

1. **JSON/YAML File Mount**: Write processed config to temporary files and mount into containers
2. **Environment Variable Expansion**: Include more computed values in environment variables  
3. **PHP Helper Functions**: Create utility functions that provide config access

#### Preferred Approach

Mount processed configuration as JSON files accessible via relative paths:

```php
$processedConfig = json_decode(file_get_contents('ddev_processed_config.json'), true);
$globalConfig = json_decode(file_get_contents('ddev_global_config.json'), true);
```

#### Configuration Access Success Criteria

- PHP scripts can access all values available via `ddev debug configyaml`
- Global configuration settings are available
- No difference in configuration access between PHP and bash actions

### 4. Output Control Implementation (MEDIUM PRIORITY)

#### Output Control Problem

PHP actions do not support `#ddev-nodisplay` directive and lack consistent error handling, making them unsuitable for quiet operations and automated scripts.

#### Output Control Requirements

- `#ddev-nodisplay` directive must suppress step output
- Error handling must match bash action behavior
- Exit codes must be properly reported
- User feedback must be consistent

#### Output Control Implementation

- Parse `#ddev-nodisplay` in `processPHPAction()` before execution
- Suppress description display when directive is present
- Capture and report exit codes properly
- Ensure error messages are displayed consistently

#### Examples

```php
<?php
#ddev-nodisplay:Skip Redis optimization prompts
// This action runs silently without "👍" output
?>
```

#### Output Control Success Criteria

- Actions with `#ddev-nodisplay` produce no visible output on success
- Error messages are displayed even when nodisplay is active
- Exit codes match bash action behavior
- Test scenarios validate error handling works correctly

### 5. Step Failure Reporting (MEDIUM PRIORITY)

#### Failure Reporting Problem

PHP action failures may not be properly reported or may produce inconsistent error output compared to bash actions.

#### Failure Reporting Requirements

- PHP script failures must be detected and reported
- Error messages must be visible to users
- Exit codes must halt installation on failure
- Error output must match bash action format

#### Test Requirements

- Create test scenarios that intentionally fail
- Verify error messages are displayed properly
- Confirm installation halts on PHP script failure
- Compare error output with equivalent bash action failures

#### Failure Reporting Success Criteria

- Failed PHP actions produce clear error messages
- Installation process stops on PHP script failure
- Error reporting matches bash action behavior exactly
- Test coverage includes failure scenarios

### 6. Interactive User Input (LOW PRIORITY - RESEARCH)

#### Interactive Input Problem

Some add-ons require user interaction (e.g., ddev-php-patch-build asking for PHP version), but container-based PHP execution may not support interactive input.

#### Research Requirements

**Investigation Tasks:**

- Analyze existing interactive bash add-ons (ddev-php-patch-build)
- Determine container limitations for interactive input
- Research Docker/container solutions for user interaction
- Evaluate terminal forwarding and TTY allocation options

**Potential Approaches:**

1. **Pre-execution Prompts**: Collect input before container execution
2. **Environment Variable Passing**: Convert prompts to environment variables
3. **TTY Forwarding**: Enable interactive terminal in containers
4. **Hybrid Approach**: Use bash for interactive parts, PHP for processing

#### Interactive Input Success Criteria

- Document feasibility of interactive PHP actions
- Provide implementation recommendation
- Create prototype if feasible
- Define limitations and workarounds

## Testing Strategy

### Unit Tests

- Test environment variable passing in `processPHPAction()`
- Validate working directory setting
- Test configuration file mounting and access
- Verify #ddev-nodisplay parsing and handling

### Integration Tests

- Re-run ddev-redis PHP translation with new features
- Test all 10 scenarios continue to pass
- Validate improvement in code simplicity and reliability
- Test failure scenarios and error reporting

### User Acceptance Tests

- Create add-ons using new features
- Validate developer experience improvements
- Test production add-on translations
- Confirm feature parity with bash actions achieved

## Success Metrics

### Quantitative Metrics

- All 10 ddev-redis test scenarios pass with simplified code
- Reduction in absolute path usage in PHP add-ons
- Elimination of manual config file parsing
- Error handling test coverage at 100%

### Qualitative Metrics

- PHP add-on development experience matches bash actions
- Add-on developers prefer PHP for complex operations
- Configuration access is intuitive and consistent
- Error messages are clear and actionable

## Definition of Done

PHP add-on implementation is complete when:

1. ✅ All environment variables available to bash actions are available to PHP actions
2. ✅ PHP actions execute in consistent working directory with relative path support
3. ✅ PHP actions can access processed configuration data
4. ✅ #ddev-nodisplay directive works correctly (inherited from bash actions)
5. ✅ Error handling and reporting matches bash action behavior with strict mode
6. ✅ All existing PHP add-on tests continue to pass
7. ✅ Documentation is updated with new capabilities
8. ✅ PHP removal actions work without requiring running project
9. ✅ Single-container optimization implemented for performance
10. ✅ Comprehensive syntax validation with include/require support
11. ⏳ Interactive input feasibility is documented (implementation optional)

## Appendix

### Related Documentation

- [PHP Add-on Translation Guide](PHP_ADDON_TRANSLATION_GUIDE.md)
- [PHP Add-on Translation Strategy](PHP_ADDON_TRANSLATION_STRATEGY.md)
- [PHP Add-ons User Guide](PHP_ADDON_GUIDE.md)

### Reference Implementation

- ddev-redis PHP translation: `/Users/rfay/workspace/php-experiments/ddev-redis-php`
- Test scenarios: All 10 original bash test cases passing
- GitHub Actions testing: Dynamic artifact fetching with fallback mechanisms

### Final Implementation Updates ✅ **COMPLETED**

**Docker Compose Container Architecture (Task 11)**:

- **Status**: ✅ **COMPLETED** - Migrated from `RunSimpleContainer` to docker-compose execution
- **Benefits Achieved**:
    - Improved container lifecycle management and cleanup
    - Proper DDEV container labeling (`com.ddev.site-name`, `com.ddev.approot`)
    - Integrated host.docker.internal setup for debugging support
    - Consistency with DDEV's standard container management
    - In-memory compose projects with no temporary files
- **Testing**: All existing PHP add-on tests continue to pass
- **Compatibility**: Fully backward compatible with existing PHP add-ons
