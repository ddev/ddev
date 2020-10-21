package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// BashBinary defines the binary name of bash.
	BashBinary = "bash.exe"

	// gitBashPath defines the default path used.
	gitBashPath = "${ProgramFiles}/Git/bin/bash.exe"

	// system32Path defines the path to Windows' System32 folder where the
	// bash.exe provided by WSL is located and which does not work for our
	// purpose here because it runs in WSL and not on the host.
	system32Path = "${SystemRoot}/System32"
)

// bashPath caches the result to speedup the FindBashPath() function.
var bashPath string = ""

// FindBashPath returns the bash binary. On Unix like systems only the name is
// returned, on Windows this will be the full path and name to the found binary.
func FindBashPath() (string, error) {
	// Speedup the function by checking for a cached result first
	if bashPath == "" {
		// Try to find the binary in the default location first
		tempBashPath, err := exec.LookPath(os.ExpandEnv(filepath.FromSlash(gitBashPath)))
		if err != nil {
			// Remove System32 from path because we can not use bash.exe
			// provided by WSL
			path := os.Getenv("path")
			_ = os.Setenv("path", removeItem(path, os.ExpandEnv(system32Path)))

			// Fall back to the binary name only
			tempBashPath, err = exec.LookPath(BashBinary)

			// Reset path to original value
			_ = os.Setenv("path", path)

			if err != nil {
				return "", fmt.Errorf("%s could not be found. Make sure you have installed Git Bash and the installation folder is added to %%PATH%% properly", BashBinary)
			}
		}

		// Cache result
		bashPath = tempBashPath
	}

	return bashPath, nil
}

// removeItem removes the item from the path string
func removeItem(path string, item string) string {
	// Convert slahes and trim trailing back slash
	item = strings.TrimRight(filepath.FromSlash(item), string(filepath.Separator))

	newPath := ""
	for _, s := range filepath.SplitList(path) {
		// Add all paths except the one defined by item
		if !strings.EqualFold(s, item) && !strings.EqualFold(s, item+string(filepath.Separator)) {
			newPath += s + string(filepath.ListSeparator)
		}
	}

	return newPath
}
