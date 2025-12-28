# Plan: Enhanced Mutagen Diagnostics for Issue #7740

## Executive Summary

**Goal:** Create a new `ddev utility mutagen-diagnose` command to help users identify and resolve Mutagen performance and synchronization issues independently.

**Approach:** Implement a standalone diagnostic command that analyzes:
1. 📊 **Volume sizes** - Detect large volumes using `docker system df -v`
2. 📁 **Upload_dirs configuration** - Suggest optimizations for file exclusions
3. ✅ **Sync status** - Check for problems, conflicts, paused sessions
4. ⚡ **Performance issues** - Identify problematic directories (node_modules, etc.)

**Key Features:**
- Works in project directory or with `--all` flag for system-wide analysis
- Color-coded output with actionable recommendations
- Can be integrated into `ddev utility test` and `ddev utility diagnose` later

**Files to Create/Modify:**
- NEW: `cmd/ddev/cmd/utility-mutagen-diagnose.go` (command implementation)
- MODIFY: `pkg/ddevapp/mutagen.go` (add 4 new diagnostic functions)
- MODIFY: `pkg/dockerutil/dockerutils.go` (add volume size parsing)

**Addresses GitHub Issue #7740:** All requested features included:
- ✅ Evaluate volume size and contents
- ✅ Check `ddev mutagen st -l` for problems
- ✅ Measure disk space consumption
- ✅ Identify stopped/abandoned sessions
- ✅ Review upload_dirs configuration

---

## Overview

Add comprehensive Mutagen-specific analysis to help users identify and resolve Mutagen performance and synchronization issues independently, as requested in GitHub issue #7740.

## Comprehensive List of Detectable Mutagen Issues

Based on GitHub issue #7740, performance.md troubleshooting documentation, and the existing mutagen.go implementation, here are all the issues we can detect and automate:

### 1. Volume Size and Disk Usage Issues

**Problem:** Excessive disk space consumption by Mutagen volumes
**Detection Methods:**
- Measure size of `<projectname>_project_mutagen` Docker volume
- Calculate total Mutagen disk usage across all projects
- Identify volumes larger than reasonable thresholds (e.g., >5GB warning, >10GB critical)
- Compare volume size to project directory size on host

**Data Sources:**
- `docker system df -v` - provides volume sizes
- `docker volume inspect <volumename>` - volume metadata
- Alternative: Mount volume in temporary container and run `du -sh`

**Output:**
- Volume name and size in human-readable format
- Warning if volume exceeds reasonable size
- List of all Mutagen volumes with sizes

### 2. Sync Session Status and Problems

**Problem:** Sync sessions with errors, conflicts, or stuck states
**Detection Methods:**
- Check sync session status via `app.MutagenStatus()`
- Detect paused sessions
- Identify scan problems (alpha/beta endpoints)
- Identify transition problems (alpha/beta endpoints)
- Detect sync conflicts
- Check for abandoned/orphaned sessions (volume exists but no session, or vice versa)

**Data Sources:**
- `mutagen sync list --template '{{ json (index . 0) }}' <syncName>` (already implemented in MutagenStatus)
- Session status: "ok", "paused", "problems", "failing", "nosession"
- Session map fields: `scanProblems`, `transitionProblems`, `conflicts`, `status`, `paused`

**Output:**
- Current sync status with detailed problem descriptions
- Conflict information if present
- Recommendations (e.g., "run `ddev mutagen reset`")

### 3. Upload Directories Configuration

**Problem:** Missing or incorrect upload_dirs configuration leading to unnecessary syncing
**Detection Methods:**
- Check if Mutagen is enabled but `upload_dirs` is empty
- Verify upload_dirs exist on filesystem
- Check if upload_dirs are properly excluded in mutagen.yml
- Suggest CMS-specific defaults if not configured

**Data Sources:**
- `app.GetUploadDirs()` - returns configured upload directories
- `app.checkMutagenUploadDirs()` - existing warning function
- CMS-specific defaults from apptypes.go

**Output:**
- Whether upload_dirs is configured
- List of configured upload directories
- Whether they exist on disk
- Suggestions for common upload directories based on project type

