package ddevapp

// DDevComposeTemplate is used to create the main docker-compose.yaml
// file for a ddev site.
const DDevComposeTemplate = `version: '{{ .ComposeVersion }}'
{{ .DdevGenerated }}
services:
  db:
    container_name: {{ .Plugin }}-${DDEV_SITENAME}-db
    build: 
      context: '{{ .DBBuildContext }}'
      args: 
        BASE_IMAGE: $DDEV_DBIMAGE
        username: '{{ .Username }}'
        uid: '{{ .UID }}'
        gid: '{{ .GID }}'
    image: ${DDEV_DBIMAGE}-built
    stop_grace_period: 60s
    volumes:
      - type: "volume"
        source: mariadb-database
        target: "/var/lib/mysql"
        volume:
          nocopy: true
      - type: "bind"
        source: "."
        target: "/mnt/ddev_config"
      - ddev-global-cache:/mnt/ddev-global-cache
    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    hostname: {{ .Name }}-db
    ports:
      - "{{ .DockerIP }}:$DDEV_HOST_DB_PORT:3306"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .Plugin }}
      com.ddev.app-type: {{ .AppType }}
      com.ddev.approot: $DDEV_APPROOT
    environment:
      - COLUMNS=$COLUMNS
      - LINES=$LINES
      - TZ={{ .Timezone }}
      - DDEV_PROJECT={{ .Name }}
    command: "$DDEV_MARIADB_LOCAL_COMMAND"
    healthcheck:
      interval: 1s
      retries: 30
      start_period: 20s
      timeout: 120s
  web:
    container_name: {{ .Plugin }}-${DDEV_SITENAME}-web
    build: 
      context: '{{ .WebBuildContext }}'
      args: 
        BASE_IMAGE: $DDEV_WEBIMAGE
        username: '{{ .Username }}'
        uid: '{{ .UID }}'
        gid: '{{ .GID }}'
    image: ${DDEV_WEBIMAGE}-built
    cap_add:
      - SYS_PTRACE
    volumes:
      - type: {{ .MountType }}
        source: {{ .WebMount }}
        target: /var/www/html
        {{ if eq .MountType "volume" }}
        volume:
          nocopy: true
        {{ else }}
        consistency: cached
        {{ end }}
      - ".:/mnt/ddev_config:ro"
      - ddev-global-cache:/mnt/ddev-global-cache
      {{ if not .OmitSSHAgent }}
      - ddev-ssh-agent_socket_dir:/home/.ssh-agent
      {{ end }}

    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    hostname: {{ .Name }}-web
    links:
      - db:db
    # ports is list of exposed *container* ports
    ports:
      - "{{ .DockerIP }}:$DDEV_HOST_WEBSERVER_PORT:80"
      - "{{ .DockerIP }}:$DDEV_HOST_HTTPS_PORT:443"
    environment:
      - DOCROOT=$DDEV_DOCROOT
      - DDEV_PHP_VERSION=$DDEV_PHP_VERSION
      - DDEV_WEBSERVER_TYPE=$DDEV_WEBSERVER_TYPE
      - DDEV_PROJECT_TYPE=$DDEV_PROJECT_TYPE
      - DDEV_ROUTER_HTTP_PORT=$DDEV_ROUTER_HTTP_PORT
      - DDEV_ROUTER_HTTPS_PORT=$DDEV_ROUTER_HTTPS_PORT
      - DDEV_XDEBUG_ENABLED=$DDEV_XDEBUG_ENABLED
      - DOCKER_IP={{ .DockerIP }}
      - HOST_DOCKER_INTERNAL_IP={{ .HostDockerInternalIP }}
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      - COLUMNS=$COLUMNS
      - LINES=$LINES
      - TZ={{ .Timezone }}
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.site:<port>
      # To expose a container port to a different host port, define the port as hostPort:containerPort
      - HTTP_EXPOSE=${DDEV_ROUTER_HTTP_PORT}:80,${DDEV_MAILHOG_PORT}:{{ .MailhogPort }}
      # You can optionally expose an HTTPS port option for any ports defined in HTTP_EXPOSE.
      # To expose an HTTPS port, define the port as securePort:containerPort.
      - HTTPS_EXPOSE=${DDEV_ROUTER_HTTPS_PORT}:80
      - SSH_AUTH_SOCK=/home/.ssh-agent/socket
      - DDEV_PROJECT={{ .Name }}
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .Plugin }}
      com.ddev.app-type: {{ .AppType }}
      com.ddev.approot: $DDEV_APPROOT
{{ if .HostDockerInternalIP }}
    extra_hosts: [ "host.docker.internal:{{ .HostDockerInternalIP }}" ]
{{ end }}
    external_links:
    {{ range $hostname := .Hostnames }}- "ddev-router:{{ $hostname }}" 
    {{ end }}
    healthcheck:
      interval: 1s
      retries: 10
      start_period: 10s
      timeout: 120s
{{ if .WebcacheEnabled }}
  bgsync:
    container_name: ddev-${DDEV_SITENAME}-bgsync
    build: 
      context: '{{ .BgsyncBuildContext }}'
      args: 
        BASE_IMAGE: $DDEV_BGSYNCIMAGE
        username: '{{ .Username }}'
        uid: '{{ .UID }}'
        gid: '{{ .GID }}'
    image: ${DDEV_BGSYNCIMAGE}-built
    restart: "on-failure"
    user: "$DDEV_UID:$DDEV_GID"
    hostname: {{ .Name }}-bgsync
    volumes:
      - ..:/hostmount:cached
      - webcachevol:/fastdockermount
      - unisoncatalogvol:/root/.unison

    environment:
    - SYNC_DESTINATION=/fastdockermount
    - SYNC_SOURCE=/hostmount
    - SYNC_MAX_INOTIFY_WATCHES=100000
    - SYNC_VERBOSE=1
    privileged: true
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: ddev
      com.ddev.app-type: drupal8
      com.ddev.approot: $DDEV_APPROOT
    healthcheck:
      interval: 10s
      retries: 24
      start_period: 240s

{{end}}

{{if not .OmitDBA }}
  dba:
    container_name: ddev-${DDEV_SITENAME}-dba
    image: $DDEV_DBAIMAGE
    restart: "no"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .Plugin }}
      com.ddev.app-type: {{ .AppType }}
      com.ddev.approot: $DDEV_APPROOT
    links:
      - db:db
    ports:
      - "80"
    hostname: {{ .Name }}-dba
    environment:
      - PMA_USER=db
      - PMA_PASSWORD=db
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      - TZ={{ .Timezone }}
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.site:<port>
      - HTTP_EXPOSE=${DDEV_PHPMYADMIN_PORT}:{{ .DBAPort }}
    healthcheck:
      interval: 120s
      timeout: 2s
      retries: 1

{{end}}
networks:
  default:
    external:
      name: ddev_default
volumes:
  mariadb-database:
    name: "${DDEV_SITENAME}-mariadb"
  {{ if not .OmitSSHAgent }}
  ddev-ssh-agent_socket_dir:
    external: true
  {{ end }}
  ddev-global-cache:
    name: ddev-global-cache
  {{ if .WebcacheEnabled }}
  webcachevol:
  unisoncatalogvol:
  {{ end }}
  {{ if .NFSMountEnabled }}
  nfsmount:
    driver: local
    driver_opts:
      type: nfs
      o: "addr={{ if .HostDockerInternalIP }}{{ .HostDockerInternalIP }}{{ else }}host.docker.internal{{end}},hard,nolock,rw"
      device: ":{{ .NFSSource }}"
  {{ end }}

  `

