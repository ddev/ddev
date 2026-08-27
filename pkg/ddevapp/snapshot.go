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
	Name        string
	Created     time.Time
	Size        int64
	DBVersion   string
	Compression string
}

// snapshotExtensions lists recognized snapshot file suffixes: the compressed
// formats (zst, gz) followed by the uncompressed raw stream formats
// (mbstream from mariabackup, xbstream from xtrabackup).
var snapshotExtensions = []string{"zst", "gz", "mbstream", "xbstream"}

// snapshotExtensionPattern is the `(zst|gz|mbstream|xbstream)`-style
// alternation used to recognize a snapshot filename's extension.
var snapshotExtensionPattern = "(" + strings.Join(snapshotExtensions, "|") + ")"

// hasSnapshotExtension reports whether name ends in one of snapshotExtensions.
func hasSnapshotExtension(name string) bool {
	for _, ext := range snapshotExtensions {
		if strings.HasSuffix(name, "."+ext) {
			return true
		}
	}
	return false
}

// snapshotCompressionLabel returns the human-readable compression type for a
// snapshot file extension, for display in `ddev snapshot --list`.
func snapshotCompressionLabel(ext string) string {
	switch ext {
	case "zst":
		return "zstd"
	case "gz":
		return "gzip"
	default:
		return "none"
	}
}

// SnapshotRestoreDefaultWaitTime is the max time we'll wait for snapshot restore.
// If default_container_timeout is set higher than that it can be more
const SnapshotRestoreDefaultWaitTime = 600

// RestoreSnapshotCommand is the ddev-dbserver entrypoint argument that makes it
// restore a snapshot instead of looking for a base_db seed.
// See containers/ddev-dbserver/files/docker-entrypoint.sh
const RestoreSnapshotCommand = "restore_snapshot"

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

// SnapshotRestoreTarget is a snapshot offered by `ddev snapshot restore`,
// either the current project's own or one found in the same project checked
// out in another git worktree of the same repository.
type SnapshotRestoreTarget struct {
	// Name is the snapshot name, as shown by `ddev snapshot --list`.
	Name string
	// Label is what the interactive picker displays for this target; it
	// names the source worktree when the snapshot isn't the current project's own.
	Label string
	// Path is what to pass to RestoreSnapshot: a bare name for the current
	// project's own snapshot, an absolute file path for one found elsewhere.
	Path string
	// SourceRoot is the project root the snapshot was found under.
	SourceRoot  string
	Created     time.Time
	Size        int64
	DBVersion   string
	Compression string
}

// ListSnapshotRestoreTargets returns the current project's own snapshots plus
// any snapshots belonging to the same project checked out in other git
// worktrees of the same repository, newest first.
func (app *DdevApp) ListSnapshotRestoreTargets() ([]SnapshotRestoreTarget, error) {
	var targets []SnapshotRestoreTarget

	own, err := app.ListSnapshots()
	if err != nil {
		return nil, err
	}
	for _, s := range own {
		targets = append(targets, SnapshotRestoreTarget{
			Name:        s.Name,
			Label:       s.Name,
			Path:        s.Name,
			SourceRoot:  app.AppRoot,
			Created:     s.Created,
			Size:        s.Size,
			DBVersion:   s.DBVersion,
			Compression: s.Compression,
		})
	}

	for _, projectRoot := range app.otherWorktreeProjectRoots() {
		snapshotsDir := filepath.Join(projectRoot, ".ddev", "db_snapshots")
		snapshots, err := listSnapshotsInDir(snapshotsDir)
		if err != nil || len(snapshots) == 0 {
			continue
		}
		label := filepath.Base(projectRoot)
		for _, s := range snapshots {
			file, err := snapshotFileInDir(s.Name, snapshotsDir)
			if err != nil {
				continue
			}
			fullPath := filepath.Join(snapshotsDir, file)
			// Path-based restore (see resolveSnapshotSource) only supports
			// snapshot files, not the legacy directory-based format.
			if fileutil.IsDirectory(fullPath) {
				continue
			}
			targets = append(targets, SnapshotRestoreTarget{
				Name:        s.Name,
				Label:       fmt.Sprintf("%s (%s)", s.Name, label),
				Path:        fullPath,
				SourceRoot:  projectRoot,
				Created:     s.Created,
				Size:        s.Size,
				DBVersion:   s.DBVersion,
				Compression: s.Compression,
			})
		}
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Created.After(targets[j].Created)
	})

	return targets, nil
}

