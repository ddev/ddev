#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

DOCKER_IMAGE=$(awk '{print $1}' .docker_image)
CONTAINER_NAME=ddev-router-test

# Wait for container to be ready.
function containercheck {
    set +x
	for i in {20..0};
	do
		# status contains uptime and health in parenthesis, sed to return health
		status="$(docker ps --format "{{.Status}}" --filter "name=$CONTAINER_NAME" | sed  's/.*(\(.*\)).*/\1/')"
		if [[ "$status" == "healthy" ]]
		then
			set -x
			return 0
		fi
		sleep 1
	done
	return 1
}

function cleanup {
	echo "Removing $CONTAINER_NAME"
	docker rm -f $CONTAINER_NAME 2>/dev/null || true
}
trap cleanup EXIT

cleanup

# Make sure rootCA is created and installed on the ddev-global-cache/mkcert
mkcert -install
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache  -v //var/run/docker.sock:/tmp/docker.sock:ro $DOCKER_IMAGE bash -c "mkdir -p /mnt/ddev-global-cache/mkcert && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache"

# Run the router alone
docker run --rm --name $CONTAINER_NAME -p 8080:80 -p 8443:443 -v //var/run/docker.sock:/tmp/docker.sock:ro -v ddev-global-cache:/mnt/ddev-global-cache --name ddev-router-test -d $DOCKER_IMAGE

CONTAINER_NAME=ddev-router-test

if ! containercheck; then
    echo "FAIL: ddev-router failed to become ready"
    echo "--- ddev-router FAIL: information"
    docker logs $CONTAINER_NAME
    docker ps -a
    docker inspect $CONTAINER_NAME
    exit 101
fi

# Make sure we can access http and https ports successfully (and with valid cert)
(curl -s -I http://localhost:8080 | grep 503) || (echo "Failed to get 503 from nginx-router by default" && exit 102)
(curl -s -I https://localhost:8443 | grep 503) || (echo "Failed to get 503 from nginx-router via https by default" && exit 103)
# Make sure internal access to https is working
docker exec -t $CONTAINER_NAME curl --fail https://localhost/healthcheck || (echo "Failed to run https healthcheck inside container" && exit 104)
