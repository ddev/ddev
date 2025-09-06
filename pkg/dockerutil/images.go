package dockerutil

import (
	"fmt"

	"github.com/ddev/ddev/pkg/util"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// ImageExistsLocally determines if an image is available locally.
func ImageExistsLocally(imageName string) (bool, error) {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return false, err
	}

	// If inspect succeeds, we have an image.
	_, err = client.ImageInspect(ctx, imageName)
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
	filterList := filters.NewArgs()
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

	ctx, client, err := GetDockerClient()
	if err != nil {
		return nil, err
	}
	images, err := client.ImageList(ctx, image.ListOptions{
		All:     true,
		Filters: filterList,
	})
	if err != nil {
		return nil, err
	}
	return images, nil
}

// RemoveImage removes an image with force
func RemoveImage(tag string) error {
	ctx, client, err := GetDockerClient()
	if err != nil {
		return err
	}
	_, err = client.ImageInspect(ctx, tag)
	if err == nil {
		_, err = client.ImageRemove(ctx, tag, image.RemoveOptions{Force: true})

		if err == nil {
			util.Debug("Deleted Docker image %s", tag)
		} else {
			util.Warning("Unable to delete %s: %v", tag, err)
		}
	}
	return nil
}