// ConfigInstructions is used to add example hooks usage
const ConfigInstructions = `
# Key features of ddev's config.yaml:

# name: <projectname> # Name of the project, automatically provides
#   http://projectname.ddev.site and https://projectname.ddev.site

# type: <projecttype>  # drupal6/7/8, backdrop, typo3, wordpress, php

# docroot: <relative_path> # Relative path to the directory containing index.php.

# php_version: "7.2"  # PHP version to use, "5.6", "7.0", "7.1", "7.2", "7.3"

# You can explicitly specify the webimage, dbimage, dbaimage lines but this
# is not recommended, as the images are often closely tied to ddev's' behavior,
# so this can break upgrades.

# webimage: <docker_image>  # nginx/php docker image.
# dbimage: <docker_image>  # mariadb docker image.
# dbaimage: <docker_image>
# bgsyncimage: <docker_image>

# router_http_port: <port>  # Port to be used for http (defaults to port 80)
# router_https_port: <port> # Port for https (defaults to 443)

# xdebug_enabled: false  # Set to true to enable xdebug and "ddev start" or "ddev restart"
# Note that for most people the commands 
# "ddev exec enable_xdebug" and "ddev exec disable_xdebug" work better,
# as leaving xdebug enabled all the time is a big performance hit.

# webserver_type: nginx-fpm  # Can be set to apache-fpm or apache-cgi as well

# timezone: Europe/Berlin
# This is the timezone used in the containers and by PHP;
# it can be set to any valid timezone, 
# see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
# For example Europe/Dublin or MST7MDT

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
# would set the destination path for ddev import-files to custom/upload/dir.

# working_dir:
#   web: /var/www/html
#   db: /home
# would set the default working directory for the web and db services. 
# These values specify the destination directory for ddev ssh and the 
# directory in which commands passed into ddev exec are run. 

# omit_containers: ["dba", "ddev-ssh-agent"]
# would omit the dba (phpMyAdmin) and ddev-ssh-agent containers. Currently
# only those two containers can be omitted here.
# Note that these containers can also be omitted globally in the 
# ~/.ddev/global_config.yaml or with the "ddev config global" command.

# nfs_mount_enabled: false
# Great performance improvement but requires host configuration first.
# See https://ddev.readthedocs.io/en/stable/users/performance/#using-nfs-to-mount-the-project-into-the-container

# webcache_enabled: false (deprecated)
# Was only for macOS, but now deprecated. 
# See https://ddev.readthedocs.io/en/stable/users/performance/#using-webcache_enabled-to-cache-the-project-directory

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

# phpmyadmin_port: "1000"
# The PHPMyAdmin port can be changed from the default 8036

# mailhog_port: "1001"
# The MailHog port can be changed from the default 8025

# webimage_extra_packages: [php-yaml, php7.3-ldap]
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

# ngrok_args: --subdomain mysite --auth username:pass
# Provide extra flags to the "ngrok http" command, see 
# https://ngrok.com/docs#http or run "ngrok http -h"

# provider: default # Currently either "default" or "pantheon"
#
# Many ddev commands can be extended to run tasks before or after the 
# ddev command is executed, for example "post-start", "post-import-db", 
# "pre-composer", "post-composer"
# See https://ddev.readthedocs.io/en/stable/users/extending-commands/ for more
# information on the commands that can be extended and the tasks you can define
# for them. Example:
#hooks:`

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

