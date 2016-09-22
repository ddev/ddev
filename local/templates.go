package local

// LegacyComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the local DRUD env
const LegacyComposeTemplate = `version: '2'
services:
  {{.name}}-db:
    container_name: {{.name}}-db
    image: drud/mysql-docker-local:5.7
    volumes:
      - "./data:/db"
    restart: always
    environment:
      MYSQL_DATABASE: data
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306"
  {{.name}}-web:
    container_name: {{.name}}-web
    image: {{.image}}
    volumes:
      - "./src:/var/www/html"
    restart: always
    depends_on:
      - {{.name}}-db
    links:
      - {{.name}}-db:db
    ports:
      - "80"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
`

// DrudComposeTemplate is used to create docker-compose.yaml for local Drud sites
// @todo use the template engine instead of fmt
const DrudComposeTemplate = `version: '2'
services:
  %[1]s-db:
    container_name: %[1]s-db
    image: drud/mysql-docker-local:5.7
    volumes:
      - "./data:/db"
    restart: always
    environment:
      MYSQL_DATABASE: data
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306"
  %[1]s-web:
    container_name: %[1]s-web
    image: %[2]s
    volumes:
      - "./src:/var/www/html"
    restart: always
    depends_on:
      - %[1]s-db
    links:
      - %[1]s-db:db
    ports:
      - "80"
    working_dir: "/var/www/html/docroot"
    environment:
      - DEPLOY_NAME=local
`
