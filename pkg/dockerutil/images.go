package dockerutil

import (
	"context"
	"fmt"
	"time"

	"github.com/containerd/platforms"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
	"github.com/moby/moby/client/pkg/versions"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageLabel returns the value of a label on a local image. It returns an
// empty string (and no error) when the image has no such label, and an error
// only when the image can't be inspected at all, for example because it hasn't
// been pulled yet.
func ImageLabel(imageName string, label string) (string, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return "", err
	}
	inspect, err := apiClient.ImageInspect(ctx, imageName)
	if err != nil {
		return "", err
	}
	if inspect.Config == nil {
		return "", nil
	}
	return inspect.Config.Labels[label], nil
}

// ImageExistsLocally determines if an image is available locally.
func ImageExistsLocally(imageName string) (bool, error) {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false, err
	}

	// If inspect succeeds, we have an image.
	_, err = apiClient.ImageInspect(ctx, imageName)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// IsRegistryReachable reports whether the Docker daemon can reach a registry,
// by pulling the local utilities image by digest, which downloads nothing.
// The daemon runs the pull itself, so it takes the path a build takes, proxy
// and registry mirrors included. Any error means unreachable, because the
// containerd image store reports an unreachable registry as not found.
// An image with no digest, such as one from docker load, falls back to the
// DNS check in globalconfig.IsInternetActive.
func IsRegistryReachable() bool {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	inspect, err := apiClient.ImageInspect(ctx, versionconstants.UtilitiesImage)
	if err != nil || len(inspect.RepoDigests) == 0 {
		util.Debug("Unable to find a digest for %s, checking DNS instead: %v", versionconstants.UtilitiesImage, err)
		return globalconfig.IsInternetActive()
	}
	resp, err := apiClient.ImagePull(ctx, inspect.RepoDigests[0], client.ImagePullOptions{})
	if err == nil {
		err = resp.Wait(ctx)
	}
	if err != nil {
		util.Debug("Unable to pull %s: %v", inspect.RepoDigests[0], err)
		return false
	}
	return true
}

// ImageExistsLocallyForPlatform is ImageExistsLocally that, when platform is
// set, also requires the local image to provide it, so a tag pulled earlier
// for another architecture counts as missing.
func ImageExistsLocallyForPlatform(imageName string, platform string) (bool, error) {
	if platform == "" {
		return ImageExistsLocally(imageName)
	}
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false, err
	}
	// An unparsable platform counts as missing, so the pull reports it.
	want, err := platforms.Parse(platform)
	if err != nil {
		return false, nil
	}
	// A containerd store tag can hold several platforms, and inspect reports
	// the host's, or empty fields when that one is missing, so ask for ours.
	var opts []client.ImageInspectOption
	if versions.GreaterThanOrEqualTo(apiClient.ClientVersion(), "1.49") {
		opts = append(opts, client.ImageInspectWithPlatform(&want))
	}
	inspect, err := apiClient.ImageInspect(ctx, imageName, opts...)
	return err == nil && platforms.NewMatcher(want).Match(ocispec.Platform{
		OS:           inspect.Os,
		Architecture: inspect.Architecture,
		Variant:      inspect.Variant,
	}), nil
}

// BuildPlatformToPull returns the platform to pull for an image built for
// buildPlatforms: none when they include the daemon's own, which the pull
// defaults to, else the first, since the registry may publish only those.
func BuildPlatformToPull(buildPlatforms []string) string {
	if len(buildPlatforms) == 0 {
		return ""
	}
	if serverVersion, err := GetServerVersion(); err == nil {
		daemon := platforms.NewMatcher(ocispec.Platform{OS: serverVersion.Os, Architecture: serverVersion.Arch})
		for _, platform := range buildPlatforms {
			if parsed, err := platforms.Parse(platform); err == nil && daemon.Match(parsed) {
				return ""
			}
		}
	}
	return buildPlatforms[0]
}

// FindImagesByLabels takes a map of label names and values and returns any Docker images which match all labels.
// danglingOnly is used to return only dangling images, otherwise return all of them, including dangling.
func FindImagesByLabels(labels map[string]string, danglingOnly bool) ([]image.Summary, error) {
	if len(labels) < 1 {
		return nil, fmt.Errorf("the provided list of labels was empty")
	}
	filterList := client.Filters{}
	for k, v := range labels {
		label := fmt.Sprintf("%s=%s", k, v)
		// If no value is specified, filter any value by the key.
		if v == "" {
			label = k
		}
		filterList.Add("label", label)
	}

	if danglingOnly {
		filterList.Add("dangling", "true")
	}

	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	images, err := apiClient.ImageList(ctx, client.ImageListOptions{
		All:     true,
		Filters: filterList,
	})
	if err != nil {
		return nil, err
	}
	return images.Items, nil
}

// RemoveImage removes an image with force
func RemoveImage(tag string) error {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return err
	}
	_, err = apiClient.ImageInspect(ctx, tag)
	if err == nil {
		_, err = apiClient.ImageRemove(ctx, tag, client.ImageRemoveOptions{Force: true})

		if err == nil {
			util.Debug("Deleted Docker image %s", tag)
		} else {
			util.Warning("Unable to delete %s: %v", tag, err)
		}
	}
	return nil
}
