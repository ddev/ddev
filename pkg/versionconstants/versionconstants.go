package versionconstants

// DdevVersion is the current version of DDEV, by default the Git committish (should be current Git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20250806_rfay_php_addon" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "2050813_rfay_mariadb_fail"

// TraefikRouterImage is image for router
var TraefikRouterImage = "ddev/ddev-traefik-router"

// TraefikRouterTag is traefik router tag
var TraefikRouterTag = "20250710_stasadev_traefik_healthcheck"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.24.7"

// XhguiImage is image for xhgui
var XhguiImage = "ddev/ddev-xhgui"

// XhguiTag is xhgui tag
var XhguiTag = "v1.24.7"

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
const RequiredDockerComposeVersionDefault = "v2.38.2"
