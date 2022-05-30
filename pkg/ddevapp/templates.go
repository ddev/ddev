package ddevapp

// ConfigInstructions is used to add example hooks usage
const ConfigInstructions = `
# Key features of ddev's config.yaml:

# name: <projectname> # Name of the project, automatically provides
#   http://projectname.ddev.site and https://projectname.ddev.site

# type: <projecttype>  # drupal6/7/8, backdrop, typo3, wordpress, php

# docroot: <relative_path> # Relative path to the directory containing index.php.

# php_version: "7.4"  # PHP version to use, "5.6", "7.0", "7.1", "7.2", "7.3", "7.4", "8.0", "8.1"

# You can explicitly specify the webimage but this
# is not recommended, as the images are often closely tied to ddev's' behavior,
# so this can break upgrades.

# webimage: <docker_image>  # nginx/php docker image.

# database:
#   type: <dbtype> # mysql, mariadb
#   version: <version> # database version, like "10.3" or "8.0"
# Note that mariadb_version or mysql_version from v1.18 and earlier
# will automatically be converted to this notation with just a "ddev config --auto"

# router_http_port: <port>  # Port to be used for http (defaults to port 80)
# router_https_port: <port> # Port for https (defaults to 443)

# xdebug_enabled: false  # Set to true to enable xdebug and "ddev start" or "ddev restart"
# Note that for most people the commands
# "ddev xdebug" to enable xdebug and "ddev xdebug off" to disable it work better,
# as leaving xdebug enabled all the time is a big performance hit.

# xhprof_enabled: false  # Set to true to enable xhprof and "ddev start" or "ddev restart"
# Note that for most people the commands
# "ddev xhprof" to enable xhprof and "ddev xhprof off" to disable it work better,
# as leaving xhprof enabled all the time is a big performance hit.

# webserver_type: nginx-fpm  # or apache-fpm

# timezone: Europe/Berlin
# This is the timezone used in the containers and by PHP;
# it can be set to any valid timezone,
# see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
# For example Europe/Dublin or MST7MDT

# composer_root: <relative_path>
# Relative path to the composer root directory from the project root. This is
# the directory which contains the composer.json and where all Composer related
# commands are executed.

# composer_version: "2"
# if composer_version:"2" it will use the most recent composer v2
# It can also be set to "1", to get most recent composer v1
# or "" for the default v2 created at release time.
# It can be set to any existing specific composer version.
# After first project 'ddev start' this will not be updated until it changes

# nodejs_version: "16"
# change from the default system Node.js version to another supported version, like 12, 14, 17, 18.
# Note that you can use 'ddev nvm' or nvm inside the web container to provide nearly any
# Node.js version, including v6, etc.

# additional_hostnames:
#  - somename
#  - someothername
# would provide http and https URLs for "somename.ddev.site"
# and "someothername.ddev.site".

# additional_fqdns:
#  - example.com
#  - sub1.example.com
# would provide http and https URLs for "example.com" and "sub1.example.com"
# Please take care with this because it can cause great confusion.

# upload_dir: custom/upload/dir
# would set the destination path for ddev import-files to <docroot>/custom/upload/dir

# working_dir:
#   web: /var/www/html
#   db: /home
# would set the default working directory for the web and db services.
# These values specify the destination directory for ddev ssh and the
# directory in which commands passed into ddev exec are run.

# omit_containers: [db, dba, ddev-ssh-agent]
# Currently only these containers are supported. Some containers can also be
# omitted globally in the ~/.ddev/global_config.yaml. Note that if you omit
# the "db" container, several standard features of ddev that access the
# database container will be unusable. In the global configuration it is also
# possible to omit ddev-router, but not here.

# nfs_mount_enabled: false
# Great performance improvement but requires host configuration first.
# See https://ddev.readthedocs.io/en/stable/users/performance/#using-nfs-to-mount-the-project-into-the-container

# mutagen_enabled: false
# Experimental performance improvement using mutagen asynchronous updates.
# See https://ddev.readthedocs.io/en/latest/users/performance/#using-mutagen

# fail_on_hook_fail: False
# Decide whether 'ddev start' should be interrupted by a failing hook

# host_https_port: "59002"
# The host port binding for https can be explicitly specified. It is
# dynamic unless otherwise specified.
# This is not used by most people, most people use the *router* instead
# of the localhost port.

# host_webserver_port: "59001"
# The host port binding for the ddev-webserver can be explicitly specified. It is
# dynamic unless otherwise specified.
# This is not used by most people, most people use the *router* instead
# of the localhost port.

# host_db_port: "59002"
# The host port binding for the ddev-dbserver can be explicitly specified. It is dynamic
# unless explicitly specified.

# phpmyadmin_port: "8036"
# phpmyadmin_https_port: "8037"
# The PHPMyAdmin ports can be changed from the default 8036 and 8037

# host_phpmyadmin_port: "8036"
# The phpmyadmin (dba) port is not normally bound on the host at all, instead being routed
# through ddev-router, but it can be specified and bound.

# mailhog_port: "8025"
# mailhog_https_port: "8026"
# The MailHog ports can be changed from the default 8025 and 8026

# host_mailhog_port: "8025"
# The mailhog port is not normally bound on the host at all, instead being routed
# through ddev-router, but it can be bound directly to localhost if specified here.

# webimage_extra_packages: [php7.4-tidy, php-bcmath]
# Extra Debian packages that are needed in the webimage can be added here

# dbimage_extra_packages: [telnet,netcat]
# Extra Debian packages that are needed in the dbimage can be added here

# use_dns_when_possible: true
# If the host has internet access and the domain configured can
# successfully be looked up, DNS will be used for hostname resolution
# instead of editing /etc/hosts
# Defaults to true

# project_tld: ddev.site
# The top-level domain used for project URLs
# The default "ddev.site" allows DNS lookup via a wildcard
# If you prefer you can change this to "ddev.local" to preserve
# pre-v1.9 behavior.

# ngrok_args: --basic-auth username:pass1234
# Provide extra flags to the "ngrok http" command, see
# https://ngrok.com/docs#http or run "ngrok http -h"

# disable_settings_management: false
# If true, ddev will not create CMS-specific settings files like
# Drupal's settings.php/settings.ddev.php or TYPO3's AdditionalConfiguration.php
# In this case the user must provide all such settings.

# You can inject environment variables into the web container with:
# web_environment:
# - SOMEENV=somevalue
# - SOMEOTHERENV=someothervalue

# no_project_mount: false
# (Experimental) If true, ddev will not mount the project into the web container;
# the user is responsible for mounting it manually or via a script.
# This is to enable experimentation with alternate file mounting strategies.
# For advanced users only!

# bind_all_interfaces: false
# If true, host ports will be bound on all network interfaces,
# not just the localhost interface. This means that ports
# will be available on the local network if the host firewall
# allows it.

# Many ddev commands can be extended to run tasks before or after the
# ddev command is executed, for example "post-start", "post-import-db",
# "pre-composer", "post-composer"
# See https://ddev.readthedocs.io/en/stable/users/extending-commands/ for more
# information on the commands that can be extended and the tasks you can define
# for them. Example:
#hooks:
`

