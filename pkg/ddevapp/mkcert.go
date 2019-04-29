package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"os/exec"
	"path/filepath"
	"strings"
)

// getCAROOT() verifies that the mkcert command is available and its root keys readable.
// 1. Find out CAROOT
// 2. Look there to see if key/crt are readable
// 3. If not, see if mkcert is even available, return informative message if not
func getCAROOT() (string, error) {
	_, err := exec.LookPath("mkcert")
	if err != nil {
		return "", fmt.Errorf("mkcert not found, TLS certs on localhost will not be trustable")
	}

	out, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		return "", fmt.Errorf("mkcert -CAROOT failed: %v", err)
	}
	caroot := strings.Trim(string(out), "\n")
	if !fileutil.FileIsReadable(filepath.Join(caroot, "rootCA-key.pem")) || !fileutil.FileExists(filepath.Join(caroot, "rootCA.pem")) {
		return caroot, fmt.Errorf("`mkcert -install` has not yet been run, please run it")
	}
	return caroot, nil
}