// GetLatestSnapshotRestoreTarget returns the most recently created snapshot
// among the current project's own snapshots and any found in sibling git
// worktrees of the same project.
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

// ResolveSnapshotRestoreTarget resolves a bare snapshot name given directly on
// the `ddev snapshot restore` command line to what should be passed to
// RestoreSnapshot: the name unchanged if the project has its own snapshot by
// that name (or nothing matches at all, so the original "not found" error is
// preserved), otherwise the absolute path of the newest same-named snapshot
// found in a sibling git worktree of the same project. A path (absolute, or
// containing a slash) is returned unchanged, since it already names an exact
// file rather than something worktree lookup could resolve.
func (app *DdevApp) ResolveSnapshotRestoreTarget(nameOrPath string) string {
	if filepath.IsAbs(nameOrPath) || strings.ContainsAny(nameOrPath, `/\`) {
		return nameOrPath
	}
	if _, err := GetSnapshotFileFromName(nameOrPath, app); err == nil {
		return nameOrPath
	}
	targets, err := app.ListSnapshotRestoreTargets()
	if err != nil {
		return nameOrPath
	}
	for _, target := range targets {
		if target.Name == nameOrPath && target.SourceRoot != app.AppRoot {
			return target.Path
		}
	}
	return nameOrPath
}

// otherWorktreeProjectRoots returns the directories where this same project
// would live in every other git worktree of its repository: the project's
// path relative to its own worktree root, re-joined onto each of the
// repository's other worktree roots. It returns nil if the project isn't in a
// git worktree, or the repository has no other worktrees - callers should
// treat that as "nothing extra to offer", not an error.
func (app *DdevApp) otherWorktreeProjectRoots() []string {
	// Not being in a git repo at all is the common case here, so it's not
	// worth a debug log - only the later checks, reached once we know we're
	// in a repo, are.
	repoRootOutput, err := exec.RunHostCommand("git", "-C", app.AppRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	repoRoot := strings.TrimSpace(repoRootOutput)
	if repoRoot == "" {
		return nil
	}

	// git's own idea of app.AppRoot's path relative to repoRoot, rather than
	// filepath.Rel(repoRoot, app.AppRoot): on macOS in particular, app.AppRoot
	// may reach the project through a symlink (e.g. /var/folders/... into
	// /private/var/folders/...) that "rev-parse --show-toplevel" already
	// resolved, so comparing the two paths directly can misjudge how deep the
	// project sits under the repo root.
	prefixOutput, err := exec.RunHostCommand("git", "-C", app.AppRoot, "rev-parse", "--show-prefix")
	if err != nil {
		util.Debug("otherWorktreeProjectRoots: git rev-parse --show-prefix: %v", err)
		return nil
	}
	relToRepo := strings.TrimSuffix(strings.TrimSpace(prefixOutput), "/")

	worktreeListOutput, err := exec.RunHostCommand("git", "-C", repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		util.Debug("otherWorktreeProjectRoots: git worktree list: %v", err)
		return nil
	}

	var projectRoots []string
	for line := range strings.SplitSeq(strings.TrimSpace(worktreeListOutput), "\n") {
		worktreeRoot, ok := strings.CutPrefix(line, "worktree ")
		if !ok {
			continue
		}
		worktreeRoot = strings.TrimSpace(worktreeRoot)
		if worktreeRoot == "" || worktreeRoot == repoRoot {
			continue
		}
		projectRoots = append(projectRoots, filepath.Join(worktreeRoot, relToRepo))
	}

	return projectRoots
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

// ListSnapshots returns a list of all project snapshots
func (app *DdevApp) ListSnapshots() ([]Snapshot, error) {
	return listSnapshotsInDir(app.GetConfigPath("db_snapshots"))
}

// listSnapshotsInDir returns all snapshots found directly in snapshotDir. It's
// the directory-parameterized core of ListSnapshots, also used to look for
// snapshots belonging to the same project checked out in other git worktrees.
func listSnapshotsInDir(snapshotDir string) ([]Snapshot, error) {
	var err error
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

	// Match snapshot files, capturing the DB type/version and the extension
	// that identifies the compression (or lack of it).
	m := regexp.MustCompile(`-(mariadb|mysql|postgres)_([0-9.]*)\.` + snapshotExtensionPattern + `$`)

	for _, f := range files {
		if f.IsDir() || hasSnapshotExtension(f.Name()) {
			n := m.ReplaceAll([]byte(f.Name()), []byte(""))
			size := f.Size()
			dbVersion := "unknown"
			compression := "none"
			if f.IsDir() {
				var err error
				size, err = fileutil.DirSize(filepath.Join(snapshotDir, f.Name()))
				if err != nil {
					return snapshots, err
				}
				versionFile := filepath.Join(snapshotDir, f.Name(), "db_mariadb_version.txt")
				if fileutil.FileExists(versionFile) {
					if v, err := fileutil.ReadFileIntoString(versionFile); err == nil {
						if full := fullDBFromVersion(strings.Trim(v, "\r\n\t ")); full != "" {
							dbVersion = full
						}
					}
				}
			} else if matches := m.FindStringSubmatch(f.Name()); len(matches) > 3 {
				dbVersion = matches[1] + "_" + matches[2]
				compression = snapshotCompressionLabel(matches[3])
			}
			snapshot := Snapshot{
				Name:        string(n),
				Created:     f.ModTime(),
				Size:        size,
				DBVersion:   dbVersion,
				Compression: compression,
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots, nil
}

// snapshotDBVersionFromFilename returns the db type/version a snapshot file
// name encodes, like "mariadb_11.8". Snapshot files are always named
// <name>-<type>_<version>.<ext> by Snapshot(), for any of snapshotExtensions.
func snapshotDBVersionFromFilename(snapshotFile string) (string, error) {
	matches := regexp.MustCompile(`((mysql|mariadb|postgres)_[0-9.]+)\.` + snapshotExtensionPattern + `$`).FindStringSubmatch(snapshotFile)
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to determine database type/version from snapshot %s", snapshotFile)
	}
	return matches[1], nil
}

// dbTypeFromVersion returns the "mariadb" of a "mariadb_11.8".
func dbTypeFromVersion(dbVersion string) string {
	dbType, _, _ := strings.Cut(dbVersion, "_")
	return dbType
}

// RestoreSnapshot restores a MariaDB snapshot of the db to be loaded
// The project must be stopped and Docker volume removed and recreated for this to work.
//
// force allows restoring a snapshot taken on a different version of the same
// database server, which the underlying restore does not itself check for.
func (app *DdevApp) RestoreSnapshot(snapshotName string, force bool) error {
	var err error
	err = app.ProcessHooks("pre-restore-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process pre-restore-snapshot hooks: %v", err)
	}

	currentDBVersion := app.Database.Type + "_" + app.Database.Version

	hostSnapshotFileOrDir, mountDir, err := resolveSnapshotSource(snapshotName, app)
	if err != nil {
		return fmt.Errorf("unable to resolve snapshot %s: %v", snapshotName, err)
	}
	snapshotFile := filepath.Base(hostSnapshotFileOrDir)

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
		// Extract the DB type/version from the filename
		m1 := regexp.MustCompile(`((mysql|mariadb|postgres)_[0-9.]+)\.` + snapshotExtensionPattern + `$`)
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
		if !force {
			return fmt.Errorf("snapshot '%s' is a DB server '%s' snapshot and is not compatible with the configured DDEV DB server version (%s).  Please restore it using the DB version it was created with, and then you can try upgrading the DDEV DB version, or use --force to attempt the restore anyway", snapshotName, snapshotDBVersion, currentDBVersion)
		}
		if dbTypeFromVersion(snapshotDBVersion) != app.Database.Type {
			return fmt.Errorf("snapshot '%s' is a '%s' snapshot, which cannot be restored into a '%s' database even with --force", snapshotName, snapshotDBVersion, currentDBVersion)
		}
		util.Warning("Restoring a '%s' snapshot into a '%s' database because --force was used.\nThe database may fail to come up, in which case you can start over with `ddev start --reset-database`.", snapshotDBVersion, currentDBVersion)
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

	// If we have no bind mounts, we need to copy our snapshot into the snapshots volme
	// With bind mounts, they'll already be there in the /mnt/ddev_config/db_snapshots folder
	if globalconfig.DdevGlobalConfig.NoBindMounts {
		uid, _, _ := dockerutil.GetContainerUser()

		// If the snapshot is an old-style directory-based snapshot, then we have to copy into a subdirectory
		// named for the snapshot
		subdir := ""
		if fileutil.IsDirectory(hostSnapshotFileOrDir) {
			subdir = snapshotName
		}

		err = dockerutil.CopyIntoVolume(hostSnapshotFileOrDir, "ddev-"+app.Name+"-snapshots", subdir, uid, "", true)
		if err != nil {
			return err
		}
	}

	if strings.HasSuffix(snapshotFile, ".gz") && !(app.Database.Type == nodeps.MariaDB && app.Database.Version == nodeps.MariaDB55) {
		// MariaDB 5.5 is the only db version that can't make a zstd snapshot
		util.Warning("This snapshot uses gzip compression. Creating a new snapshot will automatically use faster zstd compression.")
	}

	containerPath := snapshotFile
	if mountDir != "" {
		containerPath = path.Join(SeedSnapshotMountDir, snapshotFile)
	}
	app.restoreSnapshotMountDir = mountDir
	defer func() { app.restoreSnapshotMountDir = "" }()

	_ = os.Setenv("DDEV_DB_CONTAINER_COMMAND", app.snapshotRestoreContainerCommand(containerPath))
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

// snapshotRestoreContainerCommand returns the db container command that restores
// a snapshot into the data directory, replacing whatever is there. snapshotPath
// is the snapshot as the db container sees it: either a bare file/directory name
// found in /mnt/snapshots, or an absolute path such as a seed snapshot mounted at
// SeedSnapshotMountDir.
//
// The command runs before the database server starts, so it works the same on an
// empty data directory (seeding a brand-new volume) as on a populated one.
func (app *DdevApp) snapshotRestoreContainerCommand(snapshotPath string) string {
	if app.Database.Type != nodeps.Postgres {
		return RestoreSnapshotCommand + " " + snapshotPath
	}

	if !path.IsAbs(snapshotPath) {
		snapshotPath = path.Join("/mnt/snapshots", snapshotPath)
	}
	tarExtract := "-zxf"
	if strings.HasSuffix(snapshotPath, ".zst") {
		tarExtract = fmt.Sprintf(`-I "%s" -xf`, app.GetDBCompressionCommand())
	}
	dataDir := app.GetPostgresDataDir()
	dataPath := app.GetPostgresDataPath()
	confdDir := path.Join(nodeps.PostgresConfigDir, "conf.d")
	// mkdir of the data directory matters on a brand-new volume, where PostgreSQL
	// 18+ mounts the volume a level above a data directory that doesn't exist yet.
	prepare := fmt.Sprintf(`mkdir -p %s %s && chmod 700 %s && rm -rf %s/* && tar -C %s %s %s && chmod 700 %s && touch %s/recovery.signal`, dataDir, confdDir, dataDir, dataDir, dataDir, tarExtract, snapshotPath, dataPath, dataPath)
	serve := fmt.Sprintf(`postgres -c config_file=%s/postgresql.conf -c hba_file=%s/pg_hba.conf`, nodeps.PostgresConfigDir, nodeps.PostgresConfigDir)

	v, _ := strconv.Atoi(app.Database.Version)
	// PostgreSQL 18+ takes restore_command as a server parameter; before that it
	// goes in recovery.conf, which moved into conf.d in v12.
	if v >= 18 {
		return fmt.Sprintf(`bash -c '%s && %s -c restore_command=true'`, prepare, serve)
	}
	targetConfName := path.Join(confdDir, "recovery.conf")
	if v < 12 {
		targetConfName = path.Join(nodeps.PostgresConfigDir, "recovery.conf")
	}
	return fmt.Sprintf(`bash -c '%s && echo "restore_command = 'true'" >>%s && %s'`, prepare, targetConfName, serve)
}

// resolveSnapshotSource resolves a snapshot name or host path to its absolute
// file on disk and, if it lives outside .ddev/db_snapshots, the directory that
// needs to be bind-mounted into the db container so the container can reach it.
//
// A path (absolute, or containing a slash) must already exist as a file, not a
// directory. A bare name is resolved inside .ddev/db_snapshots via
// GetSnapshotFileFromName, which also supports old-style directory snapshots -
// something an external path does not.
func resolveSnapshotSource(nameOrPath string, app *DdevApp) (hostPath string, mountDir string, err error) {
	if filepath.IsAbs(nameOrPath) || strings.ContainsAny(nameOrPath, `/\`) {
		// Shell doesn't expand ~ inside --flag=value, so it reaches us literally.
		expanded, err := util.ExpandHomedir(nameOrPath)
		if err != nil {
			return "", "", fmt.Errorf("unable to resolve %s: %v", nameOrPath, err)
		}
		if hostPath, err = filepath.Abs(expanded); err != nil {
			return "", "", fmt.Errorf("unable to resolve %s: %v", nameOrPath, err)
		}
		if !fileutil.FileExists(hostPath) || fileutil.IsDirectory(hostPath) {
			return "", "", fmt.Errorf("%s is not an existing snapshot file", nameOrPath)
		}
		// A snapshot already in .ddev/db_snapshots needs no mount of its own, and
		// with no_bind_mounts the caller copies it in rather than mounting.
		if dir := filepath.Dir(hostPath); dir != app.GetConfigPath("db_snapshots") && !globalconfig.DdevGlobalConfig.NoBindMounts {
			mountDir = dir
		}
		return hostPath, mountDir, nil
	}

	snapshotFile, err := GetSnapshotFileFromName(nameOrPath, app)
	if err != nil {
		return "", "", err
	}
	return app.GetConfigPath(filepath.Join("db_snapshots", snapshotFile)), "", nil
}

// GetSnapshotFileFromName returns the filename corresponding to the snapshot name
func GetSnapshotFileFromName(name string, app *DdevApp) (string, error) {
	return snapshotFileInDir(name, app.GetConfigPath("db_snapshots"))
}

// snapshotFileInDir returns the actual filename of the snapshot called name
// inside snapshotsDir: the name unchanged for an old-style directory-based
// snapshot, or the full <name>-<type>_<version>.<ext> filename for a modern one.
func snapshotFileInDir(name, snapshotsDir string) (string, error) {
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

	m := regexp.MustCompile("^" + regexp.QuoteMeta(name) + `-(mariadb|mysql|postgres)_[0-9.]*\.` + snapshotExtensionPattern + `$`)

	for _, file := range files {
		if m.MatchString(file) {
			return file, nil
		}
	}

	return "", fmt.Errorf("snapshot %s not found in %s", name, snapshotsDir)
}
