package versionconstants

// DdevVersion is the current version of DDEV, by default the Git committish (should be current Git tag)
var DdevVersion = "v0.0.0-overridden-by-make" // Note that this is overridden by make

// AmplitudeAPIKey is the ddev-specific key for Amplitude service
// Compiled with link-time variables
var AmplitudeAPIKey = ""

// WebImg defines the default web image used for applications.
var WebImg = "ddev/ddev-webserver"

// WebTag defines the default web image tag
var WebTag = "20240610_mysql_clients" // Note that this can be overridden by make

// DBImg defines the default db image used for applications.
var DBImg = "ddev/ddev-dbserver"

// BaseDBTag is the main tag, DBTag is constructed from it
var BaseDBTag = "20240608_rfay_bump_mysql_8.0.36"

const TraditionalRouterImage = "ddev/ddev-nginx-proxy-router:v1.23.1"
const TraefikRouterImage = "ddev/ddev-traefik-router:v1.23.1"

// SSHAuthImage is image for agent
var SSHAuthImage = "ddev/ddev-ssh-agent"

// SSHAuthTag is ssh-agent auth tag
var SSHAuthTag = "20240523_stasadev_apt_get_update_or_true"

// BusyboxImage is used a couple of places for a quick-pull
var BusyboxImage = "busybox:stable"

// BUILDINFO is information with date and context, supplied by make
var BUILDINFO = "BUILDINFO should have new info"

// MutagenVersion is filled with the version we find for Mutagen in use
var MutagenVersion = ""

const RequiredMutagenVersion = "0.17.2"

const RequiredDockerComposeVersionDefault = "v2.27.0"

// Drupal11RequiredSqlite3Version for ddev-webserver
const Drupal11RequiredSqlite3Version = "3.45.1"
