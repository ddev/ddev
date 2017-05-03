package ddevapp

// DDevComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the ddev env
const DDevComposeTemplate = `version: '2'

services:
  db:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-db
    image: $DDEV_DBIMAGE
    volumes:
      - "./data:/db"
    restart: always
    environment:
      - TCP_PORT=$DDEV_HOSTNAME:{{ .dbport }}
    ports:
      - 3306
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.container-type: db
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  web:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-web
    image: $DDEV_WEBIMAGE
    volumes:
      - "{{ .docroot }}/:/var/www/html/docroot"
    restart: always
    depends_on:
      - db
    links:
      - db:$DDEV_HOSTNAME
      - db:db
    ports:
      - "80"
      - {{ .mailhogport }}
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      - VIRTUAL_PORT=80,{{ .mailhogport }}
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.container-type: web
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  dba:
    container_name: local-${DDEV_SITENAME}-dba
    image: $DDEV_DBAIMAGE
    restart: always
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.container-type: dba
      com.ddev.platform: {{ .plugin }}
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
    depends_on:
      - db
    links:
      - db:db
    ports:
      - "80"
    environment:
      - PMA_USER=root
      - PMA_PASSWORD=root
      - VIRTUAL_HOST=$DDEV_HOSTNAME
      - VIRTUAL_PORT={{ .dbaport }}
networks:
  default:
    external:
      name: ddev_default
`
