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
#   extra_repo_suffixes  further repositories the same target produces
#
# TODO(#8609): only the default db variant (mariadb_11.8) is listed - see the
# TODO on autotag-images in the top-level Makefile.

# shellcheck disable=SC2034 # consumed by whatever sources this file
DDEV_IMAGE_CONFIGS=(
  'ddev-webserver|WebTag|containers/ddev-webserver containers/containers_shared.mk|ddev-webserver|images|false|ddev-webserver-prod'
  'ddev-traefik-router|TraefikRouterTag|containers/ddev-traefik-router containers/containers_shared.mk|ddev-traefik-router|container|false|'
  'ddev-ssh-agent|SSHAuthTag|containers/ddev-ssh-agent containers/containers_shared.mk|ddev-ssh-agent|container|false|'
  'ddev-xhgui|XhguiTag|containers/ddev-xhgui containers/containers_shared.mk|ddev-xhgui|container|false|'
  'ddev-dbserver-mariadb-11.8|BaseDBTag|containers/ddev-dbserver containers/get_arch.sh|ddev-dbserver|mariadb_11.8|true|'
)
