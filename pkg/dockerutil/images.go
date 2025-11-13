package dockerutil

import (
	"fmt"

	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

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
