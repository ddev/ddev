package versionconstants

import (
	"fmt"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/nodeps"
)

// DdevVersion is the current version of ddev, by default the git committish (should be current git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// SegmentKey is the ddev-specific key for Segment service
// Compiled with link-time variables
var SegmentKey = ""

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "v1.22.0-beta1" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "v1.22.0-beta1"

const TraditionalRouterImage = "ddev/ddev-router:v1.22.0-beta1"
const TraefikRouterImage = "traefik:v2.10"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.22.0-beta1"

// Busybox is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.17.2"

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	fullWebImg := WebImg
	if globalconfig.DdevGlobalConfig.UseHardenedImages {
		fullWebImg = fullWebImg + "-prod"
	}
	return fmt.Sprintf("%s:%s", fullWebImg, WebTag)
}

// GetDBImage returns the correctly formatted db image:tag reference
func GetDBImage(dbType string, dbVersion string) string {
	v := nodeps.MariaDBDefaultVersion
	if dbVersion != "" {
		v = dbVersion
	}
	if dbType == "" {
		dbType = nodeps.MariaDB
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

// GetSSHAuthImage returns the correctly formatted sshauth image:tag reference
func GetSSHAuthImage() string {
	return fmt.Sprintf("%s:%s", SSHAuthImage, SSHAuthTag)
}

// GetRouterImage returns the router image:tag reference
func GetRouterImage() string {
	image := TraefikRouterImage
	if globalconfig.DdevGlobalConfig.Router == types.RouterTypeNginxProxy {
		image = TraditionalRouterImage
	}
	return image
}
