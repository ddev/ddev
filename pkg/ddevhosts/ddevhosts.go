package ddevhosts

// This package is a simple wrapper on goodhosts.
// Its only current purpose is to provide GetIPPosition as an
// exported function.

import (
	"github.com/lextoumbourou/goodhosts"
)

// DdevHosts uses composition to absorb all exported functions of goodhosts
type DdevHosts struct {
	goodhosts.Hosts // provides all exported functions from goodhosts
}

// GetIPPosition is the same as the unexported getIpPosition,
// providing the position of the line in the hosts file that
// supports the IP address we're looking for.
// Or it returns -1 if none is found yet.
func (h DdevHosts) GetIPPosition(ip string) int {
	for i := range h.Lines {
		line := h.Lines[i]
		if !line.IsComment() && line.Raw != "" {
			if line.IP == ip {
				return i
			}
		}
	}

	return -1
}

// New is a simple wrapper on goodhosts.NewHosts()
func New() (*DdevHosts, error) {
	h, err := goodhosts.NewHosts()
	if err != nil {
		return nil, err
	}

	return &DdevHosts{h}, nil
}
