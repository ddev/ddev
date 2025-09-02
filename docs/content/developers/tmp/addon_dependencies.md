# Addon Dependencies Implementation Plan

This document outlines the implementation plan for [GitHub Issue #5337](https://github.com/ddev/ddev/issues/5337) - automatic addon dependency resolution and installation.

## Overview

Currently, addon dependencies declared in `install.yaml` block installation if the dependency is not already installed. This plan changes that behavior to automatically install missing dependencies recursively.

Additionally, this plan adds support for **runtime-discovered dependencies** - dependencies that can only be determined during the addon installation process by analyzing the project structure.

## Current Behavior vs. New Behavior

### Current Behavior

```bash
ddev get addon-with-dependencies
# ERROR: The add-on 'my-addon' declares a dependency on 'ddev-redis'; Please ddev add-on get ddev-redis first.
```

### New Behavior

```bash
# Default: Install with dependencies
ddev get addon-with-dependencies
# Installing dependency: ddev/ddev-redis
# Installed ddev-redis:v1.0.0 from ddev/ddev-redis
# Installing my-addon...
# Installed my-addon:v1.0.0

# Skip dependencies (advanced usage)
ddev get addon-with-dependencies --no-dependencies
# ERROR: The add-on 'my-addon' declares a dependency on 'ddev-redis'; Please ddev add-on get ddev-redis first.
```

## Implementation Plan

### Phase 1: Static Dependencies (Recursive Installation)

#### 1.1 Modify Dependency Handling in `cmd/ddev/cmd/addon-get.go`

**Current blocking code (lines 159-171):**

```go
// Check to see if any dependencies are missing
if len(s.Dependencies) > 0 {
    // Read in full existing registered config
    m, err := ddevapp.GatherAllManifests(app)
    if err != nil {
        util.Failed("Unable to gather manifests: %v", err)
    }
    for _, dep := range s.Dependencies {
        if _, ok := m[dep]; !ok {
            util.Failed("The add-on '%s' declares a dependency on '%s'; Please ddev add-on get %s first.", s.Name, dep, dep)
        }
    }
}
```

**Replace with recursive installation:**

```go
// Install missing dependencies recursively
if len(s.Dependencies) > 0 {
    err = installDependencies(app, s.Dependencies, verbose)
    if err != nil {
        util.Failed("Unable to install dependencies for '%s': %v", s.Name, err)
    }
}
```

#### 1.2 Add New Functions to `cmd/ddev/cmd/addon-get.go`

```go
// installDependencies installs a list of dependencies, checking if they're already installed
func installDependencies(app *ddevapp.DdevApp, dependencies []string, verbose bool) error {
    m, err := ddevapp.GatherAllManifests(app)
    if err != nil {
        return fmt.Errorf("unable to gather manifests: %w", err)
    }

    for _, dep := range dependencies {
        if _, exists := m[dep]; !exists {
            util.Success("Installing missing dependency: %s", dep)
            err = installAddonRecursive(app, dep, verbose)
            if err != nil {
                return fmt.Errorf("failed to install dependency '%s': %w", dep, err)
            }
            // Refresh manifest cache after installation
            m, _ = ddevapp.GatherAllManifests(app)
        } else if verbose {
            util.Success("Dependency '%s' is already installed", dep)
        }
    }
    return nil
}

// Global variable to track installation stack for circular dependency detection
var installStack []string

// installAddonRecursive installs an addon and its dependencies recursively
func installAddonRecursive(app *ddevapp.DdevApp, addonName string, verbose bool) error {
    // Check for circular dependencies
    for _, stackItem := range installStack {
        if stackItem == addonName {
            return fmt.Errorf("circular dependency detected: %s", 
                strings.Join(append(installStack, addonName), " -> "))
        }
    }
    
    installStack = append(installStack, addonName)
    defer func() {
        installStack = installStack[:len(installStack)-1]
    }()
    
    return installAddonFromGitHub(app, addonName, verbose)
}

// installAddonFromGitHub handles GitHub-based addon installation (extracted from main Run function)
func installAddonFromGitHub(app *ddevapp.DdevApp, addonName string, verbose bool) error {
    // Parse owner/repo from addonName
    parts := strings.Split(addonName, "/")
    if len(parts) != 2 {
        return fmt.Errorf("invalid addon name format, expected 'owner/repo': %s", addonName)
    }
    
    owner := parts[0]
    repo := parts[1]
    ctx := context.Background()
    
    // Get GitHub release (same logic as main Run function)
    client := ddevgh.GetGithubClient(ctx)
    releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
    if err != nil {
        var rate github.Rate
        if resp != nil {
            rate = resp.Rate
        }
        return fmt.Errorf("unable to get releases for %v: %v\nresp.Rate=%v", repo, err, rate)
    }
    if len(releases) == 0 {
        return fmt.Errorf("no releases found for %v", repo)
    }
    
    // Always use latest release (index 0)
    releaseItem := 0
    
    tarballURL := releases[releaseItem].GetTarballURL()
    downloadedRelease := releases[releaseItem].GetTagName()
    
    // Download and install (reuse existing logic from main function)
    return installAddonFromTarball(app, tarballURL, owner, repo, downloadedRelease, verbose)
}

// installAddonFromTarball handles the actual installation process (extracted from main Run function)
func installAddonFromTarball(app *ddevapp.DdevApp, tarballURL, owner, repo, downloadedRelease string, verbose bool) error {
    // Extract tarball
    extractedDir, cleanup, err := archive.DownloadAndExtractTarball(tarballURL, true)
    if err != nil {
        return fmt.Errorf("unable to download %v: %v", tarballURL, err)
    }
    defer cleanup()
    
    // Parse install.yaml
    yamlFile := filepath.Join(extractedDir, "install.yaml")
    yamlContent, err := fileutil.ReadFileIntoString(yamlFile)
    if err != nil {
        return fmt.Errorf("unable to read %v: %v", yamlFile, err)
    }
    var s ddevapp.InstallDesc
    err = yaml.Unmarshal([]byte(yamlContent), &s)
    if err != nil {
        return fmt.Errorf("unable to parse %v: %v", yamlFile, err)
    }
    
    // Install dependencies recursively
    if len(s.Dependencies) > 0 {
        err = installDependencies(app, s.Dependencies, verbose)
        if err != nil {
            return fmt.Errorf("unable to install dependencies for '%s': %v", s.Name, err)
        }
    }
    
    // Continue with normal installation process...
    // (Copy existing logic for pre-install, files, post-install, manifest creation)
    
    return nil
}
```

### Phase 2: Runtime-Discovered Dependencies

#### 2.1 Modify `ProcessAddonAction` Signature

**In `pkg/ddevapp/addons.go`, modify:**

```go
// OLD:
func ProcessAddonAction(action string, installDesc InstallDesc, app *DdevApp, verbose bool) error

// NEW:  
func ProcessAddonAction(action string, installDesc InstallDesc, app *DdevApp, extractedDir string, verbose bool) error
```

**Update internal function calls:**

```go
func processBashHostAction(action string, installDesc InstallDesc, app *DdevApp, extractedDir string, verbose bool) error
func processPHPAction(action string, installDesc InstallDesc, app *DdevApp, extractedDir string, verbose bool) error
```

#### 2.2 Update All ProcessAddonAction Calls

**In `addon-get.go`, update calls:**

```go
// Pre-install actions
for i, action := range s.PreInstallActions {
    err = ddevapp.ProcessAddonAction(action, s, app, extractedDir, verbose)
    if err != nil {
        // ... error handling
    }
}

// Post-install actions
for i, action := range s.PostInstallActions {
    err = ddevapp.ProcessAddonAction(action, s, app, extractedDir, verbose)
    if err != nil {
        // ... error handling  
    }
}
```

**Also update calls in `pkg/ddevapp/addons.go` (RemoveAddon function):**

```go
err = ProcessAddonAction(action, InstallDesc{}, app, "", verbose)
```

#### 2.3 Add Runtime Dependencies File Support

**In `addon-get.go`, after pre-install actions:**

```go
// After pre-install actions, check for runtime-discovered dependencies
runtimeDepsFile := filepath.Join(extractedDir, ".runtime-deps")
if fileutil.FileExists(runtimeDepsFile) {
    runtimeDeps, err := parseRuntimeDependencies(runtimeDepsFile)
    if err != nil {
        util.Warning("Failed to parse runtime dependencies: %v", err)
    } else if len(runtimeDeps) > 0 {
        util.Success("Installing discovered dependencies: %s", strings.Join(runtimeDeps, ", "))
        err = installDependencies(app, runtimeDeps, verbose)
        if err != nil {
            util.Failed("Failed to install runtime dependencies: %v", err)
        }
    }
    // File gets cleaned up automatically when extractedDir is cleaned up
}

// parseRuntimeDependencies reads dependency names from a file, one per line
func parseRuntimeDependencies(filename string) ([]string, error) {
    content, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    deps := []string{}
    lines := strings.Split(strings.TrimSpace(string(content)), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            deps = append(deps, line)
        }
    }
    return deps, nil
}
```

## Usage Examples

### Command Line Flag Usage

```bash
# Default behavior - install addon with all dependencies
ddev get ddev-upsun
ddev get ddev/ddev-redis --version v1.0.4

# Skip dependency installation (advanced usage)
ddev get ddev-upsun --no-dependencies
ddev get ddev-complex-addon --no-dependencies --verbose

# Use cases for --no-dependencies:
# - Testing scenarios where you don't want full dependency tree
# - Debugging dependency issues by controlling installation order
# - Cases where dependencies are installed via different means
# - Reproducing the old blocking behavior for troubleshooting
```

### Static Dependencies in install.yaml

```yaml
name: my-complex-addon
dependencies:
  - ddev/ddev-redis
  - ddev/ddev-solr
  
project_files:
  - docker-compose.my-addon.yaml
  
post_install_actions:
  - echo "Complex addon installed with Redis and Solr"
```

### Runtime Dependencies Discovery

**For a Platform.sh/Upsun addon:**

```yaml
name: ddev-upsun
dependencies: []

pre_install_actions:
  - |
    #ddev-description: Analyzing Upsun services for dependencies
    
    # Check for Redis in platform services
    if [ -f .platform/services.yaml ] && grep -q "redis" .platform/services.yaml; then
        echo "ddev/ddev-redis" >> .runtime-deps
    fi
    
    # Check for Solr in platform services  
    if [ -f .platform/services.yaml ] && grep -q "solr" .platform/services.yaml; then
        echo "ddev/ddev-solr" >> .runtime-deps
    fi
    
    # Check for Elasticsearch
    if [ -f .platform/services.yaml ] && grep -q "elasticsearch" .platform/services.yaml; then
        echo "ddev/ddev-elasticsearch" >> .runtime-deps
    fi
    
    # Check for Redis in environment variables
    if [ -f .env ] && grep -q "REDIS_URL" .env; then
        echo "ddev/ddev-redis" >> .runtime-deps
    fi

project_files:
  - docker-compose.upsun.yaml
  - upsun/config.yaml
```

**PHP-based discovery example:**

```yaml
name: ddev-upsun-advanced
dependencies: []

pre_install_actions:
  - |
    <?php
    #ddev-description: Advanced Upsun service analysis
    
    $deps = [];
    
    // Parse .platform/services.yaml
    if (file_exists('.platform/services.yaml')) {
        $services = yaml_parse_file('.platform/services.yaml');
        foreach ($services as $name => $config) {
            if (isset($config['type'])) {
                if (strpos($config['type'], 'redis') !== false) {
                    $deps[] = 'ddev/ddev-redis';
                }
                if (strpos($config['type'], 'solr') !== false) {
                    $deps[] = 'ddev/ddev-solr';
                }
                if (strpos($config['type'], 'elasticsearch') !== false) {
                    $deps[] = 'ddev/ddev-elasticsearch';
                }
            }
        }
    }
    
    // Check Drupal-specific indicators
    if (file_exists('web/modules/contrib/redis')) {
        $deps[] = 'ddev/ddev-redis';
    }
    
    if (file_exists('config/sync/search_api.server.solr.yml')) {
        $deps[] = 'ddev/ddev-solr';
    }
    
    // Write discovered dependencies
    if (!empty($deps)) {
        file_put_contents('.runtime-deps', implode("\n", array_unique($deps)));
    }
```

## Testing Strategy

### Unit Tests

**Add to `cmd/ddev/cmd/addon-get_test.go`:**

1. **TestAddonGetWithDependencies** - Test static dependency installation
2. **TestAddonGetWithRuntimeDependencies** - Test runtime dependency discovery
3. **TestAddonGetCircularDependencies** - Test circular dependency detection
4. **TestAddonGetMixedDependencies** - Test combination of static and runtime dependencies
5. **TestAddonGetDependencyInstallFailure** - Test error handling when dependency installation fails

### Integration Tests

**Create test addon structures in `cmd/ddev/cmd/testdata/`:**

1. **TestAddonWithDeps/** - Addon with static dependencies
2. **TestAddonWithRuntimeDeps/** - Addon that discovers runtime dependencies
3. **TestAddonCircular/** - Addons with circular dependencies for negative testing

### Example Test Addon Structure

```
testdata/TestAddonRuntimeDeps/
├── install.yaml
├── pre_install_actions/
│   └── discover-deps.sh
└── docker-compose.test.yaml

# install.yaml:
name: test-runtime-deps
dependencies: []
pre_install_actions:
  - |
    # Simulate runtime dependency discovery
    if [ -f .platform/services.yaml ]; then
        echo "ddev/ddev-redis" >> .runtime-deps
    fi
project_files:
  - docker-compose.test.yaml
```

## Implementation Steps

### Step 1: Static Dependencies

1. Modify `addon-get.go` dependency checking logic
2. Extract GitHub installation logic into reusable functions  
3. Add circular dependency detection
4. Add tests for static dependency scenarios

### Step 2: Runtime Dependencies

1. Modify `ProcessAddonAction` signature to include `extractedDir`
2. Update all `ProcessAddonAction` calls
3. Add runtime dependency file parsing
4. Add tests for runtime dependency scenarios

### Step 3: Integration and Testing

1. Create comprehensive test addon structures
2. Test both static and runtime dependencies together
3. Test error scenarios and circular dependencies
4. Update documentation with examples

### Step 4: Add --no-dependencies Flag

**Add flag to AddonGetCmd in `addon-get.go`:**

```go
AddonGetCmd.Flags().Bool("no-dependencies", false, "Skip automatic dependency installation")
```

**Update installation logic:**

```go
skipDependencies := false
if cmd.Flags().Changed("no-dependencies") {
    skipDependencies = cmd.Flag("no-dependencies").Value.String() == "true"
}

// Install static dependencies (unless skipped)
if len(s.Dependencies) > 0 && !skipDependencies {
    err = installDependencies(app, s.Dependencies, verbose)
    if err != nil {
        util.Failed("Unable to install dependencies for '%s': %v", s.Name, err)
    }
}

// Runtime dependency discovery (unless skipped)
if !skipDependencies {
    runtimeDepsFile := filepath.Join(extractedDir, ".runtime-deps")
    if fileutil.FileExists(runtimeDepsFile) {
        // ... runtime dependency logic
    }
}
```

### Step 5: Optional Future Enhancements

1. Add dependency visualization (`ddev addon list --tree`)
2. Add dependency update checking
3. Add performance optimizations for large dependency trees
4. Add version constraint support

## Backward Compatibility

This change is **backward compatible**:

- Existing addons without dependencies continue to work unchanged
- Existing addons with static dependencies work but now install dependencies automatically instead of failing
- No changes to `install.yaml` format required
- Runtime dependency discovery is opt-in via pre-install actions

## Error Handling

1. **Circular Dependencies**: Detect and fail with clear error message showing the dependency chain
2. **Missing Dependencies**: If a dependency cannot be found or installed, fail the entire installation
3. **Network Issues**: Standard GitHub API error handling with rate limiting information
4. **Malformed Runtime Deps**: Log warnings but continue installation
5. **Permission Issues**: Clear error messages for file system permission problems

## Performance Considerations

1. **Manifest Caching**: Cache manifest lookups during recursive installations
2. **GitHub API Rate Limiting**: Reuse GitHub client and respect rate limits
3. **Parallel Installations**: Current implementation is sequential; parallel installation could be added later
4. **Dependency Resolution**: Use depth-first dependency resolution to minimize redundant operations

This implementation provides a robust, extensible foundation for both static and runtime addon dependency management while maintaining backward compatibility and clear error handling.
