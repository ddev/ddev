package versionconstants

// DdevVersion is the current version of DDEV, by default the Git committish (should be current Git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20241117_remove_python_django" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "20241109_use_bin_env"

const TraditionalRouterImage = "ddev/ddev-nginx-proxy-router:20241109_use_bin_env"
const TraefikRouterImage = "ddev/ddev-traefik-router:20241109_use_bin_env"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "20241115_rfay_ssh_agent_update"

// BusyboxImage is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// UtilitiesImage is used in bash scripts
var UtilitiesImage = "ddev/ddev-utilities"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.18.0"

const RequiredDockerComposeVersionDefault = "v2.30.3"
