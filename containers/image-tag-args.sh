#!/usr/bin/env bash
# image-tag-args.sh - sourced by the image tag cleanup scripts.
#
# expand_equals_args "$@" sets EXPANDED_ARGS to the same words with each
# --flag=value split into --flag value, so a script's parser only handles the
# two-word form. Callers then run: set -- ${EXPANDED_ARGS[@]+"${EXPANDED_ARGS[@]}"}

expand_equals_args() {
  EXPANDED_ARGS=()
  local arg
  for arg in "$@"; do
    case "$arg" in
      --*=*) EXPANDED_ARGS+=("${arg%%=*}" "${arg#*=}") ;;
      *) EXPANDED_ARGS+=("$arg") ;;
    esac
  done
}
