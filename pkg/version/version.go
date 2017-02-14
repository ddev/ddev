package version

// VERSION is supplied with the git committish this is built from
var VERSION = ""

// CliVersion is the current version of the ddev tool
var CliVersion = "0.4.0"

// WebImg defines the default web image for drud dev
var WebImg = "drud/nginx-php-fpm7-local"

// WebTag defines the default web image tag for drud dev
var WebTag = "0.2.0"

// DBImg defines the default db image for drud dev
var DBImg = "drud/mysql-docker-local"

// DBTag defines the default db image tag for drud dev
var DBTag = "5.7"

// RouterImage defines the image used for the drud dev router.
var RouterImage = "drud/nginx-proxy"

// RouterTag defines the tag used for the drud dev router.
var RouterTag = "0.1.0"
