package ddevapp

import (
	"fmt"
	"os"
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

// InitializerSnapshotName is the reserved snapshot name that ddev-dbserver uses
// to seed a brand-new database volume instead of the stock starter database.
// See containers/ddev-dbserver/files/docker-entrypoint.sh
const InitializerSnapshotName = "initializer"

// CustomBaseDBSeedPathPrefix is where a derived dbimage bakes in its own
// base_db seed; ddev-dbserver checks it ahead of the stock /mysqlbase/base_db.*
const CustomBaseDBSeedPathPrefix = "/mysqlbase/custom/base_db"

// baseDBSeedExtensions lists the base_db seed compression suffixes in the same
// preference order that ddev-dbserver's docker-entrypoint.sh uses.
var baseDBSeedExtensions = []string{"zst", "gz"}

// UsesBaseDBSeed reports whether this project's database volume gets initialized
// from a base_db seed. Only MariaDB/MySQL do; PostgreSQL uses the upstream
// postgres entrypoint.
func (app *DdevApp) UsesBaseDBSeed() bool {
	return !app.IsDBOmitted() && slices.Contains([]string{nodeps.MariaDB, nodeps.MySQL}, app.Database.Type)
}

// GetInitializerSnapshotFile returns the host path of the project's reserved
// `initializer` snapshot for the configured database type/version, or an empty
// string if there isn't one. ddev-dbserver looks for
// /mnt/snapshots/initializer-<type>_<version>.{zst,gz}, preferring .zst.
func (app *DdevApp) GetInitializerSnapshotFile() string {
	if !app.UsesBaseDBSeed() {
		return ""
	}
	for _, ext := range baseDBSeedExtensions {
		f := app.GetConfigPath(filepath.Join("db_snapshots", fmt.Sprintf("%s-%s_%s.%s", InitializerSnapshotName, app.Database.Type, app.Database.Version, ext)))
		if fileutil.FileExists(f) {
			return f
		}
	}
	return ""
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

// getDerivedDBImageSeed returns the in-image path and byte size of a base_db
// seed baked into the built dbimage. path is empty if there is none; size is
// -1 if it could not be determined. This runs a container against the image,
// so callers gate it on mayHaveDerivedDBImageSeed().
func (app *DdevApp) getDerivedDBImageSeed() (path string, size int64) {
	var candidates []string
	for _, ext := range baseDBSeedExtensions {
		candidates = append(candidates, CustomBaseDBSeedPathPrefix+"."+ext)
	}
	script := fmt.Sprintf(`for f in %s; do if [ -f "$f" ]; then printf '%%s %%s\n' "$f" "$(wc -c <"$f")"; break; fi; done`, strings.Join(candidates, " "))

	_, out, err := dockerutil.RunSimpleContainer(
		app.GetDBImage()+"-"+app.Name+"-built",
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

// BaseDBSeedDescription returns a human-readable description of what a
// brand-new database volume will be seeded from (an initializer snapshot, or
// a seed baked into a derived dbimage), or an empty string if it'll just be
// the stock starter database. Callers use this to decide whether to warn
// about a long first-boot restore and extend the container-ready timeout
// accordingly.
func (app *DdevApp) BaseDBSeedDescription() string {
	if !app.UsesBaseDBSeed() {
		return ""
	}

	if initializer := app.GetInitializerSnapshotFile(); initializer != "" {
		return fmt.Sprintf("the '%s' snapshot %s%s", InitializerSnapshotName, filepath.ToSlash(initializer), fileSizeSuffix(initializer))
	}
	if app.mayHaveDerivedDBImageSeed() {
		if seed, size := app.getDerivedDBImageSeed(); seed != "" {
			return fmt.Sprintf("%s%s baked into dbimage %s", seed, byteSizeSuffix(size), app.GetDBImage())
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
func fileSizeSuffix(path string) string {
	fi, err := os.Stat(path)
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
