package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// CloneInfo holds metadata about a single DDEV-managed clone.
// Derived dynamically from git worktree list output, not persisted to a file.
type CloneInfo struct {
	CloneName         string `json:"clone_name"`
	SourceProjectName string `json:"source_project_name"`
	ProjectName       string `json:"project_name"`
	WorktreePath      string `json:"path"`
	Branch            string `json:"branch"`
	Status            string `json:"status"`
	Current           bool   `json:"current"`
}

// cloneInfix is the naming convention separator used to identify clone projects.
const cloneInfix = "-clone-"

// getCloneProjectName returns the DDEV project name for a clone.
// Format: <sourceProjectName>-clone-<cloneName>
func getCloneProjectName(sourceProjectName, cloneName string) string {
	return sourceProjectName + cloneInfix + cloneName
}

// getCloneWorktreePath returns the expected worktree path for a clone.
// Clones are placed as sibling directories of the source project.
func getCloneWorktreePath(app *DdevApp, cloneName string) string {
	return filepath.Join(filepath.Dir(app.AppRoot), app.Name+cloneInfix+cloneName)
}

// getSourceProjectName determines the source project name from the current project.
// If the current project is itself a clone, it returns the original source project name.
func getSourceProjectName(app *DdevApp) string {
	name := app.Name
	if idx := strings.Index(name, cloneInfix); idx != -1 {
		return name[:idx]
	}
	return name
}

// getCloneNameFromProjectName extracts the clone name from a clone project name.
// Returns empty string if the project name does not match the clone pattern.
func getCloneNameFromProjectName(sourceProjectName, projectName string) string {
	prefix := sourceProjectName + cloneInfix
	if strings.HasPrefix(projectName, prefix) {
		return projectName[len(prefix):]
	}
	return ""
}

// getGitRoot returns the git root directory for the given path.
func getGitRoot(appRoot string) (string, error) {
	out, err := exec.RunHostCommand("git", "-C", appRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git is not installed): %v, output: %s", err, out)
	}
	return strings.TrimSpace(out), nil
}

// isCloneWorktree checks if a worktree path matches the clone naming convention
// for the given source project. Returns the clone name if it matches.
func isCloneWorktree(sourceProjectName, worktreePath string) (bool, string) {
	base := filepath.Base(worktreePath)
	prefix := sourceProjectName + cloneInfix
	if strings.HasPrefix(base, prefix) {
		cloneName := base[len(prefix):]
		if cloneName != "" {
			return true, cloneName
		}
	}
	return false, ""
}

// discoverClones returns all DDEV-managed clones by parsing git worktree list.
// It can be called from either the source project or any clone.
func discoverClones(app *DdevApp) ([]CloneInfo, error) {
	sourceProjectName := getSourceProjectName(app)

	// Get the git root (works from either source or clone worktree)
	gitRoot, err := getGitRoot(app.AppRoot)
	if err != nil {
		return nil, err
	}

	// Parse git worktree list --porcelain
	out, err := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list git worktrees: %v, output: %s", err, out)
	}

	var clones []CloneInfo
	var currentPath, currentBranch string

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			// Save previous entry if it was a clone
			if currentPath != "" {
				if isClone, cloneName := isCloneWorktree(sourceProjectName, currentPath); isClone {
					clones = append(clones, CloneInfo{
						CloneName:         cloneName,
						SourceProjectName: sourceProjectName,
						ProjectName:       getCloneProjectName(sourceProjectName, cloneName),
						WorktreePath:      currentPath,
						Branch:            currentBranch,
					})
				}
			}
			currentPath = strings.TrimPrefix(line, "worktree ")
			currentBranch = ""
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			// refs/heads/branch-name -> branch-name
			currentBranch = strings.TrimPrefix(ref, "refs/heads/")
		} else if line == "" {
			// End of a worktree block
			if currentPath != "" {
				if isClone, cloneName := isCloneWorktree(sourceProjectName, currentPath); isClone {
					clones = append(clones, CloneInfo{
						CloneName:         cloneName,
						SourceProjectName: sourceProjectName,
						ProjectName:       getCloneProjectName(sourceProjectName, cloneName),
						WorktreePath:      currentPath,
						Branch:            currentBranch,
					})
				}
			}
			currentPath = ""
			currentBranch = ""
		}
	}

	// Handle the last entry if the output doesn't end with a blank line
	if currentPath != "" {
		if isClone, cloneName := isCloneWorktree(sourceProjectName, currentPath); isClone {
			clones = append(clones, CloneInfo{
				CloneName:         cloneName,
				SourceProjectName: sourceProjectName,
				ProjectName:       getCloneProjectName(sourceProjectName, cloneName),
				WorktreePath:      currentPath,
				Branch:            currentBranch,
			})
		}
	}

	return clones, nil
}

