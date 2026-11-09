package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	name := ddevName()
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

func ddevName() string {
	if runtime.GOOS == "windows" {
		return "ddev.exe"
	}
	return "ddev"
}

var ddevVersionLine = regexp.MustCompile(`ddev version (\S+)`)

// versionMismatches returns a line for each of bins whose version is not
// want. The app reads projects with its own built-in code, so it can show
// them differently from a ddev of another version, and two versions that
// take turns starting the same projects keep rewriting each other's files.
func versionMismatches(want string, bins []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range bins {
		if b == "" || seen[b] {
			continue
		}
		seen[b] = true
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		res, err := exec.CommandContext(ctx, b, "--version").Output()
		cancel()
		m := ddevVersionLine.FindSubmatch(res)
		switch {
		case err != nil:
			out = append(out, fmt.Sprintf("%s: %v", b, err))
		case m == nil:
			out = append(out, b+" printed no version")
		case string(m[1]) != want:
			out = append(out, fmt.Sprintf("%s is %s", b, m[1]))
		}
	}
	return out
}

func isFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
