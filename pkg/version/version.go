package version

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/fsouza/go-dockerclient"
	"os/exec"
	"runtime"
	"strings"
)

// MariaDBDefaultVersion is the default version we use in the db container
const MariaDBDefaultVersion = "10.2"

// VERSION is supplied with the git committish this is built from
var VERSION = ""

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// DdevVersion is the current version of ddev, by default the git committish (should be current git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// SentryDSN is the ddev-specific key for the Sentry service.
// It is compiled in using link-time variables
var SentryDSN = ""

// SegmentKey is the ddev-specific key for Segment service
// Compiled with link-time variables
var SegmentKey = ""

// DockerVersionConstraint is the current minimum version of docker required for ddev.
// See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
// REMEMBER TO CHANGE docs/index.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/drud/ddev/pull/738.. and regression https://github.com/drud/ddev/issues/1431
var DockerVersionConstraint = ">= 18.06.1-alpha1"

// DockerComposeVersionConstraint is the current minimum version of docker-compose required for ddev.
// REMEMBER TO CHANGE docs/index.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/drud/ddev/pull/738.. and regression https://github.com/drud/ddev/issues/1431
var DockerComposeVersionConstraint = ">= 1.21.0-alpha1"

// DockerComposeFileFormatVersion is the compose version to be used
var DockerComposeFileFormatVersion = "3.6"

// WebImg defines the default web image used for applications.
var WebImg = "drud/ddev-webserver"

// WebTag defines the default web image tag for drud dev
var WebTag = "20190810_php74" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "drud/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "v1.10.0"

// DBAImg defines the default phpmyadmin image tag used for applications.
var DBAImg = "drud/phpmyadmin"

// DBATag defines the default phpmyadmin image tag used for applications.
var DBATag = "v1.10.0" // Note that this can be overridden by make

// BgsyncImg defines the default bgsync image tag used for applications.
var BgsyncImg = "drud/ddev-bgsync"

// BgsyncTag defines the default phpmyadmin image tag used for applications.
var BgsyncTag = "v1.10.0" // Note that this can be overridden by make

// RouterImage defines the image used for the router.
var RouterImage = "drud/ddev-router"

// RouterTag defines the tag used for the router.
var RouterTag = "v1.10.0" // Note that this can be overridden by make

var SSHAuthImage = "drud/ddev-ssh-agent"

var SSHAuthTag = "v1.10.2"

// COMMIT is the actual committish, supplied by make
var COMMIT = "COMMIT should be overridden"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// DockerVersion is cached version of docker
var DockerVersion = ""

// DockerComposeVersion is filled with the version we find for docker-compose
var DockerComposeVersion = ""

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	var err error
	versionInfo := make(map[string]string)

	versionInfo["DDEV-Local version"] = DdevVersion
	versionInfo["web"] = GetWebImage()
	versionInfo["db"] = GetDBImage()
	versionInfo["dba"] = GetDBAImage()
	versionInfo["bgsync"] = BgsyncImg + ":" + BgsyncTag
	versionInfo["router"] = RouterImage + ":" + RouterTag
	versionInfo["ddev-ssh-agent"] = SSHAuthImage + ":" + SSHAuthTag
	versionInfo["commit"] = COMMIT
	versionInfo["build info"] = BUILDINFO
	versionInfo["os"] = runtime.GOOS
	if versionInfo["docker"], err = GetDockerVersion(); err != nil {
		versionInfo["docker"] = fmt.Sprintf("failed to GetDockerVersion(): %v", err)
	}
	if versionInfo["docker-compose"], err = GetDockerComposeVersion(); err != nil {
		versionInfo["docker-compose"] = fmt.Sprintf("failed to GetDockerComposeVersion(): %v", err)
	}
	if runtime.GOOS == "windows" {
		if nodeps.IsDockerToolbox() {
			versionInfo["docker type"] = "Docker Toolbox"
		} else {
			versionInfo["docker type"] = "Docker Desktop For Windows"
		}
	}

	return versionInfo
}

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	return fmt.Sprintf("%s:%s", WebImg, WebTag)
}

// GetDBImage returns the correctly formatted db image:tag reference
func GetDBImage(mariaDBVersion ...string) string {
	version := MariaDBDefaultVersion
	if len(mariaDBVersion) > 0 {
		version = mariaDBVersion[0]
	}
	return fmt.Sprintf("%s:%s", DBImg, BaseDBTag+"-"+version)
}

// GetDBAImage returns the correctly formatted dba image:tag reference
func GetDBAImage() string {
	return fmt.Sprintf("%s:%s", DBAImg, DBATag)
}

// GetDBAImage returns the correctly formatted dba image:tag reference
func GetBgsyncImage() string {
	return fmt.Sprintf("%s:%s", BgsyncImg, BgsyncTag)
}

// GetSSHAuthImage returns the correctly formatted sshauth image:tag reference
func GetSSHAuthImage() string {
	return fmt.Sprintf("%s:%s", SSHAuthImage, SSHAuthTag)
}

// GetRouterImage returns the correctly formatted router image:tag reference
func GetRouterImage() string {
	return fmt.Sprintf("%s:%s", RouterImage, RouterTag)
}

// GetDockerComposeVersion runs docker-compose -v to get the current version
func GetDockerComposeVersion() (string, error) {
	if DockerComposeVersion != "" {
		return DockerComposeVersion, nil
	}

	executableName := "docker-compose"

	path, err := exec.LookPath(executableName)
	if err != nil {
		return "", fmt.Errorf("no docker-compose")
	}

	out, err := exec.Command(path, "version", "--short").Output()
	if err != nil {
		return "", err
	}

	v := string(out)
	DockerComposeVersion = strings.TrimSpace(v)
	return DockerComposeVersion, nil
}

// GetDockerVersion gets the cached or api-sourced version of docker engine
func GetDockerVersion() (string, error) {
	if DockerVersion != "" {
		return DockerVersion, nil
	}
	var client *docker.Client
	var err error
	if client, err = docker.NewClientFromEnv(); err != nil {
		return "", err
	}

	v, err := client.Version()
	if err != nil {
		return "", err
	}
	DockerVersion = v.Get("Version")

	return DockerVersion, nil
}
