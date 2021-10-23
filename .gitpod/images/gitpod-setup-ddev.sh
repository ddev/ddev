#!/usr/bin/env bash

# Set up ddev for use on gitpod

set -eu -o pipefail

if [ $# != 1 ]; then
  echo "Usage: gitpod-setup-ddev.sh <project-path>" && exit 1
fi

PROJDIR="$1/.ddev"
cd ${PROJDIR}

# Generate a config.gitpod.yaml that adds the gitpod
# proxied ports so they're known to ddev.
shortgpurl="${GITPOD_WORKSPACE_URL#'https://'}"

cat <<CONFIGEND > ${PROJDIR}/config.gitpod.yaml
#ddev-gitpod-generated
use_dns_when_possible: false

# Throwaway ports, otherwise Gitpod throw an error 'port needs to be > 1024'
router_http_port: "8888"
router_https_port: "8889"

additional_fqdns:
- 8888-${shortgpurl}
- 8025-${shortgpurl}
- 8036-${shortgpurl}
CONFIGEND

# We need host.docker.internal inside the container,
# So add it via docker-compose.host-docker-internal.yaml
hostip=$(awk "\$2 == \"$HOSTNAME\" { print \$1; }" /etc/hosts)

cat <<COMPOSEEND >${PROJDIR}/docker-compose.host-docker-internal.yaml
#ddev-gitpod-generated
version: "3.6"
services:
  web:
    extra_hosts:
    - "host.docker.internal:${hostip}"
    # This adds 8080 on the host (bound on all interfaces)
    # It goes directly to the web container without
    # ddev-nginx
    ports:
    - 8080:80
COMPOSEEND
