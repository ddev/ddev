package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
)

// NormalizePath prefixes secret paths with secret when necessary
func NormalizePath(sPath string) (newPath string) {
	newPath = sPath
	if !strings.HasPrefix(sPath, "secret/") || !strings.HasPrefix(sPath, "cubbyhole/") {
		if strings.HasPrefix(sPath, "/") {
			newPath = filepath.Join("secret", sPath[1:])
		} else {
			newPath = filepath.Join("secret", sPath)
		}
	}
	return
}

func askForConfirmation() bool {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "yes"}
	nokayResponses := []string{"n", "no"}
	responseLower := strings.ToLower(response)

	if containsString(okayResponses, responseLower) {
		return true
	} else if containsString(nokayResponses, responseLower) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation()
	}
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

// containsString returns true if slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}


// Failed will print an red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	color.Red(format, a...)
	os.Exit(1)
}

// Success will indicate an operation succeeded with colored confirmation text.
func Success(format string, a ...interface{}) {
	color.Cyan(format, a...)
}

// Warning will present the user with warning text.
func Warning(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// NetExists checks to see if the docker network for DRUD local development exists.
func NetExists(client *docker.Client, name string) bool {
	nets, _ := client.ListNetworks()
	for _, n := range nets {
		if n.Name == name {
			return true
		}
	}
	return false
}

// EnsureNetwork will ensure the docker network for DRUD local development is created.
func EnsureNetwork(client *docker.Client, name string) error {
	if !NetExists(client, name) {
		netOptions := docker.CreateNetworkOptions{
			Name:     name,
			Driver:   "bridge",
			Internal: false,
		}
		_, err := client.CreateNetwork(netOptions)
		if err != nil {
			return err
		}
		log.Println("Network", name, "created")

	}
	return nil
}
