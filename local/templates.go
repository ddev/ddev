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
