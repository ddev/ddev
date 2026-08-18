#!/usr/bin/env bash
# image-configs.sh - the set of images the automatic build/push flow covers.
#
# Sourced (not executed) by wait-for-images.sh and by image-build-push.yml's
# detect job so the two can't drift apart. Keep in sync with the Makefile's
# autotag-images target.
#
# Fields, pipe-separated:
#   repo_suffix          Docker Hub repository under $DOCKER_ORG
#   tag_var              variable name in pkg/versionconstants/versionconstants.go
#   hash_paths           space-separated paths fed to hash-paths.sh
#   make_dir             directory under containers/ to run make in
#   make_target          make target that builds it
#   arch_suffixed        "true" if the target name takes an _<arch> suffix
#   arches               space-separated architectures to build
#   built_by_make        "true" if `make` at the repo root builds it locally;
#                        a "false" image can only come from the registry
#   extra_repo_suffixes  further repositories the same target produces

# shellcheck disable=SC2034 # consumed by whatever sources this file
DDEV_IMAGE_CONFIGS=(
  'ddev-webserver|WebTag|containers/ddev-webserver containers/containers_shared.mk|ddev-webserver|images|false|amd64 arm64|true|ddev-webserver-prod'
  'ddev-traefik-router|TraefikRouterTag|containers/ddev-traefik-router containers/containers_shared.mk|ddev-traefik-router|container|false|amd64 arm64|true|'
  'ddev-ssh-agent|SSHAuthTag|containers/ddev-ssh-agent containers/containers_shared.mk|ddev-ssh-agent|container|false|amd64 arm64|true|'
  'ddev-xhgui|XhguiTag|containers/ddev-xhgui containers/containers_shared.mk|ddev-xhgui|container|false|amd64 arm64|true|'
)

# Every ddev-dbserver variant shares one BaseDBTag (see GetDBImage() in
# pkg/docker/images.go), so a change under containers/ddev-dbserver moves the
# tag for all of them at once and every variant has to be rebuilt and pushed
# together - otherwise TestDdevAllDatabases and every test pinning a
# non-default database pulls a tag nobody published. `make` builds only the
# default variant locally, to keep an unrelated rebuild cheap; the rest exist
# only in the registry.
#
# The variant list comes from containers/ddev-dbserver/variants.txt via
# variants.sh, shared with that directory's Makefile and with
# push-tagged-dbimage.yml.
DDEV_DBSERVER_HASH_PATHS='containers/ddev-dbserver containers/get_arch.sh'
DDEV_DBSERVER_DEFAULT_TARGET='mariadb_11.8'

while IFS='|' read -r _target _arches; do
  [ -z "$_target" ] && continue
  _built_by_make=false
  [ "$_target" = "$DDEV_DBSERVER_DEFAULT_TARGET" ] && _built_by_make=true
  DDEV_IMAGE_CONFIGS+=(
    "ddev-dbserver-${_target/_/-}|BaseDBTag|${DDEV_DBSERVER_HASH_PATHS}|ddev-dbserver|${_target}|true|${_arches}|${_built_by_make}|"
  )
done < <("$(dirname "${BASH_SOURCE[0]}")/ddev-dbserver/variants.sh" list)
unset _target _arches _built_by_make