// getProjectVolumes returns the list of source->target volume name pairs
// that need to be cloned for a project.
func getProjectVolumes(app *DdevApp, cloneProjectName string) []volumePair {
	var pairs []volumePair

	sourceProjectName := app.Name

	// Database volume (based on configured type)
	switch app.Database.Type {
	case "postgres":
		pairs = append(pairs, volumePair{
			source: app.GetPostgresVolumeName(),
			target: cloneProjectName + "-postgres",
		})
	default:
		// MariaDB or MySQL
		pairs = append(pairs, volumePair{
			source: app.GetMariaDBVolumeName(),
			target: cloneProjectName + "-mariadb",
		})
	}

	// Snapshot volume (optional)
	snapshotSource := "ddev-" + sourceProjectName + "-snapshots"
	snapshotTarget := "ddev-" + cloneProjectName + "-snapshots"
	pairs = append(pairs, volumePair{source: snapshotSource, target: snapshotTarget})

	// Config volume (optional)
	configSource := sourceProjectName + "-ddev-config"
	configTarget := cloneProjectName + "-ddev-config"
	pairs = append(pairs, volumePair{source: configSource, target: configTarget})

	// Mutagen volume (optional, only if Mutagen is enabled)
	if app.IsMutagenEnabled() {
		mutagenSource := GetMutagenVolumeName(app)
		mutagenTarget := cloneProjectName + "_project_mutagen"
		pairs = append(pairs, volumePair{source: mutagenSource, target: mutagenTarget})
	}

	return pairs
}

// volumePair holds a source and target volume name for cloning.
type volumePair struct {
	source string
	target string
}

