Scripts in this directory will be executed inside the solr
container (if it exists, of course). This is just an example,
but any named service can have a directory with commands.

Note that /mnt/ddev_config must be mounted into the 3rd-party service
with a stanza like this in the docker-compose.solr.yaml:

    volumes:
      - type: "bind"
        source: "."
        target: "/mnt/ddev_config"


A number of environment variables are supplied to the scripts, including:

DDEV_DOCROOT: Relative path from approot to docroot
DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
DDEV_PHP_VERSION
DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm, apache-cgi
DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
DDEV_ROUTER_HTTP_PORT: Router port for http
DDEV_ROUTER_HTTPS_PORT: Router port for https

More environment variables may be available, see https://github.com/drud/ddev/blob/52c7915dee41d4846f9f619520b726994c0372c5/pkg/ddevapp/ddevapp.go#L1006-L1030