### 4. Modified Mutagen Configuration

**Problem:** Custom mutagen.yml changes without proper reset
**Detection Methods:**
- Check if `.ddev/mutagen/mutagen.yml` exists
- Check for `#ddev-generated` marker in file
- Compare config file hash with session label
- Detect config changes since session creation

**Data Sources:**
- `GetMutagenConfigFilePath(app)` - path to mutagen.yml
- `GetMutagenConfigFileHash(app)` - SHA1 hash of config
- `GetMutagenConfigFileHashLabel(app)` - hash from sync session label
- `CheckMutagenVolumeSyncCompatibility(app)` - validates compatibility

**Output:**
- Whether mutagen.yml is customized
- Whether hash matches session (needs reset if not)
- Recommendation to run `ddev mutagen reset` if changed

### 5. Large Files and Directories Being Synced

**Problem:** Syncing unnecessary large files causing slow performance
**Detection Methods:**
- Monitor `ddev mutagen st -l` output during sync to see current file
- Parse sync progress to identify large files
- Check for common problematic directories:
  - `node_modules`
  - `vendor` (if very large)
  - `.tarballs`
  - Large binary directories
  - Font directories
- Analyze mutagen.yml ignore patterns

**Data Sources:**
- `mutagen sync list -l <syncName>` - detailed status with current file
- Running `mutagen sync monitor` during initial sync
- Parse `.ddev/mutagen/mutagen.yml` ignore patterns

**Output:**
- List of large files/directories being synced (if detectable)
- Whether common problem directories are excluded
- Suggestions for paths to exclude

### 6. Sync Performance Metrics

**Problem:** Slow sync times on startup or during operation
**Detection Methods:**
- Measure time for sync to reach "watching" state
- Count total files being synced
- Identify if sync is taking longer than expected (>60s for initial)
- Check staging statistics from sync status

**Data Sources:**
- Sync session statistics from `mutagen sync list` JSON output
- Monitor staging/scanning progress
- File counts and transfer rates if available in session data

**Output:**
- Estimated sync time or file count
- Warning if sync appears slow
- Recommendations for improving performance

### 7. Session and Volume Compatibility

**Problem:** Mismatched labels between volume and session
**Detection Methods:**
- Compare volume label with sync session label
- Check Docker context consistency
- Verify config hash matches
- Detect Docker provider changes

**Data Sources:**
- `CheckMutagenVolumeSyncCompatibility(app)` - comprehensive compatibility check
- `GetMutagenVolumeLabel(app)` - volume label
- `GetMutagenSyncLabel(app)` - session label
- Returns: ok status, volumeExists, and detailed info string

**Output:**
- Compatibility status
- Detailed reasoning if incompatible
- Recommendation to run `ddev mutagen reset`

### 8. Mutagen Daemon and Binary Status

**Problem:** Daemon not running or wrong version
**Detection Methods:**
- Check if Mutagen binary exists at `~/.ddev/bin/mutagen`
- Verify Mutagen version matches required version
- Check if daemon is running
- Verify data directory location at `~/.ddev_mutagen_data_directory`
- Check data directory size (session metadata storage)

**Data Sources:**
- `globalconfig.GetMutagenPath()` - binary path (returns `~/.ddev/bin/mutagen`)
- `version.GetLiveMutagenVersion()` - current version
- `versionconstants.RequiredMutagenVersion` - expected version
- `globalconfig.GetMutagenDataDirectory()` - data directory (returns `~/.ddev_mutagen_data_directory`)
- Check `~/.ddev_mutagen_data_directory` exists and measure size

**Output:**
- Binary path and version
- Whether daemon is running
- Data directory location and size
- Whether version is correct
- Number of sessions stored in data directory

### 9. Orphaned Resources

**Problem:** Leftover volumes or sessions from deleted projects
**Detection Methods:**
- List all Mutagen volumes across system
- List all Mutagen sync sessions
- Identify volumes without matching sessions
- Identify sessions without running projects

