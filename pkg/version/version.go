package version

import (
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/fsouza/go-dockerclient"
	"os/exec"
	"runtime"
	"strings"
)

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// DdevVersion is the current version of ddev, by default the git committish (should be current git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// SegmentKey is the ddev-specific key for Segment service
// Compiled with link-time variables
var SegmentKey = ""

// DockerVersionConstraint is the current minimum version of docker required for ddev.
// See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
// REMEMBER TO CHANGE docs/index.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/drud/ddev/pull/738.. and regression https://github.com/drud/ddev/issues/1431
var DockerVersionConstraint = ">= 19.03.9-alpha1"

// DockerComposeVersionConstraint is the versions allowed for ddev
// REMEMBER TO CHANGE docs/index.md if you touch this!
// The constraint MUST HAVE a -pre of some kind on it for successful comparison.
// See https://github.com/drud/ddev/pull/738.. and regression https://github.com/drud/ddev/issues/1431
var DockerComposeVersionConstraint = ">= 1.25.0-alpha1 < 2.0.0-alpha1 || >= v2.0.0-rc.2"

// DockerComposeFileFormatVersion is the compose version to be used
var DockerComposeFileFormatVersion = "3.6"

// WebImg defines the default web image used for applications.
var WebImg = "drud/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20220304_nvm" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "drud/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "v1.19.0"

// DBAImg defines the default phpmyadmin image tag used for applications.
var DBAImg = "phpmyadmin"

// DBATag defines the default phpmyadmin image tag used for applications.
var DBATag = "5" // Note that this can be overridden by make

// RouterImage defines the image used for the router.
var RouterImage = "drud/ddev-router"

// RouterTag defines the tag used for the router.
var RouterTag = "v1.19.0" // Note that this can be overridden by make

// SSHAuthImage is image for agent
var SSHAuthImage = "drud/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.19.0"

// Busybox is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// DockerVersion is cached version of docker
var DockerVersion = ""

// DockerComposeVersion is filled with the version we find for docker-compose
var DockerComposeVersion = ""

// This is var instead of const so it can be changed in test, but should not otherwise be touched.
// Otherwise we can't test if the version on the machine is equal to version required
var RequiredDockerComposeVersion = "v2.2.2"

// MutagenVersion is filled with the version we find for mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.12.0"

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	var err error
	versionInfo := make(map[string]string)

	versionInfo["DDEV version"] = DdevVersion
	versionInfo["web"] = GetWebImage()
	versionInfo["db"] = GetDBImage(nodeps.MariaDB)
	versionInfo["dba"] = GetDBAImage()
	versionInfo["router"] = RouterImage + ":" + RouterTag
	versionInfo["ddev-ssh-agent"] = SSHAuthImage + ":" + SSHAuthTag
	versionInfo["build info"] = BUILDINFO
	versionInfo["os"] = runtime.GOOS
	versionInfo["architecture"] = runtime.GOARCH
	if versionInfo["docker"], err = GetDockerVersion(); err != nil {
		versionInfo["docker"] = fmt.Sprintf("failed to GetDockerVersion(): %v", err)
	}
	if versionInfo["docker-platform"], err = GetDockerPlatform(); err != nil {
		versionInfo["docker-platform"] = fmt.Sprintf("failed to GetDockerPlatform(): %v", err)
	}
	if versionInfo["docker-compose"], err = GetDockerComposeVersion(); err != nil {
		versionInfo["docker-compose"] = fmt.Sprintf("failed to GetDockerComposeVersion(): %v", err)
	}
	versionInfo["mutagen"] = RequiredMutagenVersion

	if runtime.GOOS == "windows" {
		versionInfo["docker type"] = "Docker Desktop For Windows"
	}

	return versionInfo
}

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	fullWebImg := WebImg
	if globalconfig.DdevGlobalConfig.UseHardenedImages {
		fullWebImg = fullWebImg + "-prod"
	}
	return fmt.Sprintf("%s:%s", fullWebImg, WebTag)
}

// GetDBImage returns the correctly formatted db image:tag reference
func GetDBImage(dbType string, dbVersion ...string) string {
	v := nodeps.MariaDBDefaultVersion
	if len(dbVersion) > 0 {
		v = dbVersion[0]
	}
	switch dbType {
	case nodeps.Postgres:
		return fmt.Sprintf("%s:%s", dbType, v)
	case nodeps.MySQL:
		fallthrough
	case nodeps.MariaDB:
		fallthrough
	default:
		return fmt.Sprintf("%s-%s-%s:%s", DBImg, dbType, v, BaseDBTag)
	}
}

// GetDBAImage returns the correctly formatted dba image:tag reference
func GetDBAImage() string {
	return fmt.Sprintf("%s:%s", DBAImg, DBATag)
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

	return GetLiveDockerComposeVersion()
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

// GetDockerPlatform gets the platform used for docker engine
func GetDockerPlatform() (string, error) {
	var client *docker.Client
	var err error
	if client, err = docker.NewClientFromEnv(); err != nil {
		return "", err
	}

	info, err := client.Info()
	if err != nil {
		return "", err
	}
	platform := info.Name

	return platform, nil
}

// GetLiveMutagenVersion runs `mutagen version` and caches result
func GetLiveMutagenVersion() (string, error) {
	if MutagenVersion != "" {
		return MutagenVersion, nil
	}

	mutagenPath := globalconfig.GetMutagenPath()

	if !fileutil.FileExists(mutagenPath) {
		MutagenVersion = ""
		return MutagenVersion, nil
	}
	out, err := exec.Command(mutagenPath, "version").Output()
	if err != nil {
		return "", err
	}

	v := string(out)
	MutagenVersion = strings.TrimSpace(v)
	return MutagenVersion, nil
}

// GetLiveDockerComposeVersion runs `docker-compose --version` and caches result
func GetLiveDockerComposeVersion() (string, error) {
	if DockerComposeVersion != "" {
		return DockerComposeVersion, nil
	}

	path, err := globalconfig.GetDockerComposePath()
	if err != nil {
		return "", err
	}

	DockerComposePath := path

	if !fileutil.FileExists(DockerComposePath) {
		DockerComposeVersion = ""
		return DockerComposeVersion, nil
	}
	out, err := exec.Command(DockerComposePath, "version", "--short").Output()
	if err != nil {
		return "", err
	}
	v := strings.Trim(string(out), "\r\n")

	// docker-compose v1 returns a version without the prefix "v", so add it.
	if strings.HasPrefix(v, "1") {
		v = "v" + v
	}

	DockerComposeVersion = v
	return DockerComposeVersion, nil
}

// GetRequiredDockerComposeVersion returns the version of docker-compose we need
// based on the compiled version, or overrides in globalconfig, like
// required_docker_compose_version and use_docker_compose_from_path
// In the case of UseDockerComposeFromPath there is no required version, so this
// will return empty string.
func GetRequiredDockerComposeVersion() string {
	v := RequiredDockerComposeVersion
	switch {
	case globalconfig.DdevGlobalConfig.UseDockerComposeFromPath:
		v = ""
	case globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion != "":
		v = globalconfig.DdevGlobalConfig.RequiredDockerComposeVersion
	}
	return v
}
