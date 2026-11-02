package ddevapp

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
)

// CAROOTContainerDir is CAROOT in the ddev-webserver image, where web holds
// its own copy of the host's mkcert root CA.
const CAROOTContainerDir = "/usr/local/lib/ddev/mkcert"

// hostCAROOT returns the host's mkcert CAROOT when it holds a root CA and key.
func hostCAROOT() string {
	caRoot := globalconfig.GetCAROOT()
	if caRoot == "" || !fileutil.FileIsReadable(filepath.Join(caRoot, "rootCA.pem")) || !fileutil.FileIsReadable(filepath.Join(caRoot, "rootCA-key.pem")) {
		return ""
	}
	return caRoot
}

// caRootArchive returns a tar of the root CA and key in caRoot, rooted at
// CAROOTContainerDir and owned by uid and gid.
func caRootArchive(caRoot string, uid int, gid int) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	root := strings.TrimPrefix(CAROOTContainerDir, "/")
	now := time.Now()
	if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: root + "/", Mode: 0755, Uid: uid, Gid: gid, ModTime: now}); err != nil {
		return nil, err
	}
	for _, f := range []struct {
		name string
		mode int64
	}{{"rootCA-key.pem", 0600}, {"rootCA.pem", 0644}} {
		content, err := os.ReadFile(filepath.Join(caRoot, f.name))
		if err != nil {
			return nil, err
		}
		if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: path.Join(root, f.name), Mode: f.mode, Size: int64(len(content)), Uid: uid, Gid: gid, ModTime: now}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(content); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// copyCAROOTIntoWeb copies the host's mkcert root CA into the web container,
// which then no longer needs to mount ddev-global-cache for it.
func (app *DdevApp) copyCAROOTIntoWeb() error {
	caRoot := hostCAROOT()
	if caRoot == "" {
		return nil
	}
	uidStr, gidStr, _ := dockerutil.GetContainerUser()
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil {
		return err
	}
	archive, err := caRootArchive(caRoot, uid, gid)
	if err != nil {
		return fmt.Errorf("unable to archive %s: %v", caRoot, err)
	}
	containerName := GetContainerName(app, "web")
	if err = dockerutil.CopyArchiveToContainer(containerName, "/", bytes.NewReader(archive)); err != nil {
		return fmt.Errorf("unable to copy the mkcert root CA into %s: %v", containerName, err)
	}
	util.Debug("Copied mkcert root CA from %s to %s:%s", caRoot, containerName, CAROOTContainerDir)
	return nil
}
