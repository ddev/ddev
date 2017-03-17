package ddevapp

// DDevComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the ddev env

const DDevComposeTemplate = `version: '2'
services:
  {{ .plugin }}-{{.name }}-db:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-db
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
      com.ddev.site-name: ${DDEV_SITENAME}
      com.ddev.container-type: web
      com.ddev.app-type: {{ .appType }}
      com.ddev.docroot: $DDEV_DOCROOT
      com.ddev.approot: $DDEV_APPROOT
      com.ddev.app-url: $DDEV_URL
  {{ .plugin }}-{{ .name }}-web:
    container_name: {{ .plugin }}-${DDEV_SITENAME}-web
    image: $DDEV_WEBIMAGE
    volumes:
      - "{{ .docroot }}/:/var/www/html/docroot"
    restart: always
    depends_on:
      - {{ .plugin }}-${DDEV_SITENAME}-db
    links:
      - {{ .plugin }}-${DDEV_SITENAME}-db:db
    ports:
      - "80"
      - "8025"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
      - VIRTUAL_HOST=$DDEV_HOSTNAME
    labels:
      com.ddev.site-name: ${DDEV_SITENAME}
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
