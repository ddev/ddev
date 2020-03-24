package nodeps

import (
	"context"
	"fmt"
	"math/rand"
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
// This requires that the named record be query-able
func IsInternetActive() bool {
	const timeout = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	randomURL := RandomString(10) + ".ddev.site"
	addrs, err := net.DefaultResolver.LookupHost(ctx, randomURL)

	// Internet is active (active == true) if both err and ctx.Err() were nil
	active := err == nil && ctx.Err() == nil
	if os.Getenv("DDEV_DEBUG") != "" {
		fmt.Printf("IsInternetActive DEBUG: err=%v ctx.Err()=%v addrs=%v IsInternetactive==%v, randomURL=%v\n", err, ctx.Err(), addrs, active, randomURL)
	}
	if active == false {
		fmt.Println("Internet connection not detected")
	}
	return active
}

// From https://www.calhoun.io/creating-random-strings-in-go/
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
