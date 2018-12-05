package version

import (
	"fmt"
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

// DockerVersionConstraint is the current minimum version of docker required for ddev.
// See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
// REMEMBER TO CHANGE docs/index.md if you touch this!
var DockerVersionConstraint = ">= 18.06.0-ce"

// DockerComposeVersionConstraint is the current minimum version of docker-compose required for ddev.
// REMEMBER TO CHANGE docs/index.md if you touch this!
var DockerComposeVersionConstraint = ">= 1.21.0"

// DockerComposeFileFormatVersion is the compose version to be used
var DockerComposeFileFormatVersion = "3.6"

// WebImg defines the default web image used for applications.
var WebImg = "drud/ddev-webserver"

// WebTag defines the default web image tag for drud dev
var WebTag = "20181124_pecl_upload_progress" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "drud/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "20181203_mariadb_2_versions"

// DBAImg defines the default phpmyadmin image tag used for applications.
var DBAImg = "drud/phpmyadmin"

// DBATag defines the default phpmyadmin image tag used for applications.
var DBATag = "v1.4.0" // Note that this can be overridden by make

// BgsyncImg defines the default bgsync image tag used for applications.
var BgsyncImg = "drud/ddev-bgsync"

// BgsyncTag defines the default phpmyadmin image tag used for applications.
var BgsyncTag = "20181117_bgsync" // Note that this can be overridden by make

// RouterImage defines the image used for the router.
var RouterImage = "drud/ddev-router"

// RouterTag defines the tag used for the router.
var RouterTag = "v1.4.0" // Note that this can be overridden by make

var SSHAuthImage = "drud/ddev-ssh-agent"

var SSHAuthTag = "20181122_load_all_keys"

// COMMIT is the actual committish, supplied by make
var COMMIT = "COMMIT should be overridden"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// DockerVersion is cached version of docker
var DockerVersion = ""

// DockerComposeVersion is filled with the version we find for docker-compose
var DockerComposeVersion = ""

// DDevTLD defines the tld to use for DDev site URLs.
const DDevTLD = "ddev.local"

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	versionInfo := make(map[string]string)

	versionInfo["cli"] = DdevVersion
	versionInfo["web"] = GetWebImage()
	versionInfo["db"] = GetDBImage()
	versionInfo["dba"] = GetDBAImage()
	versionInfo["bgsync"] = BgsyncImg + ":" + BgsyncTag
	versionInfo["router"] = RouterImage + ":" + RouterTag
	versionInfo["ddev-ssh-agent"] = SSHAuthImage + ":" + SSHAuthTag
	versionInfo["commit"] = COMMIT
	versionInfo["domain"] = DDevTLD
	versionInfo["build info"] = BUILDINFO
	versionInfo["docker"] = DockerVersion
	versionInfo["docker-compose"] = DockerComposeVersion

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
