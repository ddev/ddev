package ddevapp

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/exec"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

type Snapshot struct {
	Name    string
	Created time.Time
}

// SnapshotRestoreTarget represents a snapshot that can be restored from a project path.
type SnapshotRestoreTarget struct {
	Name       string
	Created    time.Time
	SourceRoot string
	Label      string
}

// SnapshotRestoreDefaultWaitTime is the max time we'll wait for snapshot restore.
// If default_container_timeout is set higher than that it can be more
const SnapshotRestoreDefaultWaitTime = 600

// DeleteSnapshot removes the snapshot tarball or directory inside a project
func (app *DdevApp) DeleteSnapshot(snapshotName string) error {
	var err error
	err = app.ProcessHooks("pre-delete-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process pre-delete-snapshot hooks: %v", err)
	}

	snapshotFullName, err := GetSnapshotFileFromName(snapshotName, app)
	if err != nil {
		return err
	}

	snapshotFullPath := path.Join("db_snapshots", snapshotFullName)
	hostSnapshot := app.GetConfigPath(snapshotFullPath)

	if !fileutil.FileExists(hostSnapshot) {
		return fmt.Errorf("no snapshot '%s' currently exists in project '%s'", snapshotName, app.Name)
	}
	if err = os.RemoveAll(hostSnapshot); err != nil {
		return fmt.Errorf("failed to remove snapshot '%s': %v", hostSnapshot, err)
	}

	util.Success("Deleted database snapshot '%s'", snapshotName)
	err = app.ProcessHooks("post-delete-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process post-delete-snapshot hooks: %v", err)
	}

	return nil
}

// GetLatestSnapshot returns the name of the latest created snapshot of a project
func (app *DdevApp) GetLatestSnapshot() (string, error) {
	var snapshots []string

	snapshots, err := app.ListSnapshotNames()
	if err != nil {
		return "", err
	}

	if len(snapshots) == 0 {
		return "", fmt.Errorf("no snapshots found")
	}

	return snapshots[0], nil
}

// ListSnapshotRestoreTargets returns restore targets for the current project and any matching
// project paths in git worktrees.
func (app *DdevApp) ListSnapshotRestoreTargets() ([]SnapshotRestoreTarget, error) {
	targets, err := app.getSnapshotRestoreTargetsForRoot(app.AppRoot)
	if err != nil {
		return nil, err
	}

	worktreeRoots, err := app.getGitWorktreePaths()
	if err != nil {
		return nil, err
	}
	for _, worktreeRoot := range worktreeRoots {
		if samePath(worktreeRoot, app.AppRoot) {
			continue
		}
		projectRoot, err := app.getProjectRootForWorktree(worktreeRoot)
		if err != nil || projectRoot == "" {
			continue
		}
		if samePath(projectRoot, app.AppRoot) {
			continue
		}
		otherTargets, err := app.getSnapshotRestoreTargetsForRoot(projectRoot)
		if err != nil {
			continue
		}
		targets = append(targets, otherTargets...)
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Created.After(targets[j].Created)
	})

	return targets, nil
}

// GetLatestSnapshotRestoreTarget returns the latest snapshot restore target available for the current project's worktree context.
func (app *DdevApp) GetLatestSnapshotRestoreTarget() (*SnapshotRestoreTarget, error) {
	targets, err := app.ListSnapshotRestoreTargets()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}
	return &targets[0], nil
}

func (app *DdevApp) getSnapshotRestoreTargetsForRoot(root string) ([]SnapshotRestoreTarget, error) {
	var targets []SnapshotRestoreTarget
	if root == "" {
		return targets, nil
	}

	snapshots, err := listSnapshotsInDir(filepath.Join(root, ".ddev", "db_snapshots"))
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		label := snapshot.Name
		if !samePath(root, app.AppRoot) {
			label = fmt.Sprintf("%s (%s)", snapshot.Name, filepath.Base(root))
		}
		targets = append(targets, SnapshotRestoreTarget{
			Name:       snapshot.Name,
			Created:    snapshot.Created,
			SourceRoot: root,
			Label:      label,
		})
	}

	return targets, nil
}

