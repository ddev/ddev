package ddevapp

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
)

// SeedSnapshotName is the reserved snapshot name that seeds a brand-new
// database volume instead of the stock starter database, used when
// `ddev start --seed-snapshot` names nothing else.
const SeedSnapshotName = "seed"

// SeedSnapshotMountDir is where the directory holding a seed snapshot from
// outside the project gets mounted in the db container. Snapshots living in
// .ddev/db_snapshots are already visible to it at /mnt/snapshots.
const SeedSnapshotMountDir = "/mnt/seed"

// CustomBaseDBSeedPathPrefix is where a derived dbimage bakes in its own
// base_db seed; ddev-dbserver checks it ahead of the stock /mysqlbase/base_db.*
const CustomBaseDBSeedPathPrefix = "/mysqlbase/custom/base_db"

// baseDBSeedExtensions lists the base_db seed compression suffixes in the same
// preference order that ddev-dbserver's docker-entrypoint.sh uses: the
// compressed formats first, then the uncompressed raw stream formats
// (mbstream from mariabackup, xbstream from xtrabackup).
var baseDBSeedExtensions = []string{"zst", "gz", "mbstream", "xbstream"}

// UsesBaseDBSeed reports whether this project's database volume gets initialized
// from a base_db seed. Only MariaDB/MySQL do; PostgreSQL uses the upstream
// postgres entrypoint.
func (app *DdevApp) UsesBaseDBSeed() bool {
	return !app.IsDBOmitted() && slices.Contains([]string{nodeps.MariaDB, nodeps.MySQL}, app.Database.Type)
}

// GetSeedSnapshotFile returns the host path of the project's reserved `seed`
// snapshot for the configured database type/version, or an empty string if
// there isn't one. Unlike a base_db seed baked into the dbimage, this works for
// every database type, because it's restored through the same container command
// `ddev snapshot restore` uses.
func (app *DdevApp) GetSeedSnapshotFile() string {
	if app.IsDBOmitted() {
		return ""
	}
	for _, ext := range baseDBSeedExtensions {
		f := app.GetConfigPath(filepath.Join("db_snapshots", fmt.Sprintf("%s-%s_%s.%s", SeedSnapshotName, app.Database.Type, app.Database.Version, ext)))
		if fileutil.FileExists(f) {
			return f
		}
	}
	return ""
}

// SeedSnapshot describes the snapshot a brand-new database volume gets seeded
// from, resolved from `--seed-snapshot` or from the reserved `seed` snapshot.
type SeedSnapshot struct {
	// HostPath is the absolute path of the snapshot file on the host.
	HostPath string
	// MountDir is the host directory to mount into the db container, empty when
	// the snapshot is already in .ddev/db_snapshots.
	MountDir string
}

// ContainerPath returns the snapshot file as the db container sees it.
func (s *SeedSnapshot) ContainerPath() string {
	if s.MountDir != "" {
		return path.Join(SeedSnapshotMountDir, filepath.Base(s.HostPath))
	}
	return path.Join("/mnt/snapshots", filepath.Base(s.HostPath))
}

