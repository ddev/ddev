package ddevapp

import (
	"github.com/drud/ddev/pkg/fileutil"
	"os/exec"
	"path/filepath"
	"strings"
)

// getCAROOT() verifies that the mkcert command is available and its root keys readable.
// 1. Find out CAROOT
// 2. Look there to see if key/crt are readable
// 3. If not, see if mkcert is even available, return informative message if not
var caROOT = ""

func getCAROOT() string {
	if caROOT != "" {
		return caROOT
	}
	_, err := exec.LookPath("mkcert")
	if err != nil {
		return ""
	}

	out, err := exec.Command("mkcert", "-CAROOT").Output()
	if err != nil {
		return ""
	}
	root := strings.Trim(string(out), "\n")
	if !fileutil.FileIsReadable(filepath.Join(root, "rootCA-key.pem")) || !fileutil.FileExists(filepath.Join(root, "rootCA.pem")) {
		return ""
	}
	caROOT = root
	return caROOT
}
