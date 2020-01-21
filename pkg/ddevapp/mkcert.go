package ddevapp

import (
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/util"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetCAROOT() verifies that the mkcert command is available and its CA keys readable.
// 1. Find out CAROOT
// 2. Look there to see if key/crt are readable
// 3. If not, see if mkcert is even available, return informative message if not
var caROOT = ""

func GetCAROOT() string {
	if caROOT != "" && validCAROOT(caROOT) {
		return caROOT
	}
	if globalconfig.DdevGlobalConfig.MkcertCARoot != "" {
		caROOT = globalconfig.DdevGlobalConfig.MkcertCARoot
		if !validCAROOT(caROOT) {
			globalconfig.DdevGlobalConfig.MkcertCARoot = ""
			caROOT = ""
			err := globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
			if err != nil {
				util.Warning("Error writing global config: %v", err)
			}
			return ""
		} else {
			return caROOT
		}
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
	globalconfig.DdevGlobalConfig.MkcertCARoot = root
	err = globalconfig.WriteGlobalConfig(globalconfig.DdevGlobalConfig)
	if err != nil {
		util.Warning("Error writing global config: %v", err)
	}

	return caROOT
}

func validCAROOT(path string) bool {
	return fileutil.FileExists(filepath.Join(path, "rootCA-key.pem"))
}