// DdevRouterTemplate is the template for the generic router container.
const DdevRouterTemplate = `version: '{{ .compose_version }}'
services:
  ddev-router:
    image: {{ .router_image }}:{{ .router_tag }}
    container_name: ddev-router
    ports:
      {{ $dockerIP := .dockerIP }}{{ range $port := .ports }}- "{{ $dockerIP }}:{{ $port }}:{{ $port }}"
      {{ end }}
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - ddev-global-cache:/mnt/ddev-global-cache:rw
    restart: "no"
    healthcheck:
      interval: 1s
      retries: 10
      start_period: 10s
      timeout: 120s

networks:
   default:
     external:
       name: ddev_default
volumes: 
   ddev-global-cache:
     name: ddev-global-cache
`

const DdevSSHAuthTemplate = `version: '{{ .compose_version }}'

volumes:
  dot_ssh:
  socket_dir:

services:
  ddev-ssh-agent:
    container_name: ddev-ssh-agent
    hostname: ddev-ssh-agent
    build: 
      context: '{{ .BuildContext }}'
      args: 
        BASE_IMAGE: {{ .ssh_auth_image }}:{{ .ssh_auth_tag }}
        username: '{{ .Username }}'
        uid: '{{ .UID }}'
        gid: '{{ .GID }}'
    image: {{ .ssh_auth_image }}:{{ .ssh_auth_tag }}-built

    user: "$DDEV_UID:$DDEV_GID"
    volumes:
      - "dot_ssh:/tmp/.ssh"
      - "socket_dir:/tmp/.ssh-agent"
    environment:
      - SSH_AUTH_SOCK=/tmp/.ssh-agent/socket
    healthcheck:
      interval: 1s
      retries: 2
      start_period: 10s
      timeout: 62s
networks:
  default:
    external:
      name: ddev_default
`
