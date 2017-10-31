package ddevapp

// DDevComposeTemplate is used to create the main docker-compose.yaml
// file for a ddev site.
const DDevComposeTemplate = `version: '3'

services:
  db:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-db
    image: $DDEV_DBIMAGE
    volumes:
      - "${DDEV_IMPORTDIR}:/db"
      - "${DDEV_DATADIR}:/var/lib/mysql"
    restart: always
    ports:
      - "3306"
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
    environment:
      - DDEV_UID=$DDEV_UID
      - DDEV_GID=$DDEV_GID
  web:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-web
    image: $DDEV_WEBIMAGE
    volumes:
      - "../:/var/www/html:cached"
    restart: always
    depends_on:
      - db
    links:
      - db:db
    ports:
      - "80"
      - "{{ .mailhogport }}"
    working_dir: /var/www/html/${DDEV_DOCROOT}
    environment:
      - DDEV_UID=$DDEV_UID
      - DDEV_GID=$DDEV_GID
      - DDEV_URL=$DDEV_URL
      - DOCROOT=$DDEV_DOCROOT
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      # HTTP_EXPOSE allows for ports accepting HTTP traffic to be accessible from <site>.ddev.local:<port>
      # To expose a container port to a different host port, define the port as hostPort:containerPort
      - HTTP_EXPOSE=80,{{ .mailhogport }}
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.hostname: $DDEV_HOSTNAME
  dba:
    container_name: ddev-${DDEV_SITENAME}-dba
    image: $DDEV_DBAIMAGE
    restart: always
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.approot: $DDEV_APPROOT
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

// HookTemplate is used to add example hooks usage
const HookTemplate = `
# Certain ddev commands can be extended to run tasks before or after the ddev
# command is executed.
# See https://ddev.readthedocs.io/en/latest/users/extending-commands/ for more
# information on the commands that can be extended and the tasks you can define
# for them.
# hooks:
#   post-import-db:`

// Drupal8Hooks adds a d8-specific hooks example for post-import-db
const Drupal8Hooks = `
#     - exec: "drush cr"`

// Drupal7Hooks adds a d7-specific hooks example for post-import-db
const Drupal7Hooks = `
#     - exec: "drush cc all"`

// WordPressHooks adds a wp-specific hooks example for post-import-db
const WordPressHooks = `
    # Un-comment and enter the production url and local url
    # to replace in your database after import.
    # - exec: "wp search-replace <production-url> <local-url>"`