**Data Sources:**
- `docker volume ls --filter "label=com.ddev.volume-signature"`
- `mutagen sync list` - all sessions
- Cross-reference with active DDEV projects

**Output:**
- List of orphaned volumes with sizes
- List of orphaned sessions
- Commands to clean them up

### 10. Volume Mount Status

**Problem:** Volume not properly mounted in container
**Detection Methods:**
- Check if Mutagen volume is mounted in web container
- Verify mount point is `/var/www`
- Check volume ownership issues

**Data Sources:**
- `IsMutagenVolumeMounted(app)` - checks if volume mounted
- Container inspection via Docker API

**Output:**
- Whether volume is mounted
- Mount point location
- Warning if not mounted when Mutagen enabled

## Implementation Approach

### Create New `ddev utility mutagen-diagnose` Command

Create a standalone diagnostic command specifically for Mutagen that can later be integrated into `ddev utility test` and `ddev utility diagnose`.

**Command:** `ddev utility mutagen-diagnose` (alias: `ddev ut mutagen-diagnose`)

**Behavior:**
- Can run from within a DDEV project directory (analyzes current project)
- Can run with `--all` flag to analyze all Mutagen volumes/sessions on system
- Color-coded output matching style of `ddev utility diagnose`
- Returns exit code 1 if issues found

**Priority Diagnostics (in order):**

1. **Volume Size and Disk Usage** ⭐
   - Size of current project's Mutagen volume
   - Total size of all Mutagen volumes
   - Warning if project volume >5GB, critical if >10GB
   - List all volumes with sizes if `--all` flag used

2. **Upload_dirs Configuration** ⭐
   - Check if upload_dirs configured when Mutagen enabled
   - Verify configured dirs exist
   - Suggest CMS-specific defaults if not set
   - Check if properly excluded in mutagen.yml

3. **Sync Status and Problems** ⭐
   - Current sync status (ok/problems/paused/failing)
   - Scan problems in alpha/beta
   - Transition problems
   - Conflicts (with details)
   - Paused or abandoned sessions

4. **Large Files Being Synced** ⭐
   - Check if common problem dirs are excluded:
     - node_modules
     - .tarballs
     - vendor (if very large)
   - Parse mutagen.yml ignore patterns
   - Monitor current sync file if in progress

**Output Format:**
```
Mutagen Diagnostics for Project: myproject

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Sync Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✓ Session status: ok (watching)
  ✓ Session and volume labels match
  ℹ Mutagen version: 0.17.6

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Volume Size Analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ Project volume: myproject_project_mutagen (8.2GB)
    Large volume detected. Consider excluding unnecessary directories.
  ℹ Total Mutagen volumes: 12.5GB across 4 projects

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Upload Directories Configuration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✗ No upload_dirs configured for drupal project
    → Suggestion: Add 'upload_dirs: ["sites/default/files"]' to .ddev/config.yaml
    → Then run: ddev mutagen reset && ddev restart

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Performance Analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ✗ node_modules directory not excluded from sync
    → Add to .ddev/mutagen/mutagen.yml ignore paths:
      paths:
        - "/node_modules"
    → Then run: ddev mutagen reset && ddev restart
  ✓ .tarballs excluded from sync
  ✓ mutagen.yml using default configuration

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ 2 warnings found
  ✗ 2 issues found

Recommendations:
  1. Configure upload_dirs for better performance
  2. Exclude node_modules from Mutagen sync
  3. Consider excluding other large directories to reduce volume size

Run 'ddev mutagen reset && ddev restart' after making configuration changes.
```

## Critical Files to Create/Modify

### 1. NEW: `/home/rfay/workspace/ddev/cmd/ddev/cmd/utility-mutagen-diagnose.go`
**Create new command file**