// CloneCreate creates a clone of the given DDEV project.
func CloneCreate(app *DdevApp, cloneName string, branch string, noStart bool) error {
	sourceProjectName := getSourceProjectName(app)
	cloneProjectName := getCloneProjectName(sourceProjectName, cloneName)

	// Check if we're running from a clone (redirect to source)
	if sourceProjectName != app.Name {
		// We're in a clone, resolve the source
		sourceProject := globalconfig.GetProject(sourceProjectName)
		if sourceProject == nil {
			return fmt.Errorf("source project '%s' not found in DDEV project list", sourceProjectName)
		}
		sourceApp := &DdevApp{}
		if err := sourceApp.Init(sourceProject.AppRoot); err != nil {
			return fmt.Errorf("failed to initialize source project '%s': %v", sourceProjectName, err)
		}
		app = sourceApp
	}

	// Validate git repo
	gitRoot, err := getGitRoot(app.AppRoot)
	if err != nil {
		return fmt.Errorf("project '%s' is not in a git repository: %v", app.Name, err)
	}

	// Check for conflicts
	if existingProject := globalconfig.GetProject(cloneProjectName); existingProject != nil {
		return fmt.Errorf("a project named '%s' already exists. Use 'ddev clone remove %s' to remove it first", cloneProjectName, cloneName)
	}

	worktreePath := getCloneWorktreePath(app, cloneName)

	// Run pre-clone-create hook
	if err := app.ProcessHooks("pre-clone-create"); err != nil {
		return fmt.Errorf("failed to process pre-clone-create hooks: %v", err)
	}

	util.Success("Creating clone '%s' of project '%s'...", cloneName, app.Name)

	// Create git worktree
	util.Success("  Creating git worktree at %s...", worktreePath)
	var gitArgs []string
	if branch != "" {
		// Use existing branch
		gitArgs = []string{"-C", gitRoot, "worktree", "add", worktreePath, branch}
	} else {
		// Create new branch with the clone name
		gitArgs = []string{"-C", gitRoot, "worktree", "add", "-b", cloneName, worktreePath}
	}
	out, err := exec.RunHostCommand("git", gitArgs...)
	if err != nil {
		return fmt.Errorf("failed to create git worktree: %v, output: %s", err, out)
	}

	// Track cleanup needed on error
	worktreeCreated := true
	var createdVolumes []string
	defer func() {
		if err != nil {
			// Rollback: clean up on failure
			util.Warning("Clone creation failed, cleaning up...")
			for _, vol := range createdVolumes {
				_ = removeVolumeIfExists(vol)
			}
			if worktreeCreated {
				_, _ = exec.RunHostCommand("git", "-C", gitRoot, "worktree", "remove", "--force", worktreePath)
				_, _ = exec.RunHostCommand("git", "-C", gitRoot, "worktree", "prune")
			}
			_ = globalconfig.RemoveProjectInfo(cloneProjectName)
		}
	}()

	// Write .ddev/config.yaml in the clone with the clone project name
	cloneDdevDir := filepath.Join(worktreePath, ".ddev")
	cloneConfigPath := filepath.Join(cloneDdevDir, "config.yaml")
	err = writeCloneConfig(app, cloneConfigPath, cloneProjectName)
	if err != nil {
		return fmt.Errorf("failed to write clone config: %v", err)
	}

	// Determine volumes to clone
	volumePairs := getProjectVolumes(app, cloneProjectName)

	// Check source site status and stop DB if running
	status, _ := app.SiteStatus()
	dbWasRunning := status == SiteRunning

	if dbWasRunning {
		util.Success("  Stopping database container for consistent copy...")
		if _, _, composeErr := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
			ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
			Action:       []string{"stop", "db"},
		}); composeErr != nil {
			return fmt.Errorf("failed to stop source database container: %v", composeErr)
		}
		defer func() {
			util.Success("  Resuming database container...")
			if _, _, composeErr := dockerutil.ComposeCmd(&dockerutil.ComposeCmdOpts{
				ComposeFiles: []string{app.DockerComposeFullRenderedYAMLPath()},
				Action:       []string{"start", "db"},
			}); composeErr != nil {
				util.Warning("Failed to restart source database container: %v", composeErr)
			}
		}()
	}

	// Clone each volume
	cloner := GetVolumeCloner()
	for _, vp := range volumePairs {
		if !dockerutil.VolumeExists(vp.source) {
			util.Debug("Skipping volume %s (does not exist)", vp.source)
			continue
		}
		util.Success("  Cloning volume %s -> %s...", vp.source, vp.target)
		if cloneErr := cloner.CloneVolume(vp.source, vp.target); cloneErr != nil {
			err = fmt.Errorf("failed to clone volume %s: %v", vp.source, cloneErr)
			return err
		}
		createdVolumes = append(createdVolumes, vp.target)
	}

	// Init the clone app
	cloneApp := &DdevApp{}
	if initErr := cloneApp.Init(worktreePath); initErr != nil {
		err = fmt.Errorf("failed to initialize clone project: %v", initErr)
		return err
	}

	// Resolve host port conflicts before starting.
	// The clone inherits port settings from the source config, and in
	// devcontainer mode DockerEnv() assigns default ports (3306, 80, etc.)
	// that would collide with the already-running source project.
	if resolveErr := resolveClonePortConflicts(cloneApp); resolveErr != nil {
		err = fmt.Errorf("failed to resolve port conflicts for clone: %v", resolveErr)
		return err
	}

	// Register the clone project
	if regErr := globalconfig.SetProjectAppRoot(cloneProjectName, worktreePath); regErr != nil {
		err = fmt.Errorf("failed to register clone project: %v", regErr)
		return err
	}

	// Start the clone (unless --no-start)
	if !noStart {
		util.Success("  Starting clone project %s...", cloneProjectName)
		if startErr := cloneApp.Start(); startErr != nil {
			err = fmt.Errorf("failed to start clone project: %v", startErr)
			return err
		}
	}

	// Run post-clone-create hook (in clone context)
	if hookErr := cloneApp.ProcessHooks("post-clone-create"); hookErr != nil {
		util.Warning("post-clone-create hook failed: %v", hookErr)
	}

	util.Success("Successfully created clone '%s'", cloneName)
	if !noStart {
		util.Success("Clone URL: https://%s.ddev.site", cloneProjectName)
	}

	// Clear the error so deferred cleanup doesn't trigger
	err = nil
	return nil
}