func (app *DdevApp) getGitWorktreePaths() ([]string, error) {
	repoRootOutput, err := exec.RunHostCommand("git", "-C", app.AppRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, nil
	}
	repoRoot := strings.TrimSpace(repoRootOutput)
	if repoRoot == "" {
		return nil, nil
	}

	worktreeOutput, err := exec.RunHostCommand("git", "-C", repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktreePaths []string
	for _, line := range strings.Split(strings.TrimSpace(worktreeOutput), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			worktreePath := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			absPath, err := filepath.Abs(worktreePath)
			if err != nil {
				continue
			}
			worktreePaths = append(worktreePaths, absPath)
		}
	}

	return worktreePaths, nil
}

func (app *DdevApp) getProjectRootForWorktree(worktreeRoot string) (string, error) {
	repoRootOutput, err := exec.RunHostCommand("git", "-C", app.AppRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", nil
	}
	repoRoot := strings.TrimSpace(repoRootOutput)
	if repoRoot == "" {
		return "", nil
	}

	relPath, err := filepath.Rel(repoRoot, app.AppRoot)
	if err != nil {
		return "", err
	}
	if relPath == "." {
		return worktreeRoot, nil
	}
	return filepath.Join(worktreeRoot, relPath), nil
}

func samePath(path1, path2 string) bool {
	absPath1, err := filepath.Abs(path1)
	if err != nil {
		return false
	}
	absPath2, err := filepath.Abs(path2)
	if err != nil {
		return false
	}
	return filepath.Clean(absPath1) == filepath.Clean(absPath2)
}

// ListSnapshots returns a list of the names of all project snapshots
func (app *DdevApp) ListSnapshotNames() ([]string, error) {
	var names []string
	snapshots, err := app.ListSnapshots()

	for _, snapshot := range snapshots {
		names = append(names, snapshot.Name)
	}

	return names, err
}

