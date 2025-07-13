package main

import (
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/ddevhosts"
)

// addHostEntry adds an entry to default hosts file
func addHostEntry(name string, ip string) error {
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
func isHostnameInHostsFile(hostname string, ip string) (bool, error) {
	hosts, err := getHostsFile()
	if err != nil {
		return false, err
	}
	return hosts.Has(ip, hostname), nil
}

// getHostsFile returns the hosts file
func getHostsFile() (*ddevhosts.DdevHosts, error) {
	hosts, err := ddevhosts.New()
	if err != nil {
		return nil, fmt.Errorf("unable to open hosts file: %v", err)
	}
	return hosts, nil
}

// printStdout writes formatted output to standard output
func printStdout(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format, a...)
}

// printStderr writes formatted output to standard error
func printStderr(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, a...)
}