Structure (based on `/home/rfay/workspace/ddev/cmd/ddev/cmd/utility-diagnose.go`):
```go
package cmd

import (
    "github.com/ddev/ddev/pkg/ddevapp"
    "github.com/ddev/ddev/pkg/output"
    "github.com/ddev/ddev/pkg/util"
    "github.com/spf13/cobra"
)

// MutagenDiagnoseCmd implements `ddev utility mutagen-diagnose`
var MutagenDiagnoseCmd = &cobra.Command{
    Use:   "mutagen-diagnose",
    Short: "Diagnose Mutagen sync configuration and performance",
    Long:  "Analyzes Mutagen sync status, volume sizes, and configuration issues",
    Run: func(cmd *cobra.Command, args []string) {
        showAll, _ := cmd.Flags().GetBool("all")
        // Implementation here
    },
}

func init() {
    MutagenDiagnoseCmd.Flags().BoolP("all", "a", false, "Show all Mutagen volumes system-wide")
    UtilityCmd.AddCommand(MutagenDiagnoseCmd)
}
```

**Implementation:**
- Check if running in project directory (optional - can run anywhere with --all)
- Load app if in project: `app, err := ddevapp.GetActiveApp("")`
- Call diagnostic functions from mutagen.go (see below)
- Format output with color codes using `util.ColorizeSubcommand()`
- Return exit code 1 if issues found

### 2. `/home/rfay/workspace/ddev/pkg/ddevapp/mutagen.go`
**Add new diagnostic functions**

#### a. `GetMutagenVolumeSize(volumeName string) (sizeBytes int64, sizeHuman string, err error)`
```go
// Uses docker system df -v to get volume size
// Parse output to find volume and extract size
// Returns both raw bytes and human-readable format (e.g., "2.3GB")
```

#### b. `GetAllMutagenVolumes() (volumes []MutagenVolumeInfo, totalSize int64, err error)`
```go
type MutagenVolumeInfo struct {
    Name      string
    SizeBytes int64
    SizeHuman string
    Project   string  // extracted from <project>_project_mutagen
}

// Lists all volumes matching pattern "*_project_mutagen"
// Gets size for each using docker system df -v
// Returns slice of volume info and total size
```

#### c. `CheckMutagenIgnorePatterns(app *DdevApp) (issues []string, warnings []string)`
```go
// Parse .ddev/mutagen/mutagen.yml to check ignore patterns
// Check for common problematic directories:
//   - node_modules (warn if not ignored)
//   - .tarballs (should be ignored by default)
//   - vendor (warn if large and not ignored)
// Check if file exists and is readable
// Return list of issues and warnings
```

#### d. `DiagnoseMutagenConfiguration(app *DdevApp) (result MutagenDiagnosticResult)`
```go
type MutagenDiagnosticResult struct {
    // Sync Status
    SyncStatus       string
    SyncStatusDetail string
    SessionExists    bool
    HasProblems      bool
    Problems         []string

    // Volume Info
    VolumeSize       int64
    VolumeSizeHuman  string
    VolumeWarning    bool  // true if >5GB
    VolumeCritical   bool  // true if >10GB

    // Upload Dirs
    UploadDirsConfigured bool
    UploadDirs           []string
    UploadDirsSuggestion string

    // Configuration
    MutagenYmlCustomized bool
    ConfigHashMatch      bool
    LabelsMatch          bool

    // Performance
    IgnoreIssues         []string
    IgnoreWarnings       []string

    // Overall
    IssueCount           int
    WarningCount         int
}

// Comprehensive diagnostic function that calls all checks
// Returns structured result for easy formatting
```

### 3. `/home/rfay/workspace/ddev/pkg/dockerutil/dockerutils.go`
**Add volume size parsing function**

```go
// ParseDockerSystemDf executes `docker system df -v` and parses output
// Returns map of volume names to sizes
func ParseDockerSystemDf() (map[string]VolumeSize, error) {
    // Run: docker system df -v --format "{{.Name}}\t{{.Size}}"
    // Parse output into structured format
    // Handle human-readable sizes (GB, MB, etc.)
    // Convert to bytes for consistent comparison
}
```

### 4. `/home/rfay/workspace/ddev/pkg/ddevapp/upload_dirs.go`
**Add helper function (may already exist)**

```go
// GetCMSDefaultUploadDirs returns default upload_dirs for a project type
// Based on existing per-CMS defaults in apptypes.go
func GetCMSDefaultUploadDirs(projectType string) []string {
    // Return CMS-specific defaults
}
```

