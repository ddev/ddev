Scripts in this directory will be executed inside the db
container. A number of environment variables are supplied to the scripts, including:

DDEV_APPROOT: file system location of the project on the host)
DDEV_HOST_DB_PORT: Localhost port of the database server
DDEV_HOST_WEBSERVER_PORT: Localhost port of the webserver
DDEV_HOST_HTTPS_PORT: Localhost port for https on webserver
DDEV_DOCROOT: Relative path from approot to docroot
DDEV_HOSTNAME: Comma-separated list of FQDN hostnames
DDEV_PHP_VERSION
DDEV_WEBSERVER_TYPE: nginx-fpm, apache-fpm, apache-cgi
DDEV_PROJECT_TYPE: drupal8, typo3, backdrop, wordpress, etc.
DDEV_ROUTER_HTTP_PORT: Router port for http
DDEV_ROUTER_HTTPS_PORT: Router port for https

More environment variables may be available, see https://github.com/drud/ddev/blob/52c7915dee41d4846f9f619520b726994c0372c5/pkg/ddevapp/ddevapp.go#L1006-L1030
