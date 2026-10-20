package ddevapp

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
)

// GlobalCommandsContainerDir is where each container holds its copy of the
// global commands, one subdirectory per service.
const GlobalCommandsContainerDir = "/usr/local/lib/ddev/global-commands"

// legacyGlobalCommandsHashFile is where DDEV versions that copied global
// commands into the ddev-global-cache volume kept their cache key.
const legacyGlobalCommandsHashFile = ".global-commands-hash"

// PopulateGlobalCustomCommandFiles copies the global container commands into
// the running containers of app's services that have a global commands
// directory. The copy lives in each container's own filesystem, so containers
// recreated by `ddev start` always hold the commands of the global config
// directory they were started with. It returns the services it copied into.
func PopulateGlobalCustomCommandFiles(app *DdevApp) ([]string, error) {
	defer util.TimeTrackC("PopulateGlobalCustomCommandFiles")()
	sourceDir := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	archive, services, err := globalCommandsArchive(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("unable to archive %s: %v", sourceDir, err)
	}

	var copied []string
	var errs []error
	for _, service := range services {
		containerName := GetContainerName(app, service)
		err = dockerutil.CopyArchiveToContainer(containerName, "/", bytes.NewReader(archive))
		if cerrdefs.IsNotFound(err) {
			util.Debug("PopulateGlobalCustomCommandFiles: skipping %s, no such container", containerName)
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to copy global commands into %s: %v", service, err))
			continue
		}
		util.Debug("PopulateGlobalCustomCommandFiles: copied %s to %s:%s", sourceDir, containerName, GlobalCommandsContainerDir)
		copied = append(copied, service)
	}
	return copied, errors.Join(errs...)
}

// globalCommandsArchive returns a tar of sourceDir without host commands, rooted
// at GlobalCommandsContainerDir, and the services that have a directory in it.
// It stores the target of a symlinked file, because a link to a host path would
// dangle in the container, and makes every file executable, because files from
// a Windows host carry no exec bit.
func globalCommandsArchive(sourceDir string) ([]byte, []string, error) {
	// WalkDir would not descend into a symlinked root. EvalSymlinks can fail on
	// Windows for paths that work, such as subst drives, so keep the path then.
	resolved, err := filepath.EvalSymlinks(sourceDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err == nil {
		sourceDir = resolved
	}

	var buf bytes.Buffer
	var services, files []string
	tw := tar.NewWriter(&buf)
	root := strings.TrimPrefix(GlobalCommandsContainerDir, "/")
	err = filepath.WalkDir(sourceDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		name := path.Join(root, rel)
		if d.IsDir() {
			if rel == "host" {
				return fs.SkipDir
			}
			if rel != "." && !strings.Contains(rel, "/") {
				services = append(services, rel)
			}
			return tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: name + "/", Mode: 0755})
		}
		content, err := os.ReadFile(p)
		if err != nil {
			// A dangling symlink, or a symlink or Windows junction to a
			// directory, is left out.
			if !d.Type().IsRegular() {
				return nil
			}
			return err
		}
		if err = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0755, Size: int64(len(content))}); err != nil {
			return err
		}
		files = append(files, rel)
		_, err = tw.Write(content)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	util.Debug("globalCommandsArchive: %d files from %s for services %v:\n%s", len(files), sourceDir, services, strings.Join(files, "\n"))
	if err = tw.Close(); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), services, nil
}
