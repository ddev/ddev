package ddevapp

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/hostname"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/util"
)

// AddHostsEntriesIfNeeded will run sudo ddev hostname to the site URL to the host's /etc/hosts.
// This should be run without admin privs; the DDEV hostname command will handle elevation.
func (app *DdevApp) AddHostsEntriesIfNeeded() error {
	var err error
	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	if os.Getenv("DDEV_NONINTERACTIVE") == "true" {
		util.Warning("Not trying to add hostnames because DDEV_NONINTERACTIVE=true")
		return nil
	}

	for _, name := range app.GetHostnames() {
		// If we're able to resolve the hostname via DNS or otherwise we
		// don't have to worry about this. This will allow resolution
		// of <whatever>.ddev.site for example
		if app.UseDNSWhenPossible {
			// If they have provided "*.<name>" then look up the suffix
			checkName := strings.TrimPrefix(name, "*.")
			hostIPs, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", checkName)

			// If we had successful lookup and the IP address looked up is local
			// then we don't have to add it to the /etc/hosts.
			if err == nil && len(hostIPs) > 0 && netutil.HasLocalIP(hostIPs) {
				continue
			}
		}

		// We likely won't hit the hosts.Has() as true because
		// we already did a lookup. But check anyway.
		exists, err := hostname.IsHostnameInHostsFile(name)
		if exists {
			continue
		}
		if err != nil {
			util.Warning("Unable to open hosts file: %v", err)
			continue
		}
		if !govalidator.IsDNSName(name) {
			util.Warning("DDEV cannot add unresolvable hostnames like `%s` to your hosts file.\nSee docs for more info, https://docs.ddev.com/en/stable/users/configuration/config/#additional_hostnames", name)
			continue
		}
		util.Warning("The hostname %s is not currently resolvable, trying to add it to the hosts file", name)
		out, err := hostname.ElevateToAddHostEntry(name, dockerIP)
		if err != nil {
			return fmt.Errorf("%s: %v", out, err)
		}
		if out != "" {
			util.Success(out)
		}
	}

	return nil
}

// RemoveHostsEntriesIfNeeded will remove the site URL from the host's /etc/hosts.
// This should be run without administrative privileges and will elevate
// where needed.
func (app *DdevApp) RemoveHostsEntriesIfNeeded() error {
	if os.Getenv("DDEV_NONINTERACTIVE") == "true" {
		util.Warning("Not trying to remove hostnames because DDEV_NONINTERACTIVE=true")
		return nil
	}

	dockerIP, err := dockerutil.GetDockerIP()
	if err != nil {
		return fmt.Errorf("could not get Docker IP: %v", err)
	}

	for _, name := range app.GetHostnames() {
		exists, err := hostname.IsHostnameInHostsFile(name)
		if !exists {
			continue
		}
		if err != nil {
			util.Warning("Unable to open hosts file: %v", err)
			continue
		}
		out, err := hostname.ElevateToRemoveHostEntry(name, dockerIP)
		if err != nil {
			util.Warning("%s: %v", out, err)
		} else if out != "" {
			util.Success(out)
		}
	}

	return nil
}
