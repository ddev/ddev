package ddevapp

import (
	"fmt"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
)

// VolumeCloner abstracts volume duplication strategies.
type VolumeCloner interface {
	// CloneVolume copies all data from source Docker volume to target.
	// The target volume is created if it does not exist.
	CloneVolume(sourceVol, targetVol string) error
	// Name returns the strategy name for logging.
	Name() string
}

// TarCopyCloner clones volumes using an ephemeral container with tar pipe.
type TarCopyCloner struct{}

// CloneVolume copies all data from sourceVol to targetVol using an ephemeral
// container that mounts both volumes and pipes tar between them.
func (c *TarCopyCloner) CloneVolume(sourceVol, targetVol string) error {
	track := util.TimeTrackC(fmt.Sprintf("CloneVolume %s -> %s", sourceVol, targetVol))
	defer track()

	// Create the target volume if it doesn't exist
	if !dockerutil.VolumeExists(targetVol) {
		_, err := dockerutil.CreateVolume(targetVol, "local", nil, map[string]string{"com.ddev.site-name": ""})
		if err != nil {
			return fmt.Errorf("failed to create target volume %s: %v", targetVol, err)
		}
	}

	containerName := "ddev-clone-vol-" + nodeps.RandomString(6)

	labels := map[string]string{"com.ddev.site-name": ""}
	if dockerutil.IsPodmanRootless() {
		labels["com.ddev.userns"] = "keep-id"
	}

	cmd := []string{"sh", "-c", "cd /source && tar cf - . | (cd /target && tar xpf -)"}
	binds := []string{sourceVol + ":/source", targetVol + ":/target"}

	_, _, err := dockerutil.RunSimpleContainer(
		ddevImages.GetWebImage(),
		containerName,
		cmd,
		nil, // entrypoint
		nil, // env
		binds,
		"0",   // uid (root for full volume access)
		true,  // removeContainerAfterRun
		false, // detach
		labels,
		nil, // portBindings
		nil, // healthConfig
	)
	if err != nil {
		return fmt.Errorf("failed to clone volume %s to %s: %v", sourceVol, targetVol, err)
	}

	return nil
}

// Name returns the strategy name.
func (c *TarCopyCloner) Name() string {
	return "tar-copy"
}

// GetVolumeCloner returns the appropriate cloner based on configuration.
// If a strategy is configured in global config but not available, it logs a
// warning and falls back to the default TarCopyCloner.
func GetVolumeCloner() VolumeCloner {
	strategy := globalconfig.DdevGlobalConfig.VolumeCloneStrategy
	if strategy == "" || strategy == "tar-copy" {
		return &TarCopyCloner{}
	}

	// Future strategies would be matched here
	util.Warning("Volume clone strategy '%s' is not available, falling back to tar-copy", strategy)
	return &TarCopyCloner{}
}
