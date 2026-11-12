package testsetup

import (
	"fmt"
	"log"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
)

// ResolveDdevBinary returns the DDEV binary that tests should execute.
// If DDEV_BINARY_FULLPATH is set, it is used directly. Otherwise the binary
// is built from source via `make`.
func ResolveDdevBinary() (string, error) {
	if bin := os.Getenv("DDEV_BINARY_FULLPATH"); bin != "" {
		if !fileutil.FileExists(bin) {
			return "", fmt.Errorf("DDEV_BINARY_FULLPATH=%s does not exist", bin)
		}
		return bin, nil
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to locate repository root: %w", err)
	}

	binaryName := "ddev"
	if nodeps.IsWindows() {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(repoRoot, ".gotmp", "bin", runtime.GOOS+"_"+runtime.GOARCH, binaryName)

	cmd := osexec.Command("make")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("`make` failed: %w", err)
	}
	if !fileutil.FileExists(binaryPath) {
		return "", fmt.Errorf("`make` succeeded but %s was not found", binaryPath)
	}
	return binaryPath, nil
}

// MustResolveDdevBinary returns the test DDEV binary or aborts the current test process.
func MustResolveDdevBinary() string {
	bin, err := ResolveDdevBinary()
	if err != nil {
		log.Fatalf("MustResolveDdevBinary: %v", err)
	}
	return bin
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if fileutil.FileExists(filepath.Join(dir, "go.mod")) && fileutil.FileExists(filepath.Join(dir, "Makefile")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found from %s", dir)
		}
		dir = parent
	}
}