// SequelproTemplate is the template for Sequelpro config.
var SequelproTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>ContentFilters</key>
    <dict/>
    <key>auto_connect</key>
    <true/>
    <key>data</key>
    <dict>
        <key>connection</key>
        <dict>
            <key>database</key>
            <string>%s</string>
            <key>host</key>
            <string>%s</string>
            <key>name</key>
            <string>drud/%s</string>
            <key>password</key>
            <string>%s</string>
            <key>port</key>
            <integer>%s</integer>
            <key>rdbms_type</key>
            <string>mysql</string>
            <key>sslCACertFileLocation</key>
            <string></string>
            <key>sslCACertFileLocationEnabled</key>
            <integer>0</integer>
            <key>sslCertificateFileLocation</key>
            <string></string>
            <key>sslCertificateFileLocationEnabled</key>
            <integer>0</integer>
            <key>sslKeyFileLocation</key>
            <string></string>
            <key>sslKeyFileLocationEnabled</key>
            <integer>0</integer>
            <key>type</key>
            <string>SPTCPIPConnection</string>
            <key>useSSL</key>
            <integer>0</integer>
            <key>user</key>
            <string>%s</string>
        </dict>
    </dict>
    <key>encrypted</key>
    <false/>
    <key>format</key>
    <string>connection</string>
    <key>queryFavorites</key>
    <array/>
    <key>queryHistory</key>
    <array/>
    <key>rdbms_type</key>
    <string>mysql</string>
    <key>rdbms_version</key>
    <string>5.5.44</string>
    <key>version</key>
    <integer>1</integer>
</dict>
</plist>`
