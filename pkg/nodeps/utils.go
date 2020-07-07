package nodeps

import (
	"math/rand"
	"os"
	"runtime"
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

// From https://www.calhoun.io/creating-random-strings-in-go/
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// RandomString creates a random string with a set length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GetWSLDistro returns the WSL2 distro name if on Linux
func GetWSLDistro() string {
	wslDistro := ""
	if runtime.GOOS == "linux" {
		wslDistro = os.Getenv("WSL_DISTRO_NAME")
	}
	return wslDistro
}