// CloneList returns all clones for the given project with enriched status information.
func CloneList(app *DdevApp) ([]CloneInfo, error) {
	clones, err := discoverClones(app)
	if err != nil {
		return nil, err
	}

	// Enrich each clone with DDEV status
	for i := range clones {
		project := globalconfig.GetProject(clones[i].ProjectName)
		if project == nil {
			clones[i].Status = "unregistered"
			continue
		}

		cloneApp := &DdevApp{}
		if initErr := cloneApp.Init(project.AppRoot); initErr != nil {
			clones[i].Status = "unknown"
			continue
		}

		status, _ := cloneApp.SiteStatus()
		if status == "" {
			clones[i].Status = SiteStopped
		} else {
			clones[i].Status = status
		}

		// Mark the current clone if we're running from its directory
		if clones[i].WorktreePath == app.AppRoot {
			clones[i].Current = true
		}
	}

	return clones, nil
}

// CloneRemove removes a clone and all its resources.
func CloneRemove(sourceApp *DdevApp, cloneName string, force bool) error {
	sourceProjectName := getSourceProjectName(sourceApp)
	cloneProjectName := getCloneProjectName(sourceProjectName, cloneName)

	// Resolve the source app if needed
	if sourceProjectName != sourceApp.Name {
		sourceProject := globalconfig.GetProject(sourceProjectName)
		if sourceProject == nil {
			return fmt.Errorf("source project '%s' not found in DDEV project list", sourceProjectName)
		}
		resolvedApp := &DdevApp{}
		if err := resolvedApp.Init(sourceProject.AppRoot); err != nil {
			return fmt.Errorf("failed to initialize source project '%s': %v", sourceProjectName, err)
		}
		sourceApp = resolvedApp
	}

	gitRoot, err := getGitRoot(sourceApp.AppRoot)
	if err != nil {
		return fmt.Errorf("project '%s' is not in a git repository: %v", sourceApp.Name, err)
	}

	// Find the clone
	clones, err := discoverClones(sourceApp)
	if err != nil {
		return nil
	}

	var cloneInfo *CloneInfo
	for i := range clones {
		if clones[i].CloneName == cloneName {
			cloneInfo = &clones[i]
			break
		}
	}

	// If clone not found in worktrees, try to clean up by project name alone
	if cloneInfo == nil {
		project := globalconfig.GetProject(cloneProjectName)
		if project == nil {
			return fmt.Errorf("clone '%s' not found", cloneName)
		}
		// Graceful partial cleanup
		return cleanupOrphanedClone(cloneProjectName, project.AppRoot, gitRoot)
	}

	// Run pre-clone-remove hook
	if err := sourceApp.ProcessHooks("pre-clone-remove"); err != nil {
		return fmt.Errorf("failed to process pre-clone-remove hooks: %v", err)
	}

	// Check for dirty worktree
	if !force {
		dirtyOutput, dirtyErr := exec.RunHostCommand("git", "-C", cloneInfo.WorktreePath, "status", "--porcelain")
		if dirtyErr == nil && strings.TrimSpace(dirtyOutput) != "" {
			fmt.Printf("Clone '%s' has uncommitted changes:\n%s\n", cloneName, strings.TrimSpace(dirtyOutput))
			if !util.Confirm("Remove anyway?") {
				return fmt.Errorf("clone removal cancelled")
			}
		}
	}

	util.Success("Removing clone '%s' of project '%s'...", cloneName, sourceProjectName)

	// Init clone app and stop with full cleanup
	util.Success("  Stopping and removing Docker resources...")
	cloneApp := &DdevApp{}
	if initErr := cloneApp.Init(cloneInfo.WorktreePath); initErr != nil {
		// Graceful partial cleanup if init fails
		util.Warning("Unable to initialize clone app, performing manual cleanup: %v", initErr)
		return cleanupOrphanedClone(cloneProjectName, cloneInfo.WorktreePath, gitRoot)
	}

	// Stop with removeData=true to clean up all Docker resources
	if stopErr := cloneApp.Stop(true, false); stopErr != nil {
		util.Warning("Failed to fully stop clone: %v, attempting manual cleanup", stopErr)
	}

	// Remove git worktree
	util.Success("  Removing git worktree...")
	if out, removeErr := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "remove", "--force", cloneInfo.WorktreePath); removeErr != nil {
		util.Warning("Failed to remove git worktree: %v, output: %s", removeErr, out)
	}
	if out, pruneErr := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "prune"); pruneErr != nil {
		util.Warning("Failed to prune git worktrees: %v, output: %s", pruneErr, out)
	}

	// Run post-clone-remove hook
	if hookErr := sourceApp.ProcessHooks("post-clone-remove"); hookErr != nil {
		util.Warning("post-clone-remove hook failed: %v", hookErr)
	}

	util.Success("Successfully removed clone '%s'", cloneName)
	return nil
}