func listSnapshotsInDir(snapshotDir string) ([]Snapshot, error) {
	var snapshots []Snapshot

	if !fileutil.FileExists(snapshotDir) {
		return snapshots, nil
	}

	fileNames, err := fileutil.ListFilesInDir(snapshotDir)
	if err != nil {
		return snapshots, err
	}

	var files []fs.FileInfo
	for _, n := range fileNames {
		f, err := os.Stat(filepath.Join(snapshotDir, n))
		if err != nil {
			return snapshots, err
		}
		files = append(files, f)
	}

	// Sort snapshots by last modification time
	// we need that to detect the latest snapshot
	// first snapshot is the latest
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})

	// Match snapshot files created with gzip (.gz) or zstd (.zst)
	m := regexp.MustCompile(`-(mariadb|mysql|postgres)_[0-9.]*\.(gz|zst)$`)

	for _, f := range files {
		if f.IsDir() || strings.HasSuffix(f.Name(), ".gz") || strings.HasSuffix(f.Name(), ".zst") {
			n := m.ReplaceAll([]byte(f.Name()), []byte(""))
			snapshot := Snapshot{
				Name:    string(n),
				Created: f.ModTime(),
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

// ListSnapshots returns a list of all project snapshots
func (app *DdevApp) ListSnapshots() ([]Snapshot, error) {
	snapshotDir := app.GetConfigPath("db_snapshots")
	return listSnapshotsInDir(snapshotDir)
}

// RestoreSnapshot restores a MariaDB snapshot of the db to be loaded
// The project must be stopped and Docker volume removed and recreated for this to work.
func (app *DdevApp) RestoreSnapshot(snapshotName string) error {
	return app.restoreSnapshot(snapshotName, app.AppRoot)
}

// RestoreSnapshotFromWorktree restores a snapshot from the specified project root, which may live in another git worktree.
func (app *DdevApp) RestoreSnapshotFromWorktree(snapshotName, sourceRoot string) error {
	return app.restoreSnapshot(snapshotName, sourceRoot)
}

func (app *DdevApp) restoreSnapshot(snapshotName, sourceRoot string) error {
	var err error
	err = app.ProcessHooks("pre-restore-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process pre-restore-snapshot hooks: %v", err)
	}

	currentDBVersion := app.Database.Type + "_" + app.Database.Version

	snapshotFile, err := getSnapshotFileFromNameFromRoot(snapshotName, sourceRoot)
	if err != nil {
		return fmt.Errorf("no snapshot found for name %s: %v", snapshotName, err)
	}
	snapshotDir := filepath.Join(sourceRoot, ".ddev", "db_snapshots")
	hostSnapshotFileOrDir := filepath.Join(snapshotDir, snapshotFile)

	if !fileutil.FileExists(hostSnapshotFileOrDir) {
		return fmt.Errorf("failed to find a snapshot at %s", hostSnapshotFileOrDir)
	}

	snapshotDBVersion := ""

	// If the snapshot is a directory, (old obsolete style) then
	// look for db_mariadb_version.txt in the directory to get the version.
	if fileutil.IsDirectory(hostSnapshotFileOrDir) {
		// Find out the MariaDB version that correlates to the snapshot.
		versionFile := filepath.Join(hostSnapshotFileOrDir, "db_mariadb_version.txt")
		if fileutil.FileExists(versionFile) {
			snapshotDBVersion, err = fileutil.ReadFileIntoString(versionFile)
			if err != nil {
				return fmt.Errorf("unable to read the version file in the snapshot (%s): %v", versionFile, err)
			}
			snapshotDBVersion = strings.Trim(snapshotDBVersion, "\r\n\t ")
			snapshotDBVersion = fullDBFromVersion(snapshotDBVersion)
		} else {
			snapshotDBVersion = "unknown"
		}
	} else {
		// Extract the DB type/version from the filename, supporting both .gz and .zst
		m1 := regexp.MustCompile(`((mysql|mariadb|postgres)_[0-9.]+)\.(gz|zst)$`)
		matches := m1.FindStringSubmatch(snapshotFile)
		if len(matches) > 2 {
			snapshotDBVersion = matches[1]
		} else {
			return fmt.Errorf("unable to determine database type/version from snapshot %s", snapshotFile)
		}

		if !(strings.HasPrefix(snapshotDBVersion, "mariadb_") || strings.HasPrefix(snapshotDBVersion, "mysql_") || strings.HasPrefix(snapshotDBVersion, "postgres_")) {
			return fmt.Errorf("unable to determine database type/version from snapshot name %s", snapshotFile)
		}
	}

	if snapshotDBVersion != currentDBVersion {
		return fmt.Errorf("snapshot '%s' is a DB server '%s' snapshot and is not compatible with the configured DDEV DB server version (%s). Please restore it using the DB version it was created with, and then you can try upgrading the DDEV DB version", snapshotName, snapshotDBVersion, currentDBVersion)
	}

	status, _ := app.SiteStatus()
	start := time.Now()

	// For mariadb/mysql restart container and wait for restore
	if status == SiteRunning || status == SitePaused {
		util.Success("Stopping db container for snapshot restore of '%s'...", snapshotFile)
		maxWaitTime := max(SnapshotRestoreDefaultWaitTime, app.GetMaxContainerWaitTime())
		util.Success("With large snapshots this may take a long time.\nThis may time out after %d seconds \nbut you can increase it by changing default_container_timeout.", maxWaitTime)
		dbContainer, err := GetContainer(app, "db")
		if err != nil || dbContainer == nil {
			return fmt.Errorf("no container found for db; err=%v", err)
		}
		err = dockerutil.RemoveContainer(dbContainer.ID)
		if err != nil {
			return fmt.Errorf("failed to remove db container: %v", err)
		}
	}

	// If we have no bind mounts, we need to copy our snapshot into the snapshots volume
	// With bind mounts, the snapshot must exist under the current project's .ddev/db_snapshots
	// If the snapshot comes from a different worktree (sourceRoot != app.AppRoot) and
	// bind mounts are used, copy the snapshot into the current project's .ddev/db_snapshots
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		uid, _, _ := dockerutil.GetContainerUser()

		// If the snapshot is an old-style directory-based snapshot, then we have to copy into a subdirectory
		// named for the snapshot
		subdir := ""
		if fileutil.IsDirectory(hostSnapshotFileOrDir) {
			subdir = snapshotName
		}

		err = dockerutil.CopyIntoVolume(filepath.Join(snapshotDir, snapshotFile), "ddev-"+app.Name+"-snapshots", subdir, uid, "", true)
		if err != nil {
			return err
		}
	} else if !samePath(sourceRoot, app.AppRoot) {
		// Bind mounts are used but the snapshot is in another worktree; copy it into
		// this project's .ddev/db_snapshots so the bind mount will expose it to the container.
		destSnapshotsDir := filepath.Join(app.AppRoot, ".ddev", "db_snapshots")
		if !fileutil.FileExists(destSnapshotsDir) {
			if err := os.MkdirAll(destSnapshotsDir, 0o755); err != nil {
				return fmt.Errorf("failed to create snapshots dir %s: %v", destSnapshotsDir, err)
			}
		}
		srcPath := filepath.Join(snapshotDir, snapshotFile)
		destPath := filepath.Join(destSnapshotsDir, filepath.Base(srcPath))
		// Remove any existing destination before copying
		if fileutil.FileExists(destPath) {
			if err := os.RemoveAll(destPath); err != nil {
				return fmt.Errorf("failed to remove existing snapshot at %s: %v", destPath, err)
			}
		}
		if fileutil.IsDirectory(srcPath) {
			if err := fileutil.CopyDir(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy snapshot directory from %s to %s: %v", srcPath, destPath, err)
			}
		} else {
			if err := fileutil.CopyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("failed to copy snapshot file from %s to %s: %v", srcPath, destPath, err)
			}
		}
	}

	restoreCmd := "restore_snapshot " + snapshotFile
	// Determine compression type for potential conditional restore handling
	isGzip := strings.HasSuffix(snapshotFile, ".gz")
	isZstd := strings.HasSuffix(snapshotFile, ".zst")
	if isGzip {
		// MariaDB 5.5 does not support zstd compression
		isMariaDB55 := app.Database.Type == nodeps.MariaDB && app.Database.Version == nodeps.MariaDB55
		if !isMariaDB55 {
			util.Warning("This snapshot uses gzip compression. Creating a new snapshot will automatically use faster zstd compression.")
		}
	}
	if app.Database.Type == nodeps.Postgres {
		postgresDataDir := app.GetPostgresDataDir()
		postgresDataPath := app.GetPostgresDataPath()
		confdDir := path.Join(nodeps.PostgresConfigDir, "conf.d")
		v, _ := strconv.Atoi(app.Database.Version)
		// Choose proper tar flags based on compression
		tarExtract := "-zxf" // gzip default
		if isZstd {
			tarExtract = fmt.Sprintf(`-I "%s" -xf`, app.GetDBCompressionCommand())
		}
		// PostgreSQL 18+ requires restore_command parameter, older versions use recovery.conf
		if v >= 18 {
			restoreCmd = fmt.Sprintf(`bash -c 'chmod 700 %s && mkdir -p %s && rm -rf %s/* && tar -C %s %s /mnt/snapshots/%s && chmod 700 %s && touch %s/recovery.signal && postgres -c config_file=%s/postgresql.conf -c hba_file=%s/pg_hba.conf -c restore_command=true'`, postgresDataDir, confdDir, postgresDataDir, postgresDataDir, tarExtract, snapshotFile, postgresDataPath, postgresDataPath, nodeps.PostgresConfigDir, nodeps.PostgresConfigDir)
		} else {
			targetConfName := path.Join(confdDir, "recovery.conf")
			// Before PostgreSQL v12 the recovery info went into its own file
			if v < 12 {
				targetConfName = path.Join(nodeps.PostgresConfigDir, "recovery.conf")
			}
			restoreCmd = fmt.Sprintf(`bash -c 'chmod 700 %s && mkdir -p %s && rm -rf %s/* && tar -C %s %s /mnt/snapshots/%s && chmod 700 %s && touch %s/recovery.signal && echo "restore_command = 'true'" >>%s && postgres -c config_file=%s/postgresql.conf -c hba_file=%s/pg_hba.conf'`, postgresDataDir, confdDir, postgresDataDir, postgresDataDir, tarExtract, snapshotFile, postgresDataPath, postgresDataPath, targetConfName, nodeps.PostgresConfigDir, nodeps.PostgresConfigDir)
		}
	}
	_ = os.Setenv("DDEV_DB_CONTAINER_COMMAND", restoreCmd)
	// nolint: errcheck
	defer os.Unsetenv("DDEV_DB_CONTAINER_COMMAND")
	// If the default_container_timeout does not already specify a longer period
	// then allow extra time by default for the snapshot restore. This is arbitrary but may help.
	origTimeout := app.DefaultContainerTimeout
	if t, _ := strconv.Atoi(app.DefaultContainerTimeout); t <= SnapshotRestoreDefaultWaitTime {
		app.DefaultContainerTimeout = strconv.Itoa(SnapshotRestoreDefaultWaitTime)
	}
	err = app.Start()
	if err != nil {
		return fmt.Errorf("failed to start project for RestoreSnapshot: %v", err)
	}

	// On mysql/mariadb the snapshot restore doesn't actually complete right away after
	// the mariabackup/xtrabackup returns.
	if app.Database.Type != nodeps.Postgres {
		output.UserOut.Printf("Waiting up to %ss for snapshot restore to complete...\nYou can also follow the restore progress in another terminal window with `ddev logs -s db -f %s`", app.DefaultContainerTimeout, app.Name)
		// Now it's up, but we need to find out when it finishes loading.
		for {
			// We used to use killall -1 mysqld here
			// also used to use "pidof mysqld", but apparently the
			// server may not quite be ready when its pid appears
			out, _, err := app.Exec(&ExecOpts{
				Cmd:     fmt.Sprintf(`(echo "SHOW VARIABLES like 'v%%';" | %s 2>/dev/null) || true`, app.GetDBClientCommand()),
				Service: "db",
				Tty:     false,
			})
			if err != nil {
				return err
			}
			if out != "" {
				break
			}
			time.Sleep(1 * time.Second)
			if !output.JSONOutput {
				fmt.Print(".")
			}
		}
	}
	app.DefaultContainerTimeout = origTimeout
	util.Success("\nDatabase snapshot %s was restored in %vs", snapshotName, int(time.Since(start).Seconds()))
	err = app.ProcessHooks("post-restore-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process post-restore-snapshot hooks: %v", err)
	}
	return nil
}

func getSnapshotFileFromNameFromRoot(name, sourceRoot string) (string, error) {
	snapshotsDir := filepath.Join(sourceRoot, ".ddev", "db_snapshots")
	snapshotFullPath := filepath.Join(snapshotsDir, name)

	// If old-style directory-based snapshot, then use the name, no massaging required
	if fileutil.IsDirectory(snapshotFullPath) {
		return name, nil
	}

	// But if it's a compressed tarball, we have to get the filename.
	files, err := fileutil.ListFilesInDir(snapshotsDir)
	if err != nil {
		return "", err
	}

	m := regexp.MustCompile("^" + regexp.QuoteMeta(name) + `-(mariadb|mysql|postgres)_[0-9.]*\.(gz|zst)$`)

	for _, file := range files {
		if m.MatchString(file) {
			return file, nil
		}
	}

	return "", fmt.Errorf("snapshot %s not found in %s", name, snapshotsDir)
}

// GetSnapshotFileFromName returns the filename corresponding to the snapshot name
func GetSnapshotFileFromName(name string, app *DdevApp) (string, error) {
	return getSnapshotFileFromNameFromRoot(name, app.AppRoot)
}
