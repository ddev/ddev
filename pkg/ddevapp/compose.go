package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/drud/ddev/pkg/version"
	"os"
	"runtime"
)

// DownloadDockerComposeIfNeeded downloads the proper version of docker-compose
// if it's either not yet installed or has the wrong version.
func DownloadDockerComposeIfNeeded(app *DdevApp) error {
	curVersion, err := version.GetLiveDockerComposeVersion()
	if err != nil || curVersion != version.RequiredDockerComposeVersion {
		err = DownloadDockerCompose()
		if err != nil {
			return err
		}
	}
	return nil
}

// DownloadDockerCompose gets the docker-compose binary and puts it into
// ~/.ddev/.bin
func DownloadDockerCompose() error {
	arch := runtime.GOARCH
	if arch == "arm64" {
		arch = "aarch64"
	}
	flavor := runtime.GOOS + "-" + arch
	globalBinDir := globalconfig.GetDDEVBinDir()
	destFile := globalconfig.GetDockerComposePath()
	composeURL := fmt.Sprintf("https://github.com/docker/compose/releases/download/%s/docker-compose-%s", version.RequiredDockerComposeVersion, flavor)
	output.UserOut.Printf("Downloading %s ...", composeURL)

	_ = os.Remove(globalconfig.GetDockerComposePath())

	_ = os.MkdirAll(globalBinDir, 0777)
	err := util.DownloadFile(destFile, composeURL, "true" != os.Getenv("DDEV_NONINTERACTIVE"))
	if err != nil {
		return err
	}
	output.UserOut.Printf("Download complete.")

	err = os.Chmod(destFile, 0755)
	if err != nil {
		return err
	}

	return nil
}
