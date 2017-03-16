package ddevapp

// DDevComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the ddev env
// @TODO: this should be updated to simplify things where possible and remove 'ddev' in favor of ddev.
// This was not undertaken when moving the template into the appconfig package to reduce churn.
const DDevComposeTemplate = `version: '2'
services:
  {{ .plugin }}-{{ .name }}-db:
    container_name: {{ .plugin }}-{{ .name }}-db
    image: $DDEV_DBIMAGE
    volumes:
      - "./data:/db"
    restart: always
    environment:
      MYSQL_DATABASE: data
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306"
    labels:
      com.ddev.site-name: {{ .name }}
      com.ddev.container-type: web
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  {{ .plugin }}-{{ .name }}-web:
    container_name: {{ .plugin }}-{{ .name }}-web
    image: $DDEV_WEBIMAGE
    volumes:
      - "{{ .docroot }}/:/var/www/html/docroot"
    restart: always
    depends_on:
      - {{ .plugin }}-{{ .name }}-db
    links:
      - {{ .plugin }}-{{ .name }}-db:db
    ports:
      - "80"
      - "8025"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
    labels:
      com.ddev.site-name: {{ .name }}
      com.ddev.container-type: db
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
networks:
  default:
    external:
      name: ddev_default
`
