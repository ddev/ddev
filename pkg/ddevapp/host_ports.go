package ddevapp

import (
	"fmt"
	"net/netip"
	"strconv"
	"sync"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/ddev/ddev/pkg/dockerutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/netutil"
	"github.com/ddev/ddev/pkg/util"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
)

// MinHostPort and MaxHostPort bound the host ports DDEV chooses for published
// container ports that leave the host side empty, such as "127.0.0.1::3306".
// Left empty, the engine picks from the kernel's ephemeral range, which
// outgoing connections share, and rootless Podman doesn't bind until container
// start, so the port can be taken in between. This range is below the
// ephemeral range on Linux (32768+), macOS and Windows (49152+).
const (
	MinHostPort = 20000
	MaxHostPort = 29999
)

var (
	hostPortsMu sync.Mutex
	// hostPortsAssigned holds every port this process handed out, so projects
	// started together don't get the same port before their containers exist.
	hostPortsAssigned = make(map[string]bool)
	// hostPortChoices remembers the port chosen for each project service port,
	// so rendering the compose file again in this process keeps the same port.
	hostPortChoices = make(map[string]string)
)

// hostPortAllocator fills in host ports for one project's compose render.
type hostPortAllocator struct {
	app *DdevApp
	// localPorts is false when the engine publishes where a local bind test
	// can't see; the engine then picks the host port.
	localPorts bool
	// inUse holds host ports bound by existing DDEV containers, stopped ones
	// included, since a stopped container binds its port again when started.
	inUse map[string]bool
}

func newHostPortAllocator(app *DdevApp) *hostPortAllocator {
	return &hostPortAllocator{
		app:        app,
		localPorts: dockerutil.CanCheckHostPortsLocally(),
	}
}

// assignHostPort returns the host port for a service port with an empty
// published port, or "" to leave the choice to the engine.
func (a *hostPortAllocator) assignHostPort(service string, p composeTypes.ServicePortConfig) string {
	if !a.localPorts || (p.Protocol != "" && p.Protocol != "tcp") {
		return ""
	}
	key := fmt.Sprintf("%s/%s/%s/%d", a.app.GetName(), service, p.HostIP, p.Target)

	// Reusing an existing container's port keeps compose's config hash stable,
	// so a running or stopped container isn't recreated just to change port.
	if port := a.existingContainerHostPort(service, p); port != "" {
		hostPortsMu.Lock()
		hostPortChoices[key] = port
		hostPortsMu.Unlock()
		return port
	}

	hostPortsMu.Lock()
	defer hostPortsMu.Unlock()
	if port, ok := hostPortChoices[key]; ok {
		if portNum, _ := strconv.Atoi(port); netutil.IsHostPortFree(p.HostIP, portNum) {
			return port
		}
	}

	if a.inUse == nil {
		a.inUse = ddevContainerHostPorts()
	}
	skip := func(port string) bool {
		return hostPortsAssigned[port] || a.inUse[port] || globalconfig.HostPostIsAllocated(port) != ""
	}
	port := netutil.AllocateHostPort(p.HostIP, MinHostPort, MaxHostPort, skip)
	if port == "" {
		// A long-running process (the test suite) can hand out the whole range;
		// ports it assigned earlier are protected by their containers by now.
		port = netutil.AllocateHostPort(p.HostIP, MinHostPort, MaxHostPort, func(port string) bool {
			return a.inUse[port] || globalconfig.HostPostIsAllocated(port) != ""
		})
	}
	if port == "" {
		util.Debug("No host port available for %s port %d in %d-%d, leaving it to the engine", service, p.Target, MinHostPort, MaxHostPort)
		return ""
	}
	hostPortsAssigned[port] = true
	hostPortChoices[key] = port
	return port
}

// existingContainerHostPort returns the host port that this project's existing
// container for service binds for p, or "". A container that isn't running
// doesn't hold its port, so its port is returned only if still free; otherwise
// it would fail to bind on every start.
func (a *hostPortAllocator) existingContainerHostPort(service string, p composeTypes.ServicePortConfig) string {
	c, err := a.app.FindContainerByType(service)
	if err != nil || c == nil {
		return ""
	}
	bindings, err := dockerutil.GetContainerPortBindings(c.ID)
	if err != nil {
		return ""
	}
	target, err := network.ParsePort(fmt.Sprintf("%d/tcp", p.Target))
	if err != nil {
		return ""
	}
	wantIP, _ := netip.ParseAddr(p.HostIP)
	for _, b := range bindings[target] {
		if b.HostPort == "" || (b.HostIP.IsValid() && b.HostIP != wantIP) {
			continue
		}
		if c.State != container.StateRunning {
			if portNum, _ := strconv.Atoi(b.HostPort); !netutil.IsHostPortFree(p.HostIP, portNum) {
				util.Debug("Host port %s of stopped %s container is taken, choosing another", b.HostPort, service)
				return ""
			}
		}
		return b.HostPort
	}
	return ""
}

// ddevContainerHostPorts returns the host ports bound by all DDEV containers.
func ddevContainerHostPorts() map[string]bool {
	inUse := map[string]bool{}
	containers, err := dockerutil.FindContainersWithLabel("com.ddev.site-name")
	if err != nil {
		util.Debug("Unable to list DDEV containers to check their host ports: %v", err)
		return inUse
	}
	for _, c := range containers {
		ports, err := dockerutil.GetBoundHostPorts(c.ID)
		if err != nil {
			continue
		}
		for _, port := range ports {
			inUse[port] = true
		}
	}
	return inUse
}
