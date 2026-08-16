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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ALLOWED_SUFFIXES=(
  ddev-webserver
  ddev-webserver-prod
  ddev-traefik-router
  ddev-ssh-agent
  ddev-xhgui
)

# The db repositories are exactly the variants in
# containers/ddev-dbserver/variants.txt - no pattern match, so a repository
# this project doesn't publish can't slip through on shape alone.
while IFS= read -r _repo; do
  [ -n "$_repo" ] && ALLOWED_SUFFIXES+=("$_repo")
done < <("$SCRIPT_DIR/ddev-dbserver/variants.sh" repos)

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

echo "validate-image-repo.sh: '${REPO}' is not one of the repositories this flow may push" >&2
exit 1