// ClonePrune detects and cleans up stale clones whose worktree directories no longer exist.
func ClonePrune(app *DdevApp, dryRun bool) ([]string, error) {
	sourceProjectName := getSourceProjectName(app)
	prefix := sourceProjectName + cloneInfix

	gitRoot, err := getGitRoot(app.AppRoot)
	if err != nil {
		return nil, fmt.Errorf("project '%s' is not in a git repository: %v", app.Name, err)
	}

	var pruned []string

	// Check all DDEV projects matching the clone pattern
	for name, project := range globalconfig.DdevProjectList {
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// Check if the worktree path still exists
		if fileExists(project.AppRoot) {
			continue
		}

		cloneName := getCloneNameFromProjectName(sourceProjectName, name)

		if dryRun {
			util.Success("Would remove stale clone '%s' (worktree %s no longer exists)", cloneName, project.AppRoot)
			pruned = append(pruned, cloneName)
			continue
		}

		util.Success("Pruning stale clone '%s'...", cloneName)

		// Try to clean up Docker resources
		util.Success("  Removing Docker resources for %s...", name)
		cloneApp := &DdevApp{}
		if initErr := cloneApp.Init(project.AppRoot); initErr == nil {
			if stopErr := cloneApp.Stop(true, false); stopErr != nil {
				util.Warning("Failed to stop stale clone: %v", stopErr)
			}
		} else {
			// Manual cleanup: remove project from global list
			if removeErr := globalconfig.RemoveProjectInfo(name); removeErr != nil {
				util.Warning("Failed to remove project info for %s: %v", name, removeErr)
			}
		}

		pruned = append(pruned, cloneName)
	}

	// Run git worktree prune to clean git state
	util.Success("  Pruning git worktree references...")
	if out, pruneErr := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "prune"); pruneErr != nil {
		util.Warning("Failed to prune git worktrees: %v, output: %s", pruneErr, out)
	}

	return pruned, nil
}

// cleanupOrphanedClone handles cleanup when a clone's app cannot be properly initialized.
func cleanupOrphanedClone(cloneProjectName, worktreePath, gitRoot string) error {
	// Remove project from global list
	if removeErr := globalconfig.RemoveProjectInfo(cloneProjectName); removeErr != nil {
		util.Warning("Failed to remove project info: %v", removeErr)
	}

	// Try to remove git worktree
	if out, removeErr := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "remove", "--force", worktreePath); removeErr != nil {
		util.Warning("Failed to remove git worktree: %v, output: %s", removeErr, out)
	}
	if out, pruneErr := exec.RunHostCommand("git", "-C", gitRoot, "worktree", "prune"); pruneErr != nil {
		util.Warning("Failed to prune git worktrees: %v, output: %s", pruneErr, out)
	}

	util.Success("Cleaned up orphaned clone '%s'", cloneProjectName)
	return nil
}

