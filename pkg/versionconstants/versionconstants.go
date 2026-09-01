package versionconstants

import (
	"os"
	"os/exec"
	"regexp"
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

// BuildSource records where a build came from: on GitHub Actions, a run URL
// with the PR number appended when triggered by a pull_request event;
// otherwise who ran `make` and from which branch. Set via -ldflags from the
// Makefile — see scripts/build-source.sh.
var BuildSource = ""

// unreleasedVersionPattern matches a DdevVersion that isn't a clean release
// tag: the `git describe --tags --always --dirty` long form left by commits
// past a tag (v1.2.3-15-gabcdef1), a dirty working tree (-dirty), or the
// debug.BuildInfo short-hash fallback used when a build has no embedded git
// tag info (gabcdef1).
var unreleasedVersionPattern = regexp.MustCompile(`-\d+-g[0-9a-f]{6,}|-dirty$|^g[0-9a-f]{6,}$`)

// IsUnreleasedDdevVersion returns true if version looks like it was built
// from an untagged commit (a PR, local branch, or other dev build) rather
// than an official release.
func IsUnreleasedDdevVersion(version string) bool {
	return unreleasedVersionPattern.MatchString(version)
}

// Image tags are bare content hashes (see containers/hash-paths.sh), so the
// same image resolves to the same tag no matter which branch or repository
// published it. Each has a companion <Name>Branch naming the branch its
// content was built from - the readable hint a bare hash costs, shown by
// `ddev version` and republished as a <branch>-<hash> alias in the registry.
// All of these lines are maintained by containers/autotag.sh; running `make`
// updates them.

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "c202e92108" // 20260814_rfay_docker_update_phase_2-c202e92108

// WebTagBranch is the branch WebTag's content was built from.
var WebTagBranch = "20260814_rfay_docker_update_phase_2"

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "cca00aa01d" // 20260831_rfay_mysql_race-cca00aa01d

// BaseDBTagBranch is the branch BaseDBTag's content was built from.
var BaseDBTagBranch = "20260831_rfay_mysql_race"

// TraefikRouterImage is image for router
var TraefikRouterImage = "ddev/ddev-traefik-router"

// TraefikRouterTag is traefik router tag
var TraefikRouterTag = "bffcda31c5" // 20260814_rfay_docker_update_phase_2-bffcda31c5

// TraefikRouterTagBranch is the branch TraefikRouterTag's content was built from.
var TraefikRouterTagBranch = "20260814_rfay_docker_update_phase_2"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "bb5e9f0003" // 20260814_rfay_docker_update_phase_2-bb5e9f0003

// SSHAuthTagBranch is the branch SSHAuthTag's content was built from.
var SSHAuthTagBranch = "20260814_rfay_docker_update_phase_2"

// XhguiImage is image for xhgui
var XhguiImage = "ddev/ddev-xhgui"

// XhguiTag is xhgui tag
var XhguiTag = "8757c1e92a" // 20260814_rfay_docker_update_phase_2-8757c1e92a

// XhguiTagBranch is the branch XhguiTag's content was built from.
var XhguiTagBranch = "20260814_rfay_docker_update_phase_2"

// UtilitiesImage is used in bash scripts
var UtilitiesImage = "ddev/ddev-utilities:latest"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

// RequiredMutagenVersion defines the required version of Mutagen
// This value is a hard requirement
const RequiredMutagenVersion = "0.18.1"

// DockerBuildxMinVersion defines the minimum required version of buildx.
// Must match buildxMinVersion in vendor/github.com/docker/compose/v5/pkg/compose/api_versions.go
// Sync enforced by TestBuildxMinVersionInSync.
// This value is a hard requirement
const DockerBuildxMinVersion = "0.17.0"

// DockerBuildxRecommendedVersion defines the recommended version of buildx
// to use if the installed version doesn't match the minimum.
// Sync enforced by TestBuildxRecommendedVersionInSync.
// This value is a recommendation, not a hard requirement
const DockerBuildxRecommendedVersion = "0.36.1"

// DockerMinVersion defines the recommended minimum version of Docker Engine
// List of supported Docker versions: https://endoflife.date/docker-engine
// This value is a recommendation, not a hard requirement
const DockerMinVersion = "25.0"

// DockerMinAPIVersion defines the recommended minimum API version of Docker Engine
// Should be in sync with DockerMinVersion
// See https://docs.docker.com/reference/api/engine/#api-version-matrix
// This value is a recommendation, not a hard requirement
const DockerMinAPIVersion = "1.44"

// PodmanMinVersion defines the recommended minimum version of Podman.
// Podman 4.x doesn't run properly on GitHub runners, so we recommend Podman 5.0+
// This value is a recommendation, not a hard requirement
const PodmanMinVersion = "5.0"

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
