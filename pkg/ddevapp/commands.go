package ddevapp

import (
	"fmt"
	"os"
	"path/filepath"

	dockerImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// PopulateGlobalCustomCommandFiles sets up the custom command files in the project
// directories where they need to go.
func PopulateGlobalCustomCommandFiles() error {
	sourceGlobalCommandPath := filepath.Join(globalconfig.GetGlobalDdevDir(), "commands")
	err := os.MkdirAll(sourceGlobalCommandPath, 0755)
	if err != nil {
		return nil
	}

	// Remove contents of the directory, if the directory exists and has some contents
	commandDirInVolume := "/mnt/ddev-global-cache/global-commands/"
	_, _, err = performTaskInContainer([]string{"rm", "-rf", commandDirInVolume})
	if err != nil {
		return fmt.Errorf("unable to rm %s: %v", commandDirInVolume, err)
	}

	// Copy commands into container (this will create the directory if it's not there already)
	uid, _, _ := util.GetContainerUIDGid()
	err = dockerutil.CopyIntoVolume(sourceGlobalCommandPath, "ddev-global-cache", "global-commands", uid, "host", false)
	if err != nil {
		return err
	}

	// Make sure all commands can be executed
	_, stderr, err := performTaskInContainer([]string{"sh", "-c", "chmod -R u+rwx " + commandDirInVolume})
	if err != nil {
		return fmt.Errorf("unable to chmod %s: %v (stderr=%s)", commandDirInVolume, err, stderr)
	}

	return nil
}

// performTaskInContainer runs a command in the web container if it's available,
// but uses an anonymous container otherwise.
func performTaskInContainer(command []string) (string, string, error) {
	app, err := GetActiveApp("")
	if err == nil {
		status, _ := app.SiteStatus()
		if status == SiteRunning {
			// Prepare docker exec command
			opts := &ExecOpts{
				RawCmd:    command,
				Tty:       false,
				NoCapture: false,
			}
			return app.Exec(opts)
		}
	}

	// If there is no running active site, use an anonymous container instead.
	containerName := "performTaskInContainer" + nodeps.RandomString(12)
	uid, _, _ := util.GetContainerUIDGid()
	return dockerutil.RunSimpleContainer(dockerImages.GetWebImage(), containerName, command, nil, nil, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false, map[string]string{"com.ddev.site-name": ""}, nil, &dockerutil.NoHealthCheck)
}
