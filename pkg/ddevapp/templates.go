package ddevapp

// DDevComposeTemplate is used to create the main docker-compose.yaml
// file for a ddev site.
const DDevComposeTemplate = `version: '3'

services:
  db:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-db
    image: $DDEV_DBIMAGE
    stop_grace_period: 60s
    volumes:
      - "${DDEV_IMPORTDIR}:/db"
      - "${DDEV_DATADIR}:/var/lib/mysql"
      - ".:/mnt/ddev_config"
    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    ports:
      - "3306"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  web:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-web
    image: $DDEV_WEBIMAGE
    volumes:
      - "../:/var/www/html:cached"
      - ".:/mnt/ddev_config"
    restart: "no"
    user: "$DDEV_UID:$DDEV_GID"
    depends_on:
      - db
    links:
      - db:db
    # ports is list of exposed *container* ports
    ports:
      - "80"
      - "{{ .mailhogport }}"
    working_dir: /var/www/html/${DDEV_DOCROOT}
    environment:
      - DDEV_URL=$DDEV_URL
      - DOCROOT=$DDEV_DOCROOT
      - DDEV_PHP_VERSION=$DDEV_PHP_VERSION
      - DDEV_PROJECT_TYPE=$DDEV_PROJECT_TYPE
      - DDEV_ROUTER_HTTP_PORT=$DDEV_ROUTER_HTTP_PORT
      - DDEV_ROUTER_HTTPS_PORT=$DDEV_ROUTER_HTTPS_PORT
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.local:<port>
      # To expose a container port to a different host port, define the port as hostPort:containerPort
      - HTTP_EXPOSE=${DDEV_ROUTER_HTTP_PORT}:80,{{ .mailhogport }}
      # You can optionally expose an HTTPS port option for any ports defined in HTTP_EXPOSE.
      # To expose an HTTPS port, define the port as securePort:containerPort.
      - HTTPS_EXPOSE=${DDEV_ROUTER_HTTPS_PORT}:80
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  dba:
    container_name: ddev-${DDEV_SITENAME}-dba
    image: $DDEV_DBAIMAGE
    restart: "no"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
    depends_on:
      - db
    links:
      - db:db
    ports:
      - "80"
    environment:
      - PMA_USER=db
      - PMA_PASSWORD=db
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.local:<port>
      - HTTP_EXPOSE={{ .dbaport }}
networks:
  default:
    external:
      name: ddev_default
`

// ConfigInstructions is used to add example hooks usage
const ConfigInstructions = `
# Key features of ddev's config.yaml:

# name: <projectname> # Name of the project, automatically provides 
#   http://projectname.ddev.local and https://projectname.ddev.local

# type: <projecttype>  # drupal6/7/8, backdrop, typo3, wordpress, php

# docroot: <relative_path> # Relative path to the directory containing index.php.

# php_version: "7.1"  # PHP version to use, "5.6", "7.0", "7.1", "7.2"

# You can delete the webimage, dbimage, dbaimage lines to use the defaults
# in ddev.

# webimage: <docker_image>  # nginx/php docker image. 
# dbimage: <docker_image>  # mariadb docker image. 
# dbaimage: <docker_image>

# router_http_port: <port>  # Port to be used for http (defaults to port 80)
# router_https_port: <port> # Port for https (defaults to 443)

#additional_hostnames:
# - somename
# - someothername
# would provide http and https URLs for "somename.ddev.local"
# and "someothername.ddev.local".

# provider: default # Currently either "default" or "pantheon"
#
# Many ddev commands can be extended to run tasks after the ddev command is
# executed.
# See https://ddev.readthedocs.io/en/latest/users/extending-commands/ for more
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
const DdevRouterTemplate = `version: '3'
services:
  ddev-router:
    image: {{ .router_image }}:{{ .router_tag }}
    container_name: ddev-router
    ports:
      {{ range $port := .ports }}- "{{ $port }}:{{ $port }}"
      {{ end }}
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
      - ./certs:/etc/nginx/certs:cached
    restart: "no"
networks:
   default:
     external:
       name: ddev_default
`
