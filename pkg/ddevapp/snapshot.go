package ddevapp

import (
	"fmt"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

// GetLatestSnapshot returns the latest created snapshot of a project
func (app *DdevApp) GetLatestSnapshot() (string, error) {
	var snapshots []string

	snapshots, err := app.ListSnapshots()
	if err != nil {
		return "", err
	}

	if len(snapshots) == 0 {
		return "", fmt.Errorf("no snapshots found")
	}

	return snapshots[0], nil
}

// ListSnapshots returns a list of the names of all project snapshots
func (app *DdevApp) ListSnapshots() ([]string, error) {
	var err error
	var snapshots []string

	snapshotDir := app.GetConfigPath("db_snapshots")

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

	m := regexp.MustCompile(`-(mariadb|mysql|postgres)_[0-9.]*\.gz$`)

	for _, f := range files {
		if f.IsDir() || strings.HasSuffix(f.Name(), ".gz") {
			n := m.ReplaceAll([]byte(f.Name()), []byte(""))
			snapshots = append(snapshots, string(n))
		}
	}

	return snapshots, nil
}

// RestoreSnapshot restores a mariadb snapshot of the db to be loaded
// The project must be stopped and docker volume removed and recreated for this to work.
func (app *DdevApp) RestoreSnapshot(snapshotName string) error {
	var err error
	err = app.ProcessHooks("pre-restore-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process pre-restore-snapshot hooks: %v", err)
	}

	currentDBVersion := app.Database.Type + "_" + app.Database.Version

	snapshotFile, err := GetSnapshotFileFromName(snapshotName, app)
	if err != nil {
		return fmt.Errorf("no snapshot found for name %s: %v", snapshotName, err)
	}
	snapshotFileOrDir := filepath.Join("db_snapshots", snapshotFile)

	hostSnapshotFileOrDir := app.GetConfigPath(snapshotFileOrDir)

	if !fileutil.FileExists(hostSnapshotFileOrDir) {
		return fmt.Errorf("failed to find a snapshot at %s", hostSnapshotFileOrDir)
	}

	snapshotDBVersion := ""

	// If the snapshot is a directory, (old obsolete style) then
	// look for db_mariadb_version.txt in the directory to get the version.
	if fileutil.IsDirectory(hostSnapshotFileOrDir) {
		// Find out the mariadb version that correlates to the snapshot.
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
		m1 := regexp.MustCompile(`((mysql|mariadb|postgres)_[0-9.]+)\.gz$`)
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
		return fmt.Errorf("snapshot '%s' is a DB server '%s' snapshot and is not compatible with the configured ddev DB server version (%s).  Please restore it using the DB version it was created with, and then you can try upgrading the ddev DB version", snapshotName, snapshotDBVersion, currentDBVersion)
	}

	status, _ := app.SiteStatus()
	start := time.Now()

	// For mariadb/mysql restart container and wait for restore
	if status == SiteRunning || status == SitePaused {
		util.Success("Stopping db container for snapshot restore of '%s'...", snapshotFile)
		util.Success("With large snapshots this may take a long time.\nThis will normally time out after %d seconds (max of all container timeouts)\nbut you can increase it by changing default_container_timeout.", app.FindMaxTimeout())
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
		uid, _, _ := util.GetContainerUIDGid()
		// for postgres, must be written with postgres user
		if app.Database.Type == nodeps.Postgres {
			uid = "999"
		}

		// If the snapshot is an old-style directory-based snapshot, then we have to copy into a subdirectory
		// named for the snapshot
		subdir := ""
		if fileutil.IsDirectory(hostSnapshotFileOrDir) {
			subdir = snapshotName
		}

		err = dockerutil.CopyIntoVolume(filepath.Join(app.GetConfigPath("db_snapshots"), snapshotFile), "ddev-"+app.Name+"-snapshots", subdir, uid, "", true)
		if err != nil {
			return err
		}
	}

	restoreCmd := "restore_snapshot " + snapshotFile
	if app.Database.Type == nodeps.Postgres {
		confdDir := path.Join(nodeps.PostgresConfigDir, "conf.d")
		targetConfName := path.Join(confdDir, "recovery.conf")
		v, _ := strconv.Atoi(app.Database.Version)
		// Before postgres v12 the recovery info went into its own file
		if v < 12 {
			targetConfName = path.Join(nodeps.PostgresConfigDir, "recovery.conf")
		}
		restoreCmd = fmt.Sprintf(`bash -c 'chmod 700 /var/lib/postgresql/data && mkdir -p %s && rm -rf /var/lib/postgresql/data/* && tar -C /var/lib/postgresql/data -zxf /mnt/snapshots/%s && touch /var/lib/postgresql/data/recovery.signal && cat /var/lib/postgresql/recovery.conf >>%s && postgres -c config_file=%s/postgresql.conf -c hba_file=%s/pg_hba.conf'`, confdDir, snapshotFile, targetConfName, nodeps.PostgresConfigDir, nodeps.PostgresConfigDir)
	}
	_ = os.Setenv("DDEV_DB_CONTAINER_COMMAND", restoreCmd)
	// nolint: errcheck
	defer os.Unsetenv("DDEV_DB_CONTAINER_COMMAND")
	err = app.Start()
	if err != nil {
		return fmt.Errorf("failed to start project for RestoreSnapshot: %v", err)
	}

	// On mysql/mariadb the snapshot restore doesn't actually complete right away after
	// the mariabackup/xtrabackup returns.
	if app.Database.Type != nodeps.Postgres {
		output.UserOut.Printf("Waiting for snapshot restore to complete...\nYou can also follow the restore progress in another terminal window with `ddev logs -s db -f %s`", app.Name)
		// Now it's up, but we need to find out when it finishes loading.
		for {
			// We used to use killall -1 mysqld here
			// also used to use "pidof mysqld", but apparently the
			// server may not quite be ready when its pid appears
			out, _, err := app.Exec(&ExecOpts{
				Cmd:     `(echo "SHOW VARIABLES like 'v%';" | mysql 2>/dev/null) || true`,
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
			fmt.Print(".")
		}
	}
	util.Success("\nDatabase snapshot %s was restored in %vs", snapshotName, int(time.Since(start).Seconds()))
	err = app.ProcessHooks("post-restore-snapshot")
	if err != nil {
		return fmt.Errorf("failed to process post-restore-snapshot hooks: %v", err)
	}
	return nil
}

// GetSnapshotFileFromName returns the filename corresponding to the snapshot name
func GetSnapshotFileFromName(name string, app *DdevApp) (string, error) {
	snapshotsDir := app.GetConfigPath("db_snapshots")
	snapshotFullPath := filepath.Join(snapshotsDir, name)
	// If old-style directory-based snapshot, then just use the name, no massaging required
	if fileutil.IsDirectory(snapshotFullPath) {
		return name, nil
	}
	// But if it's a gzipped tarball, we have to get the filename.
	files, err := fileutil.ListFilesInDir(snapshotsDir)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if strings.HasPrefix(file, name+"-") {
			return file, nil
		}
	}
	return "", fmt.Errorf("snapshot %s not found in %s", name, snapshotsDir)
}
