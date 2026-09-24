package dockerutil

import (
	"context"
	"fmt"
	"time"

	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
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

// RegistrySearchTerm is the repository IsRegistryReachable searches for, a
// variable so tests can point it at a registry that can't be reached.
var RegistrySearchTerm = "docker.io/ddev/ddev-utilities"

// IsRegistryReachable reports whether the Docker daemon can reach Docker Hub.
// The daemon runs the search itself, so it takes the same network path, proxy
// included, as a pull or build. Search works on Podman, unlike
// DistributionInspect, and the term is fully qualified because Podman reads a
// bare org as a registry host.
func IsRegistryReachable() bool {
	ctx, apiClient, err := GetDockerClient()
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err = apiClient.ImageSearch(ctx, RegistrySearchTerm, client.ImageSearchOptions{Limit: 1})
	if err != nil {
		util.Debug("Unable to reach a registry searching for %s: %v", RegistrySearchTerm, err)
		return false
	}
	return true
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
