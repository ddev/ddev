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
	// The markers set PATH apart from what the shell's startup files print.
	const mark = "__DDEV_PATH__"
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", `printf '`+mark+`%s`+mark+`' "$PATH"`).Output()
	if err != nil {
		return
	}
	parts := strings.Split(string(out), mark)
	if len(parts) != 3 || parts[1] == "" {
		return
	}
	_ = os.Setenv("PATH", parts[1])
}

// findDdev returns the ddev binary to run: $DDEV_BINARY, the ddev next to
// this app's executable, the first ddev on PATH, or one where DDEV's
// installers put it. A package installs the app next to its ddev, so they
// are one version.
func findDdev() (string, error) {
	if b := os.Getenv("DDEV_BINARY"); b != "" {
		return b, nil
	}
	if b := siblingDdev(); b != "" {
		return b, nil
	}
	name := ddevName()
	if b, err := exec.LookPath(name); err == nil {
		return b, nil
	}
	for _, dir := range wellKnownDirs() {
		if b := filepath.Join(dir, name); isFile(b) {
			return b, nil
		}
	}
	return "", errors.New("ddev was not found next to this app or on PATH (" + strings.ReplaceAll(os.Getenv("PATH"), string(os.PathListSeparator), ", ") + "). Install DDEV, or set DDEV_BINARY.")
}

// siblingDdev returns the ddev next to this app's executable, or next to
// DDEV.app on macOS, as `ddev utility download-ddev` leaves them; or "".
func siblingDdev() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return ddevBeside(filepath.Dir(exe))
}

func ddevBeside(exeDir string) string {
	dirs := []string{exeDir}
	if filepath.Base(exeDir) == "MacOS" && strings.HasSuffix(filepath.Dir(filepath.Dir(exeDir)), ".app") {
		dirs = append(dirs, filepath.Dir(filepath.Dir(filepath.Dir(exeDir))))
	}
	for _, d := range dirs {
		if b := filepath.Join(d, ddevName()); isFile(b) {
			return b
		}
	}
	return ""
}

// wellKnownDirs are where Homebrew, the install script, and Linux packages
// put ddev, for a login shell whose PATH lacks them. Tests replace it.
var wellKnownDirs = func() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/opt/homebrew/bin", "/usr/local/bin"}
	case "linux":
		return []string{"/usr/bin", "/usr/local/bin", "/home/linuxbrew/.linuxbrew/bin"}
	}
	return nil
}

func ddevName() string {
	if runtime.GOOS == "windows" {
		return "ddev.exe"
	}
	return "ddev"
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
