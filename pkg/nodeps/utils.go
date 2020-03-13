package nodeps

import (
	"context"
	"net"
	"os"
	"time"
)

// ArrayContainsString returns true if slice contains element
func ArrayContainsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// IsDockerToolbox detects if the running docker is docker toolbox
// It shouldn't be run much as it requires actually running the executable.
// This lives here instead of in dockerutils to avoid unecessary import cycles.
// Inspired by https://stackoverflow.com/questions/43242218/how-can-a-script-distinguish-docker-toolbox-and-docker-for-windows
func IsDockerToolbox() bool {
	dockerToolboxPath := os.Getenv("DOCKER_TOOLBOX_INSTALL_PATH")
	if dockerToolboxPath != "" {
		return true
	}
	return false
}

//IsInternetActive() checks to see if we have a viable
// internet connection. It just tries a quick DNS query.
// This requires that the named record be query-able.
func IsInternetActive() bool {
	const timeout = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := net.LookupHost("i-exist.ddev.site")
	return err == nil && ctx.Err() == nil
}
