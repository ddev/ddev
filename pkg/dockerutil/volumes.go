package dockerutil

import (
	"fmt"
	"os"
	"strings"

	ddevImages "github.com/ddev/ddev/pkg/docker"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

// RemoveVolume removes named volume. Does not throw error if the volume did not exist.
func RemoveVolume(volumeName string) error {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return err
	}
	if _, err := apiClient.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{}); err == nil {
		_, err := apiClient.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: true})
		if err != nil {
			if err.Error() == "volume in use and cannot be removed" {
				containers, err := apiClient.ContainerList(ctx, client.ContainerListOptions{
					All:     true,
					Filters: client.Filters{}.Add("volume", volumeName),
				})
				// Get names of containers which are still using the volume.
				var containerNames []string
				if err == nil {
					for _, c := range containers.Items {
						containerNames = append(containerNames, ContainerName(&c))
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
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false
	}
	_, err = apiClient.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{})
	if err != nil {
		return false
	}
	return true
}

// VolumeLabels returns map of labels found on volume.
func VolumeLabels(volumeName string) (map[string]string, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	v, err := apiClient.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{})
	if err != nil {
		return nil, err
	}
	return v.Volume.Labels, nil
}

// CreateVolume creates a Docker volume
func CreateVolume(volumeName string, driver string, driverOpts map[string]string, labels map[string]string) (volume.Volume, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return volume.Volume{}, err
	}
	vol, err := apiClient.VolumeCreate(ctx, client.VolumeCreateOptions{Name: volumeName, Labels: labels, Driver: driver, DriverOpts: driverOpts})
	if err != nil {
		return volume.Volume{}, err
	}

	return vol.Volume, err
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

	labels := map[string]string{"com.ddev.site-name": ""}
	if IsPodmanRootless() {
		labels["com.ddev.userns"] = "keep-id"
	}
	containerID, _, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, labels, nil, nil)
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

// VolumeSize holds information about a Docker volume's size
type VolumeSize struct {
	Name      string
	SizeBytes int64
	SizeHuman string
}

// ParseDockerSystemDf retrieves volume sizes using the Docker API
// Returns map of volume names to their sizes
func ParseDockerSystemDf() (map[string]VolumeSize, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker client: %v", err)
	}

	// Use Docker API to get disk usage with verbose volume information
	diskUsage, err := apiClient.DiskUsage(ctx, client.DiskUsageOptions{
		Volumes: true,
		Verbose: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage from Docker API: %v", err)
	}

	volumeSizes := make(map[string]VolumeSize)

	// Extract volume sizes from the disk usage result
	for _, vol := range diskUsage.Volumes.Items {
		var sizeBytes int64
		// UsageData is only available for local volumes
		if vol.UsageData != nil && vol.UsageData.Size >= 0 {
			sizeBytes = vol.UsageData.Size
		}

		sizeHuman := util.FormatBytes(sizeBytes)

		volumeSizes[vol.Name] = VolumeSize{
			Name:      vol.Name,
			SizeBytes: sizeBytes,
			SizeHuman: sizeHuman,
		}
	}

	return volumeSizes, nil
}

// GetVolumeSize returns the size of a specific Docker volume
func GetVolumeSize(volumeName string) (int64, string, error) {
	volumeSizes, err := ParseDockerSystemDf()
	if err != nil {
		return 0, "", err
	}

	if volSize, exists := volumeSizes[volumeName]; exists {
		return volSize.SizeBytes, volSize.SizeHuman, nil
	}

	// Volume not found in df output, might not exist or have no size
	return 0, "0B", nil
}

// PurgeDirectoryContentsInVolume removes all files inside directories within a Docker volume
// while keeping the directories themselves intact. This is important for inotify watchers that
// monitor the directory - if the directory is deleted and recreated, the watch breaks.
// volumeName is the volume to operate on
// subdirs are the paths within the volume to purge (e.g., "traefik/config", "traefik/certs")
func PurgeDirectoryContentsInVolume(volumeName string, subdirs []string, uid string) error {
	volPath := "/mnt/v"

	containerName := "PurgeInVolume_" + nodeps.RandomString(12)

	track := util.TimeTrackC("PurgeDirectoryContentsInVolume " + volumeName)

	// Build mkdir commands and rm glob patterns
	var mkdirs []string
	var rmPaths []string
	for _, subdir := range subdirs {
		fullPath := volPath + "/" + subdir
		mkdirs = append(mkdirs, fmt.Sprintf(`"%s"`, fullPath))
		rmPaths = append(rmPaths, fmt.Sprintf(`"%s"/*`, fullPath))
	}
	c := fmt.Sprintf("mkdir -p %s && rm -rf %s", strings.Join(mkdirs, " "), strings.Join(rmPaths, " "))

	labels := map[string]string{"com.ddev.site-name": ""}
	if IsPodmanRootless() {
		labels["com.ddev.userns"] = "keep-id"
	}
	containerID, _, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, labels, nil, nil)
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer RemoveContainer(containerID)

	track()
	return nil
}

// ListFilesInVolume returns a list of filenames in a volume subdirectory.
// volumeName is the volume to list from
// subdir is the path within the volume (e.g., "traefik/config")
// Returns a slice of filenames (not full paths)
func ListFilesInVolume(volumeName string, subdir string) ([]string, error) {
	volPath := "/mnt/v"
	fullPath := volPath + "/" + subdir

	containerName := "ListInVolume_" + nodeps.RandomString(12)

	track := util.TimeTrackC("ListFilesInVolume " + volumeName + "/" + subdir)
	defer track()

	// List files, suppress errors if directory doesn't exist
	c := fmt.Sprintf(`ls -1 "%s" 2>/dev/null || true`, fullPath)

	labels := map[string]string{"com.ddev.site-name": ""}
	if IsPodmanRootless() {
		labels["com.ddev.userns"] = "keep-id"
	}
	containerID, stdout, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, labels, nil, nil)
	if err != nil {
		return nil, err
	}
	// nolint: errcheck
	defer RemoveContainer(containerID)

	// Parse the output into a slice of filenames
	var files []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// RemoveFilesFromVolume removes specific files from a volume subdirectory.
// volumeName is the volume to operate on
// subdir is the path within the volume (e.g., "traefik/config")
// files is a list of filenames to remove (not full paths)
func RemoveFilesFromVolume(volumeName string, subdir string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	volPath := "/mnt/v"
	fullPath := volPath + "/" + subdir

	containerName := "RemoveFromVolume_" + nodeps.RandomString(12)

	track := util.TimeTrackC("RemoveFilesFromVolume " + volumeName + "/" + subdir)
	defer track()

	// Build rm command for each file
	var rmPaths []string
	for _, f := range files {
		rmPaths = append(rmPaths, fmt.Sprintf(`"%s/%s"`, fullPath, f))
	}
	c := fmt.Sprintf("rm -f %s", strings.Join(rmPaths, " "))

	labels := map[string]string{"com.ddev.site-name": ""}
	if IsPodmanRootless() {
		labels["com.ddev.userns"] = "keep-id"
	}
	containerID, _, err := RunSimpleContainer(ddevImages.GetWebImage(), containerName, []string{"bash", "-c", c}, nil, nil, []string{volumeName + ":" + volPath}, "0", false, true, labels, nil, nil)
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer RemoveContainer(containerID)

	return nil
}
