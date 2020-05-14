#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset
set -x

# As of Docker 2.3.0.2 May 2020, the mount of /var/run/docker.sock doesn't seem to be possible any more.
if [ "${OS:-$(uname)}" = "Windows_NT" ]; then exit; fi

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
docker run -t --rm  -v "$(mkcert -CAROOT):/mnt/mkcert" -v ddev-global-cache:/mnt/ddev-global-cache busybox sh -c "mkdir -p /mnt/ddev-global-cache/mkcert && chmod -R ugo+w /mnt/ddev-global-cache/* && cp -R /mnt/mkcert /mnt/ddev-global-cache"

# Run the router alone
docker run --rm --name $CONTAINER_NAME -p 8080:80 -p 8443:443 --mount "type=bind,src=/var/run/docker.sock,target=/tmp/docker.sock" -v ddev-global-cache:/mnt/ddev-global-cache --name ddev-router-test -d $DOCKER_IMAGE

CONTAINER_NAME=ddev-router-test

if ! containercheck; then
    printf "=============== FAIL: $CONTAINER_NAME failed to become ready ====================\n"
    printf "=============== FAIL: $CONTAINER_NAME FAIL: information =================\n"
    docker logs $CONTAINER_NAME
    docker ps -a
    docker inspect $CONTAINER_NAME
    exit 101
fi

# Make sure we can access http and https ports successfully (and with valid cert)
(curl -s -I http://127.0.0.1:8080 | grep 503) || (echo "Failed to get 503 from nginx-router by default" && exit 102)
# mkcert is not respected by git-bash curl, so don't try the test on windows.
if [ "${OS:-$(uname)}" != "Windows_NT" ]; then
    (curl -s -I https://127.0.0.1:8443 | grep 503) || (echo "Failed to get 503 from nginx-router via https by default" && exit 103)
fi
# Make sure internal access to https is working
docker exec -t $CONTAINER_NAME curl --fail https://localhost/healthcheck || (echo "Failed to run https healthcheck inside container" && exit 104)
