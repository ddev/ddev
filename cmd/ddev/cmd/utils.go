package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/drud-go/utils/system"
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

// SetHomedir gets homedir and sets it to global homedir
func SetHomedir() {
	var err error
	homedir, err = system.GetHomeDir()
	if err != nil {
		log.Fatal(err)
	}
}

// PrepConf sets global cfgFile with abs path to conf file
// and then creates a default config file if one does not exist
func PrepConf() {
	if strings.HasPrefix(cfgFile, "$HOME") || strings.HasPrefix(cfgFile, "~") {
		cfgFile = strings.Replace(cfgFile, "$HOME", homedir, 1)
		cfgFile = strings.Replace(cfgFile, "~", homedir, 1)
	}

	if !strings.HasPrefix(cfgFile, "/") {
		absPath, absErr := filepath.Abs(cfgFile)
		if absErr != nil {
			log.Fatal(absErr)
		}
		cfgFile = absPath
	}

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		var cFile, err = os.Create(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		cFile.Close()
	}
}

// getMAC returns the mac address for interface en0 or the first in the list otherwise
func getMAC() (string, error) {
	var macADDR string
	ifs, _ := net.Interfaces()
	for _, v := range ifs {
		h := v.HardwareAddr.String()
		if len(h) == 0 {
			continue
		}
		if v.Name == "en0" {
			macADDR = h
		}
	}
	if macADDR == "" {
		macADDR = ifs[0].HardwareAddr.String()
	}
	if macADDR == "" {
		return macADDR, fmt.Errorf("no MAC Address found")
	}
	return macADDR, nil
}

// ParseConfigFlag is needed in order to get the value of the flag before cobra can
func ParseConfigFlag() string {
	value := cfgFile
	args := os.Args

	for i, arg := range args {
		if strings.HasPrefix(arg, "--config=") {
			value = strings.TrimPrefix(arg, "--config=")
		} else if arg == "--config" {
			if len(args) > i+1 {
				value = args[i+1]
			} else {
				log.Fatalln("--config requires a configuration file to be specified.")
			}
		}
	}

	return value
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
