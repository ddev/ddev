#!/bin/bash
set -eu -o pipefail

rm -f /tmp/healthy

export HTTP2=http2
if [ "${DISABLE_HTTP2:-}" = "true" ]; then HTTP2=""; fi

# Warn if the DOCKER_HOST socket does not exist
if [[ $DOCKER_HOST = unix://* ]]; then
	socket_file=${DOCKER_HOST#unix://}
	if ! [ -S $socket_file ]; then
		cat >&2 <<-EOT
			ERROR: you need to share your Docker host socket with a volume at $socket_file
			Typically you should run this container with: \`-v /var/run/docker.sock:$socket_file:ro\`
			See the documentation at http://git.io/vZaGJ
		EOT
		socketMissing=1
	fi
fi

# Compute the DNS resolvers for use in the templates - if the IP contains ":", it's IPv6 and must be enclosed in []
export RESOLVERS=$(awk '$1 == "nameserver" {print ($2 ~ ":")? "["$2"]": $2}' ORS=' ' /etc/resolv.conf | sed 's/ *$//g')
if [ "x$RESOLVERS" = "x" ]; then
    echo "Warning: unable to determine DNS resolvers for nginx" >&2
    unset RESOLVERS
fi

# If the user has run the default command and the socket doesn't exist, fail
if [ "${socketMissing:-}" = 1 -a "$1" = forego -a "$2" = start -a "$3" = '-r' ]; then
	exit 1
fi

if [ ! -z "${USE_LETSENCRYPT:-}" ]; then
  echo "Lets Encrypt is enabled:"
  certbot certificates
fi

mkcert -install

# It's unknown what docker event causes an attempt to use these certs/.crt files, but they might as well exist
# to prevent it.
mkcert -cert-file /etc/nginx/certs/.crt -key-file /etc/nginx/certs/.key "*.ddev.local" "*.ddev.site" 127.0.0.1 localhost ddev-router ddev-router.ddev_default

if [ ! -f /etc/nginx/certs/master.crt ]; then
  mkcert -cert-file /etc/nginx/certs/master.crt -key-file /etc/nginx/certs/master.key "*.ddev.local" "*.ddev.site" "ddev-router" "ddev-router.ddev_default" 127.0.0.1 localhost
fi
exec "$@"
