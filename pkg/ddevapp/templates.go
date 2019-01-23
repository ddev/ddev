package ddevapp

// DDevComposeTemplate is used to create the main docker-compose.yaml
// file for a ddev site.
const DDevComposeTemplate = `version: '{{ .ComposeVersion }}'
{{ .DdevGenerated }}
services:
  db:
    container_name: {{ .Plugin }}-${DDEV_SITENAME}-db
    image: $DDEV_DBIMAGE
    stop_grace_period: 60s
    volumes:
      - type: "volume"
        source: mariadb-database
        target: "/var/lib/mysql"
        volume:
          nocopy: true
      - type: "bind"
        source: "${DDEV_IMPORTDIR}"
        target: "/db"
      - type: "bind"
        source: "."
        target: "/mnt/ddev_config"
    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    ports:
      - "3306"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .Plugin }}
      com.ddev.app-type: {{ .AppType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
    environment:
      - COLUMNS=$COLUMNS
      - LINES=$LINES
    command: "$DDEV_MARIADB_LOCAL_COMMAND"
    healthcheck:
      interval: 5s
      retries: 4
      start_period: 20s
  web:
    container_name: {{ .Plugin }}-${DDEV_SITENAME}-web
    image: $DDEV_WEBIMAGE
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
      - ddev-composer-cache:/mnt/composer_cache
      {{ if not .OmitSSHAgent }}
      - ddev-ssh-agent_socket_dir:/home/.ssh-agent
      {{ end }}

    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    links:
      - db:db
    # ports is list of exposed *container* ports
    ports:
      - "80"
      - "{{ .MailhogPort }}"
    environment:
      - DDEV_URL=$DDEV_URL
      - DOCROOT=$DDEV_DOCROOT
      - DDEV_PHP_VERSION=$DDEV_PHP_VERSION
      - DDEV_WEBSERVER_TYPE=$DDEV_WEBSERVER_TYPE
      - DDEV_PROJECT_TYPE=$DDEV_PROJECT_TYPE
      - DDEV_ROUTER_HTTP_PORT=$DDEV_ROUTER_HTTP_PORT
      - DDEV_ROUTER_HTTPS_PORT=$DDEV_ROUTER_HTTPS_PORT
      - DDEV_XDEBUG_ENABLED=$DDEV_XDEBUG_ENABLED
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      - COLUMNS=$COLUMNS
      - LINES=$LINES
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.local:<port>
      # To expose a container port to a different host port, define the port as hostPort:containerPort
      - HTTP_EXPOSE=${DDEV_ROUTER_HTTP_PORT}:80,{{ .MailhogPort }}
      # You can optionally expose an HTTPS port option for any ports defined in HTTP_EXPOSE.
      # To expose an HTTPS port, define the port as securePort:containerPort.
      - HTTPS_EXPOSE=${DDEV_ROUTER_HTTPS_PORT}:80
      - SSH_AUTH_SOCK=/home/.ssh-agent/socket
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .Plugin }}
      com.ddev.app-type: {{ .AppType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
{{ if .HostDockerInternalIP }}
    extra_hosts: [ "{{ .HostDockerInternalHostname }}:{{ .HostDockerInternalIP }}" ]
{{ end }}
    external_links:
      - ddev-router:$DDEV_HOSTNAME
    healthcheck:
      interval: 4s
      retries: 6
      start_period: 10s
{{ if .WebcacheEnabled }}
  bgsync:
    container_name: ddev-${DDEV_SITENAME}-bgsync
    image: $DDEV_BGSYNCIMAGE
    restart: "on-failure"
    user: "$DDEV_UID:$DDEV_GID"
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
      com.ddev.app-url: $DDEV_URL
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
      com.ddev.app-url: $DDEV_URL
    links:
      - db:db
    ports:
      - "80"
    environment:
      - PMA_USER=db
      - PMA_PASSWORD=db
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.local:<port>
      - HTTP_EXPOSE={{ .DBAPort }}
    healthcheck:
      interval: 90s
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
  ddev-composer-cache:
    name: ddev-composer-cache
  {{ if eq .MountType "volume" }}
  webcachevol:
  unisoncatalogvol:
  {{ end }}
`

// ConfigInstructions is used to add example hooks usage
const ConfigInstructions = `
# Key features of ddev's config.yaml:

# name: <projectname> # Name of the project, automatically provides
#   http://projectname.ddev.local and https://projectname.ddev.local

# type: <projecttype>  # drupal6/7/8, backdrop, typo3, wordpress, php

# docroot: <relative_path> # Relative path to the directory containing index.php.

# php_version: "7.1"  # PHP version to use, "5.6", "7.0", "7.1", "7.2", "7.3"

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

# webserver_type: nginx-fpm  # Can be set to apache-fpm or apache-cgi as well

# additional_hostnames:
#  - somename
#  - someothername
# would provide http and https URLs for "somename.ddev.local"
# and "someothername.ddev.local".

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


# provider: default # Currently either "default" or "pantheon"
#
# Many ddev commands can be extended to run tasks after the ddev command is
# executed.
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
      {{ range $port := .ports }}- "{{ $port }}:{{ $port }}"
      {{ end }}
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - type: "volume"
        source: ddev-router-cert-cache
        target: "/etc/nginx/certs"
        volume:
          nocopy: true
    restart: "no"
    healthcheck:
      interval: 5s
      retries: 3
      start_period: 10s

networks:
   default:
     external:
       name: ddev_default
volumes:
  ddev-router-cert-cache:
    name: "ddev-router-cert-cache"
`

const DdevSSHAuthTemplate = `version: '{{ .compose_version }}'

volumes:
  dot_ssh:
  socket_dir:

services:
  ddev-ssh-agent:
    container_name: ddev-ssh-agent
    image: {{ .ssh_auth_image }}:{{ .ssh_auth_tag }}
    user: "$DDEV_UID:$DDEV_GID"
    volumes:
      - "dot_ssh:/tmp/.ssh"
      - "socket_dir:/tmp/.ssh-agent"
    environment:
      - SSH_AUTH_SOCK=/tmp/.ssh-agent/socket
    healthcheck:
      interval: 2s
      retries: 5
networks:
  default:
    external:
      name: ddev_default
`