// ResolveSeedSnapshot works out which snapshot should seed a brand-new database
// volume: the one named by `--seed-snapshot`, or the reserved `seed` snapshot if
// the flag wasn't used. It returns nil when there is nothing to seed from.
//
// The flag takes either a snapshot name resolved inside .ddev/db_snapshots, or a
// path to a snapshot file elsewhere. Either way the file has to keep the standard
// `-<type>_<version>.{zst,gz}` suffix, so a snapshot from a different database
// can't be dropped into a fresh volume it will not work in.
func (app *DdevApp) ResolveSeedSnapshot() (*SeedSnapshot, error) {
	if app.IsDBOmitted() {
		if app.SeedSnapshot != "" {
			return nil, fmt.Errorf("--seed-snapshot is not usable on project %s because its database is omitted", app.Name)
		}
		return nil, nil
	}

	if app.SeedSnapshot == "" {
		if f := app.GetSeedSnapshotFile(); f != "" {
			return &SeedSnapshot{HostPath: f}, nil
		}
		return nil, nil
	}

	hostPath := app.SeedSnapshot
	mountDir := ""
	if filepath.IsAbs(hostPath) || strings.ContainsAny(hostPath, `/\`) {
		var err error
		if hostPath, err = filepath.Abs(hostPath); err != nil {
			return nil, fmt.Errorf("unable to resolve --seed-snapshot=%s: %v", app.SeedSnapshot, err)
		}
		if !fileutil.FileExists(hostPath) || fileutil.IsDirectory(hostPath) {
			return nil, fmt.Errorf("--seed-snapshot=%s is not an existing snapshot file", app.SeedSnapshot)
		}
		// A snapshot already in .ddev/db_snapshots needs no mount of its own, and
		// with no_bind_mounts prepareSeedSnapshot copies it in rather than mounting.
		if dir := filepath.Dir(hostPath); dir != app.GetConfigPath("db_snapshots") && !globalconfig.DdevGlobalConfig.NoBindMounts {
			mountDir = dir
		}
	} else {
		snapshotFile, err := GetSnapshotFileFromName(app.SeedSnapshot, app)
		if err != nil {
			return nil, fmt.Errorf("unable to use --seed-snapshot=%s: %v", app.SeedSnapshot, err)
		}
		hostPath = app.GetConfigPath(filepath.Join("db_snapshots", snapshotFile))
	}

	snapshotDBVersion, err := snapshotDBVersionFromFilename(filepath.Base(hostPath))
	if err != nil {
		return nil, fmt.Errorf("unable to use --seed-snapshot=%s: %v", app.SeedSnapshot, err)
	}
	if currentDBVersion := app.Database.Type + "_" + app.Database.Version; snapshotDBVersion != currentDBVersion {
		return nil, fmt.Errorf("--seed-snapshot=%s is a '%s' snapshot, which cannot seed a '%s' database", app.SeedSnapshot, snapshotDBVersion, currentDBVersion)
	}

	return &SeedSnapshot{HostPath: hostPath, MountDir: mountDir}, nil
}

// GetDBBuildDockerfiles returns the global and project db-build Dockerfiles.
func (app *DdevApp) GetDBBuildDockerfiles() []string {
	var dockerfiles []string
	for _, dir := range []string{filepath.Join(globalconfig.GetGlobalDdevDir(), "db-build"), app.GetConfigPath("db-build")} {
		found, err := getBuildDockerfilesInDir(dir)
		if err != nil {
			continue
		}
		dockerfiles = append(dockerfiles, found...)
	}
	return dockerfiles
}

// GetDBBuildSeedDockerfiles returns the db-build Dockerfiles that bake a
// base_db seed into the derived dbimage.
func (app *DdevApp) GetDBBuildSeedDockerfiles() []string {
	if !app.UsesBaseDBSeed() {
		return nil
	}
	var seedDockerfiles []string
	for _, dockerfile := range app.GetDBBuildDockerfiles() {
		if found, err := fileutil.FgrepStringInFile(dockerfile, CustomBaseDBSeedPathPrefix); err == nil && found {
			seedDockerfiles = append(seedDockerfiles, dockerfile)
		}
	}
	return seedDockerfiles
}

// mayHaveDerivedDBImageSeed reports whether a base_db seed could possibly be
// baked into this project's dbimage: either the dbimage is an explicitly
// configured non-default one, or a db-build Dockerfile could COPY a seed in.
// Used to skip inspecting the image at all in the ordinary stock-image case.
func (app *DdevApp) mayHaveDerivedDBImageSeed() bool {
	if app.DBImage != "" && app.DBImage != docker.GetDBImage(app.Database.Type, app.Database.Version) {
		return true
	}
	return len(app.GetDBBuildDockerfiles()) > 0
}

// derivedBuiltImageRef returns the tag of the project's built "-<project>-built"
// image for a given base image, matching what `docker compose build` actually
// tags it as: the image string is used as-is as the new tag, and Docker
// defaults a tag-less repo to ":latest". dockerutil.RunSimpleContainer
// requires an explicit tag on its input, so a bare repo (e.g. `dbimage:
// randyfay/dbserver-2m` with no ":tag") needs ":latest" appended here too, or
// it's rejected before ever reaching Docker.
func derivedBuiltImageRef(baseImage, appName string) string {
	image := baseImage + "-" + appName + "-built"
	if !strings.Contains(image, ":") {
		image += ":latest"
	}
	return image
}

// getDerivedDBImageSeed returns the in-image path and byte size of a base_db
// seed baked into the built dbimage. seedPath is empty if there is none; size
// is -1 if it could not be determined. This runs a container against the image,
// so callers gate it on mayHaveDerivedDBImageSeed().
func (app *DdevApp) getDerivedDBImageSeed() (seedPath string, size int64) {
	var candidates []string
	for _, ext := range baseDBSeedExtensions {
		candidates = append(candidates, CustomBaseDBSeedPathPrefix+"."+ext)
	}
	script := fmt.Sprintf(`for f in %s; do if [ -f "$f" ]; then printf '%%s %%s\n' "$f" "$(wc -c <"$f")"; break; fi; done`, strings.Join(candidates, " "))

	_, out, err := dockerutil.RunSimpleContainer(
		derivedBuiltImageRef(app.GetDBImage(), app.Name),
		"db-seed-check-"+app.Name+"-"+util.RandString(6),
		[]string{"-c", script},
		[]string{"/bin/sh"},
		nil,   // env
		nil,   // binds
		"",    // uid
		true,  // rm
		false, // detach
		map[string]string{`com.ddev.site-name`: ""},
		nil, // portBindings
		nil, // healthcheck
	)
	if err != nil {
		util.Debug("Unable to check dbimage for a baked-in base_db seed: %v, output=%s", err, out)
		return "", -1
	}
	// The container's output can carry unrelated noise, so only accept a line
	// whose first field actually names one of the seed paths we asked about.
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !slices.Contains(candidates, fields[0]) {
			continue
		}
		size = -1
		if len(fields) > 1 {
			if n, convErr := strconv.ParseInt(fields[1], 10, 64); convErr == nil {
				size = n
			}
		}
		return fields[0], size
	}
	return "", -1
}

// prepareSeedSnapshot makes a seed snapshot reachable by the db container.
// With bind mounts it's already visible, either in .ddev/db_snapshots or
// through the mount RenderComposeYAML adds for SeedSnapshotMountDir. With
// no_bind_mounts the db container sees /mnt/snapshots as a Docker volume
// instead, so wherever the snapshot came from it has to be copied in.
// Must run after the snapshots volume is created, so it gets the usual labels.
func (app *DdevApp) prepareSeedSnapshot(seed *SeedSnapshot) error {
	if !globalconfig.DdevGlobalConfig.NoBindMounts {
		return nil
	}
	uid, _, _ := dockerutil.GetContainerUser()
	if err := dockerutil.CopyIntoVolume(seed.HostPath, "ddev-"+app.Name+"-snapshots", "", uid, "", false); err != nil {
		return fmt.Errorf("failed to copy %s into the snapshots volume: %v", seed.HostPath, err)
	}
	return nil
}

// SeedDescription returns a human-readable description of what a brand-new
// database volume will be seeded from, or an empty string if it'll just be the
// stock starter database. Callers use this to decide whether to warn about a
// long first-boot restore and extend the container-ready timeout accordingly.
func (app *DdevApp) SeedDescription(seed *SeedSnapshot) string {
	if seed != nil {
		description := fmt.Sprintf("the '%s' snapshot", SeedSnapshotName)
		if app.SeedSnapshot != "" {
			description = fmt.Sprintf("snapshot '%s'", app.SeedSnapshot)
		}
		return fmt.Sprintf("%s\n%s%s", description, fileutil.ShortHomeJoin(seed.HostPath), fileSizeSuffix(seed.HostPath))
	}
	if app.UsesBaseDBSeed() && app.mayHaveDerivedDBImageSeed() {
		if baseDBSeed, size := app.getDerivedDBImageSeed(); baseDBSeed != "" {
			return fmt.Sprintf("%s%s baked into dbimage %s", baseDBSeed, byteSizeSuffix(size), app.GetDBImage())
		}
	}
	return ""
}

// announceBaseDBSeed tells the user what a brand-new database volume is about to
// be seeded from, in the same spirit as RestoreSnapshot: what it's doing, which
// seed it's using, and that it may take a while. maxWaitTime is the
// container-ready timeout that will actually be in effect, so the message
// matches reality.
func (app *DdevApp) announceBaseDBSeed(description string, maxWaitTime int) {
	util.Success("Initializing new database volume from %s...", description)
	util.Success("With a large database this may take a long time.\nThis may time out after %d seconds \nbut you can increase it by changing default_container_timeout.", maxWaitTime)
	output.UserOut.Printf("You can follow the progress in another terminal window with `ddev logs -s db -f %s`", app.Name)
}

// fileSizeSuffix returns a " (2.3GB)" style suffix for a file, or an empty
// string if the size can't be determined.
func fileSizeSuffix(filePath string) string {
	fi, err := os.Stat(filePath)
	if err != nil || fi.IsDir() {
		return ""
	}
	return byteSizeSuffix(fi.Size())
}

// byteSizeSuffix returns a " (2.3GB)" style suffix for a byte count, or an
// empty string if size is negative (unknown).
func byteSizeSuffix(size int64) string {
	if size < 0 {
		return ""
	}
	return fmt.Sprintf(" (%s)", util.FormatBytes(size))
}
