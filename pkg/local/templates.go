package local

// LegacyComposeTemplate is used to create the docker-compose.yaml for
// legacy sites in the local DRUD env
const LegacyComposeTemplate = `version: '2'
services:
  {{.name}}-db:
    container_name: {{.name}}-db
    image: {{.db_image}}
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
    image: {{.web_image}}
    volumes:
      - "./src:{{ .srctarget }}"
      - "./files:/files"
    restart: always
    depends_on:
      - {{.name}}-db
    links:
      - {{.name}}-db:db
    ports:
      - "80"
      - "8025"
    working_dir: "/var/www/html"
    environment:
      - DEPLOY_NAME=local
      - DB_HOST=db
      - DEPLOY_URL={{ .deploy_url }}
      - VIRTUAL_HOST={{ .name }}

networks:
  default:
    external:
      name: drud_default
`

// SequelproTemplate is used to create the config file for Sequel Pro
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

// DrudRouterTemplate is used to create the docker compose file for the router
const DrudRouterTemplate = `version: '2'
services:
  nginx-proxy:
    image: {{ .router_image }}:{{ .router_tag }}
    container_name: nginx-proxy
    ports:
      - "80:80"
      - "8025:8025"
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock:ro
networks:
   default:
     external:
       name: drud_default
`
