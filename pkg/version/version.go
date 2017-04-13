package version

// VERSION is supplied with the git committish this is built from
var VERSION = ""

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// DdevVersion is the current version of the ddev tool, by default the git committish (should be current git tag)
var DdevVersion = "v0.1.0-dev" // Note that this is overridden by make

// WebImg defines the default web image for drud dev
var WebImg = "drud/nginx-php-fpm7-local" // Note that this is overridden by make

// WebTag defines the default web image tag for drud dev
var WebTag = "v0.3.4" // Note that this is overridden by make

// DBImg defines the default db image for drud dev
var DBImg = "drud/mysql-docker-local-57" // Note that this is overridden by make

// DBTag defines the default db image tag for drud dev
var DBTag = "v0.2.0" // Note that this is overridden by make

// RouterImage defines the image used for the drud dev router.
var RouterImage = "drud/nginx-proxy" // Note that this is overridden by make

// RouterTag defines the tag used for the drud dev router.
var RouterTag = "v0.2.0" // Note that this is overridden by make
