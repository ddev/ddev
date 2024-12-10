#ddev-generated
Custom scripts (named *.sh) in this directory will be run during web container startup,
before the php-fpm server or other daemons are run.

This can be useful, for example, for introducing environment variables into the context of the nginx and php-fpm servers.
Use this carefully, because custom entrypoints can very easily break things.
