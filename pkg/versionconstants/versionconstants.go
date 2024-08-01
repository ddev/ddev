package versionconstants

// DdevVersion is the current version of DDEV, by default the Git committish (should be current Git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20240723_stasadev_n_install_auto" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "20240719_rfay_innodb_doublewrite"

const TraditionalRouterImage = "ddev/ddev-nginx-proxy-router:v1.23.3"
const TraefikRouterImage = "ddev/ddev-traefik-router:20240616_traefik_3"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "v1.23.3"

// BusyboxImage is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// UtilitiesImage is used in bash scripts
var UtilitiesImage = "ddev/ddev-utilities"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.17.2"

const RequiredDockerComposeVersionDefault = "v2.29.0"

// Drupal11RequiredSqlite3Version for ddev-webserver
const Drupal11RequiredSqlite3Version = "3.45.1"
