package versionconstants

import (
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
)

// DdevVersion is the current version of DDEV. Normally set via -ldflags from the Makefile
// using `git describe --tags --always --dirty`. If not provided, we derive a best-effort
// value. Prefer VERSION env var, otherwise use Go build-info hash.
var DdevVersion = "" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20260213_stasadev_mariadb_skip_ssl" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "v1.25.0"

// TraefikRouterImage is image for router
var TraefikRouterImage = "ddev/ddev-traefik-router"

// TraefikRouterTag is traefik router tag
var TraefikRouterTag = "20260210_jonesrussell_no_wrn_traefik_stderr"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.25.0"

// XhguiImage is image for xhgui
var XhguiImage = "ddev/ddev-xhgui"

// XhguiTag is xhgui tag
var XhguiTag = "v1.25.0"

// UtilitiesImage is used in bash scripts
var UtilitiesImage = "ddev/ddev-utilities:latest"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

// RequiredMutagenVersion defines the required version of Mutagen
const RequiredMutagenVersion = "0.18.1"

// RequiredDockerComposeVersionDefault defines the required version of docker-compose
// Keep this in sync with github.com/compose-spec/compose-go/v2 in go.mod,
// matching the version used in https://github.com/docker/compose/blob/main/go.mod for the same tag
const RequiredDockerComposeVersionDefault = "v5.0.2"

// ---
// Fallback version derivation for developer builds not using the Makefile
// ---

func init() {
	if DdevVersion == "" {
		// 1) Explicit env override: VERSION=vX.Y.Z ddev ...
		if v := deriveVersionFromEnv(); v != "" {
			DdevVersion = v
			return
		}
		// 2) Fall back to build info short hash
		if v := deriveVersionFromBuildInfo(); v != "" {
			DdevVersion = v
			return
		}
		// 3) Try direct git command as final fallback
		if v := deriveVersionFromGit(); v != "" {
			DdevVersion = v
			return
		}
		// 4) Last resort - use build info without VCS or unknown version
		DdevVersion = "v0.0.0-overridden-by-make"
	}
}

// deriveVersionFromEnv reads VERSION environment variable (if set) and returns it.
func deriveVersionFromEnv() string {
	v := strings.TrimSpace(os.Getenv("VERSION"))
	return v
}

// deriveVersionFromBuildInfo uses Go's embedded VCS info (enabled by default since Go 1.18+)
// to produce a short commit-based version like "gabcd123[-dirty]".
func deriveVersionFromBuildInfo() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	var rev string
	var dirty bool
	var hasModifiedInfo bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			hasModifiedInfo = true
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return ""
	}
	short := rev
	if len(short) > 7 {
		short = short[:7]
	}
	v := "g" + short
	// If we don't have modified info from build, assume it might be dirty
	// since we're in a development context
	if dirty || !hasModifiedInfo {
		v += "-dirty"
	}
	return v
}

var gitVersionCache string
var gitVersionCacheInitialized bool

// deriveVersionFromGit attempts to run git describe directly to get version info.
// This is used as a fallback when build info doesn't contain VCS information.
// The result is cached to avoid repeated git command execution.
func deriveVersionFromGit() string {
	if gitVersionCacheInitialized {
		return gitVersionCache
	}

	gitVersionCacheInitialized = true

	// Try git describe --tags --always --dirty (same as Makefile)
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	if output, err := cmd.Output(); err == nil {
		gitVersionCache = strings.TrimSpace(string(output))
		return gitVersionCache
	}

	// Fallback to just getting the commit hash
	cmd = exec.Command("git", "rev-parse", "--short=7", "HEAD")
	if output, err := cmd.Output(); err == nil {
		hash := strings.TrimSpace(string(output))
		if hash != "" {
			// Check if working directory is dirty
			cmd = exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
			if err := cmd.Run(); err != nil {
				gitVersionCache = "g" + hash + "-dirty"
			} else {
				gitVersionCache = "g" + hash
			}
			return gitVersionCache
		}
	}

	gitVersionCache = ""
	return gitVersionCache
}
