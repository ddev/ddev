package ddevapp

import (
	"github.com/ddev/ddev/pkg/util"
	"syscall"
)

// failOnRosetta checks to see if we're running under Rosetta and fails if we are.
func failOnRosetta() {
	r, err := syscall.Sysctl("sysctl.proc_translated")
	if err == nil {
		// from https://www.yellowduck.be/posts/detecting-apple-silicon-via-go
		if r == "\x01\x00\x00" {
			util.Failed("You seem to be running under Rosetta, please install the ARM64 version of DDEV. If you're using homebrew, you need to reinstall it properly, see https://brew.sh")
		}
	}
}