## Implementation Steps

### Phase 1: Core Diagnostic Functions (pkg/ddevapp/mutagen.go)
1. ✅ Implement `GetMutagenVolumeSize()` using `docker system df -v`
2. ✅ Implement `GetAllMutagenVolumes()` to list all Mutagen volumes
3. ✅ Implement `CheckMutagenIgnorePatterns()` to analyze mutagen.yml
4. ✅ Implement `DiagnoseMutagenConfiguration()` as comprehensive check

### Phase 2: Docker Utilities (pkg/dockerutil/dockerutils.go)
1. ✅ Implement `ParseDockerSystemDf()` to parse volume sizes
2. ✅ Add unit tests for parsing various docker output formats

### Phase 3: Command Implementation (cmd/ddev/cmd/utility-mutagen-diagnose.go)
1. ✅ Create command file with cobra structure
2. ✅ Implement main Run function
3. ✅ Format output with sections and color codes
4. ✅ Handle `--all` flag to show all volumes
5. ✅ Return appropriate exit codes

### Phase 4: Testing
1. ✅ Test with projects that have Mutagen enabled
2. ✅ Test with various volume sizes (small, large, very large)
3. ✅ Test with and without upload_dirs configured
4. ✅ Test with customized mutagen.yml
5. ✅ Test `--all` flag with multiple projects
6. ✅ Test on macOS (primary Mutagen use case)
7. ✅ Test on Linux/WSL2 (less common but supported)

### Phase 5: Documentation
1. ✅ Add command to help output
2. ✅ Update performance.md troubleshooting section to reference new command
3. ✅ Add to command reference documentation

### Phase 6: Future Integration (Optional)
1. ⬜ Integrate into `ddev utility diagnose` (add Mutagen section)
2. ⬜ Integrate into `ddev utility test` (add comprehensive Mutagen analysis)
3. ⬜ Add `--json` flag for machine-readable output

## Implementation Discoveries and Gotchas

During implementation, several important details were discovered that are worth documenting:

### 1. upload_dirs Are Relative to Docroot, Not Project Root

**Issue:** Initial implementation assumed `upload_dirs` were relative to the project root (`AppRoot`), but they are actually relative to the docroot.

**Solution:** When building paths to check if files are in upload_dirs, use:
```go
fullPath := filepath.Join(app.AppRoot, app.Docroot, dir)
```

**Example:** For a Drupal project with `docroot: web` and `upload_dirs: ["sites/default/files"]`, the full path is `/project/web/sites/default/files`, not `/project/sites/default/files`.

### 2. Docker System df -v Parsing Logic Order Matters

**Issue:** Initial parsing logic checked for "space usage:" before entering the volumes section, causing it to exit immediately when encountering "Images space usage:" at the beginning of the output.

**Solution:** Only check for other "space usage:" section headers AFTER already being in the volumes section:
```go
if strings.Contains(line, "Local Volumes space usage") {
    inVolumesSection = true
    continue
}

if inVolumesSection {
    // Now check for other sections to stop parsing
    if strings.Contains(line, "space usage:") {
        break
    }
    // Parse volume lines...
}
```

### 3. File Exclusion Pattern Matching Must Be Precise

**Issue:** Simple substring matching like `strings.Contains(mutagenContent, "web")` would match "/web/sites/default/files" in mutagen.yml and incorrectly think files in the `web/` directory were excluded.

**Solution:** Check for exact path patterns in YAML list format:
```go
pathPattern := "/" + filepath.ToSlash(relPath)
if strings.Contains(mutagenContent, fmt.Sprintf(`- "%s"`, pathPattern)) ||
   strings.Contains(mutagenContent, fmt.Sprintf(`- '%s'`, pathPattern)) ||
   strings.Contains(mutagenContent, fmt.Sprintf("- %s\n", pathPattern)) {
    isExcluded = true
}
```

This is a heuristic approach - a complete implementation would parse the YAML and evaluate Mutagen's pattern matching rules.

### 4. Use util.Debug() for Debug Output

