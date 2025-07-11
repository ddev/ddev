package main

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/ddevhosts"
	"github.com/ddev/ddev/pkg/output"
)

// addHostEntry adds an entry to default hosts file
func addHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		output.UserErr.Warn("DDEV_NONINTERACTIVE is set. Not adding the host entry.")
		return nil
	}
	hosts, err := getHostsFile()
	if err != nil {
		return err
	}
	err = hosts.Add(ip, name)
	if err != nil {
		return err
	}
	hosts.HostsPerLine(8)
	err = hosts.Flush()
	return err
}

// removeHostEntry removes named /etc/hosts entry if it exists
func removeHostEntry(name string, ip string) error {
	if os.Getenv("DDEV_NONINTERACTIVE") != "" {
		output.UserErr.Warn("DDEV_NONINTERACTIVE is set. Not removing the host entry.")
		return nil
	}
	hosts, err := getHostsFile()
	if err != nil {
		return err
	}
	err = hosts.Remove(ip, name)
	if err != nil {
		return err
	}
	err = hosts.Flush()
	return err
}

// isHostnameInHostsFile checks to see if the hostname already exists
func isHostnameInHostsFile(hostname string, dockerIP string) (bool, error) {
	hosts, err := getHostsFile()
	if err != nil {
		return false, err
	}
	return hosts.Has(dockerIP, hostname), nil
}

// getHostsFile returns the hosts file
// On WSL2 it normally assumes that the hosts file is in WSL2WindowsHostsFile
// Otherwise it lets goodhosts decide where the hosts file is.
func getHostsFile() (*ddevhosts.DdevHosts, error) {
	var hosts *ddevhosts.DdevHosts
	var err error
	if wslFlag {
		hosts, err = ddevhosts.NewCustomHosts(ddevhosts.WSL2WindowsHostsFile)
	} else {
		hosts, err = ddevhosts.New()
	}
	if err != nil {
		return nil, fmt.Errorf("unable to open hosts file: %v", err)
	}
	return hosts, nil
}
