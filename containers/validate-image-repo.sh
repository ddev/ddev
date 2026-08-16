#!/usr/bin/env bash
# validate-image-repo.sh <image-repo>
#
# Validates a `<org>/<name>` repository string before it's used in any `docker
# push`/`docker buildx imagetools create` command. The companion to
# validate-image-tag.sh: image-push.yml reads the repository list out of a
# build artifact produced by a job that may have run untrusted (fork PR)
# content, so without this a fork could name any repository the push
# credential can write to.
#
# Env:
#   DOCKER_ORG - the only organization a push is allowed to target (required)

set -eu -o pipefail

ALLOWED_SUFFIXES=(
  ddev-webserver
  ddev-webserver-prod
  ddev-traefik-router
  ddev-ssh-agent
  ddev-xhgui
)

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <image-repo>" >&2
  exit 2
fi

REPO="$1"
DOCKER_ORG="${DOCKER_ORG:?validate-image-repo.sh: DOCKER_ORG must be set}"

if [ "${REPO%%/*}" != "$DOCKER_ORG" ] || [ "$REPO" = "${REPO#*/}" ]; then
  echo "validate-image-repo.sh: '${REPO}' is not under the '${DOCKER_ORG}/' organization" >&2
  exit 1
fi

SUFFIX="${REPO#*/}"

for allowed in "${ALLOWED_SUFFIXES[@]}"; do
  if [ "$SUFFIX" = "$allowed" ]; then
    exit 0
  fi
done

if [[ "$SUFFIX" =~ ^ddev-dbserver-(mariadb|mysql)-[0-9]+\.[0-9]+$ ]]; then
  exit 0
fi

echo "validate-image-repo.sh: '${REPO}' is not one of the repositories this flow may push" >&2
exit 1