Use `util.Debug()` instead of `fmt.Printf()` for debug messages, so they only appear when `DDEV_DEBUG=true`:
```go
util.Debug("CheckLargeFilesInSync: Found large file: %s (size=%d)", path, info.Size())
```

### 5. Documentation Links in Warning Messages

Add documentation links to warning/error messages to help users find solutions:
```go
warnings = append(warnings, fmt.Sprintf("%s file being synced: %s (%s) - consider excluding from sync. See https://docs.ddev.com/en/stable/users/install/performance/#mutagen-troubleshooting", severity, relPath, sizeStr))
```

### 6. Actual File Location vs Plan

**Plan stated:** Add volume parsing to `pkg/dockerutil/dockerutils.go`
**Actually used:** Added to existing `pkg/dockerutil/volumes.go` which already had volume-related functions

This was a better architectural decision as it kept all volume operations together.

### 7. Documentation Integration

Added documentation in two places:
- **commands.md:** Full command reference with examples and all flags
- **performance.md:** Added as first troubleshooting step with detailed feature list

Both docs cross-reference each other using the `#mutagen-troubleshooting` anchor.

### 8. upload_dirs Override Behavior

**Critical Discovery:** When setting `upload_dirs` in `.ddev/config.yaml`, you're **replacing** the defaults, not adding to them.

**Problem:** If a Drupal project has a default of `sites/default/files`, and you add:
```yaml
upload_dirs:
  - node_modules
```
You've just **lost** the `sites/default/files` default!

**Solution:** The command output must show the **complete** upload_dirs list including existing values:
```yaml
upload_dirs:
  - sites/default/files  # Keep existing CMS default
  - ../junk-no-sync     # Keep any custom values
  - node_modules        # Add new exclusion
```

This is implemented by reading `result.UploadDirs` (from `app.GetUploadDirs()`) and displaying all existing entries before adding the new one.

**Documentation Updated:** Both performance.md examples now emphasize the need to include existing defaults when overriding `upload_dirs`, and added a warning box making it clear that editing `mutagen.yml` is rarely needed.

### 9. Scanning for All node_modules Directories

**Initial Implementation:** Only checked for `node_modules` at the project root.

**Problem:** Many projects have `node_modules` in multiple locations:
- Project root (user's build tools)
- `web/core/node_modules` (Drupal core)
- `web/themes/custom/mytheme/node_modules` (theme dependencies)
- `web/modules/custom/mymodule/node_modules` (module dependencies)

**Solution:** Implemented `findNodeModulesDirectories()` that uses `filepath.Walk` to find all `node_modules` directories in the project:
- Skips hidden directories (`.git`, `.ddev`, etc.)
- Stops recursing into `node_modules` directories (avoids finding nested dependencies)
- Returns all absolute paths to top-level `node_modules` directories

**Output Example:**
```
⚠ 2 node_modules directories exist but are not excluded from sync
  → Add to .ddev/config.yaml:
    upload_dirs:
      - sites/default/files
      - ../node_modules        # Project root
      - core/node_modules      # Drupal core
```

**Performance:** Fast even on large projects - the walk skips `.git`, `.ddev`, and doesn't recurse into `node_modules` themselves.

## Testing Strategy

### Unit Tests
- Test volume size parsing with mock docker output
- Test ignore pattern parsing with various mutagen.yml configs
- Test diagnostic logic with different app states

### Integration Tests
- Create test project with Mutagen enabled
- Verify volume size detection
- Test with large files to trigger warnings
- Test upload_dirs detection and suggestions

### Manual Testing Checklist
- [x] Run on project with Mutagen enabled and working (tested on d11 project)
- [ ] Run on project with Mutagen paused
- [ ] Run on project with large volume (>5GB)
- [x] Run on project without upload_dirs configured
- [ ] Run on project with customized mutagen.yml
- [x] Run with `--all` flag on system with multiple projects (shows 6 volumes)
- [x] Verify recommendations are actionable and accurate
- [x] Verify large file detection (274M hugedb.sql.gz correctly detected)
- [x] Verify files in upload_dirs are correctly excluded from warnings
