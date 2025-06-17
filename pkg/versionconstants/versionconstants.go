package versionconstants

// DdevVersion is the current version of DDEV, by default the Git committish (should be current Git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20250612_stasadev_rebuild_images" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "20250213_rfay_mariadb_11.8"

// TraefikRouterImage is image for router
var TraefikRouterImage = "ddev/ddev-traefik-router"

// TraefikRouterTag is traefik router tag
var TraefikRouterTag = "20250612_stasadev_rebuild_images"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.24.6"

// BusyboxImage is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// XhguiImage is image for xhgui
var XhguiImage = "ddev/ddev-xhgui"

// XhguiTag is xhgui tag
var XhguiTag = "v1.24.6"

// UtilitiesImage is used in bash scripts
var UtilitiesImage = "ddev/ddev-utilities"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.18.1"

const RequiredDockerComposeVersionDefault = "v2.36.1"
