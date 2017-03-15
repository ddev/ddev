package ddevapp

// DDevComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the ddev env
// @TODO: this should be updated to simplify things where possible and remove 'drud' in favor of ddev.
// This was not undertaken when moving the template into the appconfig package to reduce churn.
const DDevComposeTemplate = `version: '2'
services:
  {{.name}}-db:
    container_name: {{.name}}-db
    image: $DRUD_DBIMAGE
    volumes:
      - "./data:/db"
    restart: always
    environment:
      MYSQL_DATABASE: data
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306"
    labels:
      com.drud.site-name: {{ .name }}
      com.drud.container-type: web
  {{.name}}-web:
    container_name: {{.name}}-web
    image: $DRUD_WEBIMAGE
    volumes:
      - "{{ .docroot }}/:/var/www/html/docroot"
    restart: always
    depends_on:
      - {{.name}}-db
    links:
      - {{.name}}-db:db
    ports:
      - "80"
      - "8025"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
      - VIRTUAL_HOST={{ .app_url }}
    labels:
      com.drud.site-name: {{ .name }}
      com.drud.container-type: db

networks:
  default:
    external:
      name: drud_default
`
