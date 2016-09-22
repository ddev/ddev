package utils

const legacyComposeTemplate = `version: '2'
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