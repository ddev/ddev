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
	"slices"
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

// PopulateGlobalCustomCommandFiles replaces the global container commands in
// the containers of app's services that have a global commands directory.
// The copy lives in each container's own filesystem, so every container holds
// the commands of the global config directory it was last started or
// refreshed with. It returns the services it copied into.
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
		// Extracting only adds files, so remove the commands deleted since the
		// last copy. An image without a shell keeps them.
		_, stderr, err := dockerutil.Exec(containerName, "rm -rf "+GlobalCommandsContainerDir, "0")
		if cerrdefs.IsNotFound(err) {
			util.Debug("PopulateGlobalCustomCommandFiles: skipping %s, no such container", containerName)
			continue
		}
		if err != nil {
			util.Debug("PopulateGlobalCustomCommandFiles: unable to remove %s:%s: %v %s", containerName, GlobalCommandsContainerDir, err, stderr)
		}
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

// populateGlobalCommandsInBackground starts PopulateGlobalCustomCommandFiles,
// so the copy overlaps the wait for the containers to become ready, and
// returns a function that waits for it to finish.
func (app *DdevApp) populateGlobalCommandsInBackground() func() {
	done := make(chan error, 1)
	go func() {
		_, err := PopulateGlobalCustomCommandFiles(app)
		done <- err
	}()
	return func() {
		if err := <-done; err != nil {
			util.Warning("Failed to populate global custom command files: %v", err)
		}
	}
}

// globalCommandsArchive returns a tar of sourceDir without host commands and
// hidden directories, rooted at GlobalCommandsContainerDir, and the services
// that have a directory in it. It follows symlinks, because a link to a host
// path would dangle in the container, and makes every file executable, because
// files from a Windows host carry no exec bit.
func globalCommandsArchive(sourceDir string) ([]byte, []string, error) {
	rootInfo, err := os.Stat(sourceDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	var services, files []string
	tw := tar.NewWriter(&buf)
	root := strings.TrimPrefix(GlobalCommandsContainerDir, "/")

	var addDir func(dir string, rel string, ancestors []os.FileInfo) error
	addDir = func(dir string, rel string, ancestors []os.FileInfo) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			p := filepath.Join(dir, entry.Name())
			entryRel := path.Join(rel, entry.Name())
			info, err := os.Stat(p)
			if errors.Is(err, fs.ErrNotExist) {
				util.Verbose("globalCommandsArchive: skipping %s, a dangling symlink", p)
				continue
			}
			if err != nil {
				return err
			}
			name := path.Join(root, entryRel)
			if info.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") || entryRel == "host" {
					continue
				}
				// A symlink to an ancestor would otherwise recurse forever.
				if slices.ContainsFunc(ancestors, func(a os.FileInfo) bool { return os.SameFile(a, info) }) {
					util.Verbose("globalCommandsArchive: skipping %s, a symlink to a parent directory", p)
					continue
				}
				if rel == "" {
					services = append(services, entryRel)
				}
				if err = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: name + "/", Mode: 0755, ModTime: info.ModTime()}); err != nil {
					return err
				}
				if err = addDir(p, entryRel, append(slices.Clip(ancestors), info)); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if err = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0755, Size: int64(len(content)), ModTime: info.ModTime()}); err != nil {
				return err
			}
			if _, err = tw.Write(content); err != nil {
				return err
			}
			files = append(files, entryRel)
		}
		return nil
	}

	if err = tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: root + "/", Mode: 0755, ModTime: rootInfo.ModTime()}); err != nil {
		return nil, nil, err
	}
	if err = addDir(sourceDir, "", []os.FileInfo{rootInfo}); err != nil {
		return nil, nil, err
	}
	util.Verbose("globalCommandsArchive: %d files from %s for services %v: %s", len(files), sourceDir, services, strings.Join(files, " "))
	if err = tw.Close(); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), services, nil
}
