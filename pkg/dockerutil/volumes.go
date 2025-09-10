package dockerutil

import (
	"fmt"
	"os"
	"strings"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

// RemoveVolume removes named volume. Does not throw error if the volume did not exist.
func RemoveVolume(volumeName string) error {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
	if _, err := client.VolumeInspect(ctx, volumeName); err == nil {
		err := client.VolumeRemove(ctx, volumeName, true)
		if err != nil {
			if err.Error() == "volume in use and cannot be removed" {
				containers, err := client.ContainerList(ctx, container.ListOptions{
					All:     true,
					Filters: filters.NewArgs(filters.KeyValuePair{Key: "volume", Value: volumeName}),
				})
				// Get names of containers which are still using the volume.
				var containerNames []string
				if err == nil {
					for _, c := range containers {
						// Skip first character, it's a slash.
						containerNames = append(containerNames, c.Names[0][1:])
					}
					var containerNamesString = strings.Join(containerNames, " ")
					return fmt.Errorf("docker volume '%s' is in use by one or more containers and cannot be removed. Use 'docker rm -f %s' to remove them", volumeName, containerNamesString)
				}
				return fmt.Errorf("docker volume '%s' is in use by a container and cannot be removed. Use 'docker rm -f $(docker ps -aq)' to remove all containers", volumeName)
			}
			return err
		}
	}
	return nil
}

// VolumeExists checks to see if the named volume exists.
func VolumeExists(volumeName string) bool {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return false
	}
	_, err = client.VolumeInspect(ctx, volumeName)
	if err != nil {
		return false
	}
	return true
}

// VolumeLabels returns map of labels found on volume.
func VolumeLabels(volumeName string) (map[string]string, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	v, err := client.VolumeInspect(ctx, volumeName)
	if err != nil {
		return nil, err
	}
	return v.Labels, nil
}

// CreateVolume creates a Docker volume
func CreateVolume(volumeName string, driver string, driverOpts map[string]string, labels map[string]string) (volume.Volume, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return volume.Volume{}, err
	}
	vol, err := client.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName, Labels: labels, Driver: driver, DriverOpts: driverOpts})

	return vol, err
}

// CopyIntoVolume copies a file or directory on the host into a Docker volume
// sourcePath is the host-side full path
// volumeName is the volume name to copy to
// targetSubdir is where to copy it to on the volume
// uid is the uid of the resulting files
// exclusion is a path to be excluded
// If destroyExisting the specified targetSubdir is removed and recreated
func CopyIntoVolume(sourcePath string, volumeName string, targetSubdir string, uid string, exclusion string, destroyExisting bool) error {
	volPath := "/mnt/v"
	targetSubdirFullPath := volPath + "/" + targetSubdir
	_, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	f, err := os.Open(sourcePath)
	if err != nil {
		util.Failed("Failed to open %s: %v", sourcePath, err)
	}

	// nolint errcheck
	defer f.Close()

	containerName := "CopyIntoVolume_" + nodeps.RandomString(12)

	track := util.TimeTrackC("CopyIntoVolume " + sourcePath + " " + volumeName)

	var c = ""
	if destroyExisting {
		c = c + `rm -rf "` + targetSubdirFullPath + `"/{*,.*} && `
	}
	c = c + "mkdir -p " + targetSubdirFullPath + " && sleep infinity "

	containerID, _, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, map[string]string{"com.ddev.site-name": ""}, nil, nil)
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer RemoveContainer(containerID)

	err = CopyIntoContainer(sourcePath, containerName, targetSubdirFullPath, exclusion)

	if err != nil {
		return err
	}

	// chown/chmod the uploaded content
	command := fmt.Sprintf("chown -R %s %s", uid, targetSubdirFullPath)
	stdout, stderr, err := Exec(containerID, command, "0")
	util.Debug("Exec %s stdout=%s, stderr=%s, err=%v", command, stdout, stderr, err)

	if err != nil {
		return err
	}
	track()
	return nil
}
