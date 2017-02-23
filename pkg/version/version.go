package version

// VERSION is supplied with the git committish this is built from
var VERSION = ""

// IMPORTANT: These versions are overridden by version ldflags specifications VERSION_VARIABLES in the Makefile

// Current version of the ddev tool, by default the git committish (should be current git tag)
var DdevVersion = ""  // 0.4.0

// WebImg defines the default web image for drud dev
var WebImg = ""  // drud/nginx-php-fpm7-local

// WebTag defines the default web image tag for drud dev
var WebTag = "" // 0.2.0

// DBImg defines the default db image for drud dev
var DBImg = ""  // drud/mysql-docker-local-57

// DBTag defines the default db image tag for drud dev
var DBTag = "" // 0.1.0

// RouterImage defines the image used for the drud dev router.
var RouterImage = "" // drud/nginx-proxy

// RouterTag defines the tag used for the drud dev router.
var RouterTag = ""  // 0.1.0
