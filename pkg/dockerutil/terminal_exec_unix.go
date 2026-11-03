//go:build !windows

package dockerutil

import "os"

// killWithParent does nothing here, because ddev's interrupt handler kills
// the child on SIGINT, SIGTERM, and SIGHUP.
func killWithParent(*os.Process) (release func()) {
	return func() {}
}
