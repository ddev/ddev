package version

// VERSION is supplied with the git committish this is built from
var VERSION = ""

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// DdevVersion is the current version of ddev, by default the git committish (should be current git tag)
var DdevVersion = "v0.3.0-dev" // Note that this is overridden by make

// DockerVersionConstraint is the current minimum version of docker required for ddev.
// See https://godoc.org/github.com/Masterminds/semver#hdr-Checking_Version_Constraints
// for examples defining version constraints.
var DockerVersionConstraint = ">= 17.05.0-ce-alpha1"

// DockerComposeVersionConstraint is the current minimum version of docker-compose required for ddev.
var DockerComposeVersionConstraint = ">= 1.10.0-alpha1"

// WebImg defines the default web image used for applications.
var WebImg = "drud/nginx-php-fpm-local" // Note that this is overridden by make

// WebTag defines the default web image tag for drud dev
var WebTag = "v1.2.2" // Note that this is overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "drud/mariadb-local" // Note that this is overridden by make

// DBTag defines the default db image tag for drud dev
var DBTag = "v0.9.0" // Note that this is overridden by make

// DBAImg defines the default phpmyadmin image tag used for applications.
var DBAImg = "drud/phpmyadmin"

// DBATag defines the default phpmyadmin image tag used for applications.
var DBATag = "v0.2.0"

// RouterImage defines the image used for the router.
var RouterImage = "drud/ddev-router" // Note that this is overridden by make

// RouterTag defines the tag used for the router.
var RouterTag = "v0.5.0" // Note that this is overridden by make

// COMMIT is the actual committish, supplied by make
var COMMIT = "COMMIT should be overridden"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// DDevTLD defines the tld to use for DDev site URLs.
const DDevTLD = "ddev.local"

// GetVersionInfo returns a map containing the version info defined above.
func GetVersionInfo() map[string]string {
	versionInfo := make(map[string]string)

	versionInfo["cli"] = DdevVersion
	versionInfo["web"] = WebImg + ":" + WebTag
	versionInfo["db"] = DBImg + ":" + DBTag
	versionInfo["dba"] = DBAImg + ":" + DBATag
	versionInfo["router"] = RouterImage + ":" + RouterTag
	versionInfo["commit"] = COMMIT
	versionInfo["domain"] = DDevTLD
	versionInfo["build info"] = BUILDINFO

	return versionInfo
}
