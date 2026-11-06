package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// useLoginShellPath takes PATH from the user's login shell. An app started
// from the Finder or a desktop launcher gets a minimal PATH, without
// Homebrew, so neither ddev nor the docker and mkcert it runs would be found.
func useLoginShellPath() {
	if runtime.GOOS == "windows" {
		return
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", `printf '%s' "$PATH"`).Output()
	if err != nil || len(out) == 0 {
		return
	}
	_ = os.Setenv("PATH", string(out))
}

// findDdev returns the ddev binary that actions run: $DDEV_BINARY, a ddev
// next to this app's executable, or the first ddev on PATH.
func findDdev() (string, error) {
	if b := os.Getenv("DDEV_BINARY"); b != "" {
		return b, nil
	}
	name := "ddev"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		if b := filepath.Join(filepath.Dir(exe), name); isFile(b) {
			return b, nil
		}
	}
	if b, err := exec.LookPath(name); err == nil {
		return b, nil
	}
	return "", errors.New("ddev was not found on PATH (" + strings.ReplaceAll(os.Getenv("PATH"), string(os.PathListSeparator), ", ") + "). Install DDEV, or set DDEV_BINARY.")
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
