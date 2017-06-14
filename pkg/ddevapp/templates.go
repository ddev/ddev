package ddevapp

// DDevComposeTemplate is used to create the main docker-compose.yaml
// file for a ddev site.
const DDevComposeTemplate = `version: '3'

services:
  db:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-db
    image: $DDEV_DBIMAGE
    volumes:
      - "./data:/db"
    restart: always
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
      com.ddev.app-url: $DDEV_URL
  dba:
    container_name: ddev-${DDEV_SITENAME}-dba
    image: $DDEV_DBAIMAGE
    restart: always
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