// writeCloneConfig reads the source project's config.yaml and writes a modified
// version to the clone with the clone's project name.
func writeCloneConfig(sourceApp *DdevApp, cloneConfigPath, cloneProjectName string) error {
	// Read existing config content
	sourceConfigPath := filepath.Join(sourceApp.AppRoot, ".ddev", "config.yaml")
	configBytes, err := os.ReadFile(sourceConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read source config: %v", err)
	}

	// Parse and modify the name field
	configStr := string(configBytes)
	// Replace the name line in the YAML
	lines := strings.Split(configStr, "\n")
	nameFound := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "name:") {
			lines[i] = "name: " + cloneProjectName
			nameFound = true
			break
		}
	}
	if !nameFound {
		// Prepend name if not found
		lines = append([]string{"name: " + cloneProjectName}, lines...)
	}

	return os.WriteFile(cloneConfigPath, []byte(strings.Join(lines, "\n")), 0644)
}

// resolveClonePortConflicts clears host port settings that were inherited
// from the source project's config to prevent bind conflicts. In devcontainer
// environments, it allocates unique ephemeral ports because DockerEnv() would
// otherwise assign the same well-known defaults to every project.
func resolveClonePortConflicts(app *DdevApp) error {
	// Clear any explicitly configured host ports inherited from the source.
	// For non-devcontainer environments this is sufficient: empty ports mean
	// the services are only accessible through the Docker network / ddev-router.
	app.HostDBPort = ""
	app.HostWebserverPort = ""
	app.HostHTTPSPort = ""
	app.HostMailpitPort = ""

	// In devcontainer/Codespaces mode DockerEnv() assigns well-known default
	// host ports (80, 3306, 8443, 8027 …) when the fields are empty.
	// Pre-assign unique ephemeral ports so the clone does not collide with
	// the source project that already holds those defaults.
	if nodeps.IsDevcontainer() {
		localIP := "127.0.0.1"

		port, err := globalconfig.GetFreePort(localIP)
		if err != nil {
			return fmt.Errorf("allocating host DB port: %v", err)
		}
		app.HostDBPort = port

		port, err = globalconfig.GetFreePort(localIP)
		if err != nil {
			return fmt.Errorf("allocating host webserver port: %v", err)
		}
		app.HostWebserverPort = port

		port, err = globalconfig.GetFreePort(localIP)
		if err != nil {
			return fmt.Errorf("allocating host HTTPS port: %v", err)
		}
		app.HostHTTPSPort = port

		port, err = globalconfig.GetFreePort(localIP)
		if err != nil {
			return fmt.Errorf("allocating host Mailpit port: %v", err)
		}
		app.HostMailpitPort = port
	}

	// Persist the updated port configuration so subsequent restarts
	// keep using the same ports.
	if err := app.WriteConfig(); err != nil {
		return fmt.Errorf("writing updated clone config: %v", err)
	}

	return nil
}

// removeVolumeIfExists removes a Docker volume if it exists.
func removeVolumeIfExists(volumeName string) error {
	if dockerutil.VolumeExists(volumeName) {
		return dockerutil.RemoveVolume(volumeName)
	}
	return nil
}

// fileExists checks if a path exists on disk.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetSourceProjectNameExported is an exported wrapper for getSourceProjectName.
func GetSourceProjectNameExported(app *DdevApp) string {
	return getSourceProjectName(app)
}

// GetCloneNamesFunc returns a ValidArgsFunction that provides tab completion
// for clone names based on discoverClones().
func GetCloneNamesFunc(numArgs int) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if numArgs > 0 && len(args)+1 > numArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		app, err := GetActiveApp("")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		clones, err := discoverClones(app)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var names []string
		for _, c := range clones {
			names = append(names, c.CloneName)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
