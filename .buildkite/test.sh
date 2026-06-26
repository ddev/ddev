#!/usr/bin/env bash
# This script is used to build ddev/ddev using buildkite

set -eu -o pipefail

# Disable git pager
export GIT_PAGER=""

# We can skip builds with commit message of [skip buildkite] or [skip ci]
DDEV_COMMIT_MESSAGE=$(git log -1 --pretty=%s 2>/dev/null || echo "")
if [[ ${BUILDKITE_MESSAGE:-} == *"[skip buildkite]"* ]] || [[ ${BUILDKITE_MESSAGE:-} == *"[skip ci]"* ]] || [[ ${DDEV_COMMIT_MESSAGE} == *"[skip buildkite]"* ]] || [[ ${DDEV_COMMIT_MESSAGE} == *"[skip ci]"* ]]; then
  echo "+++ SKIP: Build skipped due to commit message"
  echo "BUILDKITE_MESSAGE=${BUILDKITE_MESSAGE:-}"
  echo "DDEV_COMMIT_MESSAGE=${DDEV_COMMIT_MESSAGE}"
  exit 0
fi

git fetch --depth=1 --no-tags https://github.com/ddev/ddev public-variables:refs/public-variables-tmp
while IFS= read -r varname; do
  [[ "$varname" == "README.md" ]] && continue
  # MSYS_NO_PATHCONV prevents Git for Windows bash from mangling the ref:path syntax
  value=$(MSYS_NO_PATHCONV=1 git show "refs/public-variables-tmp:.github/public-variables/$varname")
  echo "$varname=${value}"
  export "$varname=$value"
done < <(MSYS_NO_PATHCONV=1 git ls-tree --name-only refs/public-variables-tmp:.github/public-variables/)
git update-ref -d refs/public-variables-tmp

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin
os=$(go env GOOS)

# GOTEST_SHORT=16 means drupal11
export GOTEST_SHORT=16
if [ ${OSTYPE:-unknown}  = "msys" ]; then export GOTEST_SHORT=true; fi

export DDEV_SKIP_NODEJS_TEST=true

export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false

# Helper: try to remove all containers via an SSH-style shell command.
# Usage: try_cleanup_containers_via_ssh <cmd> [<args>...]
# Returns 0 if containers are clean, 1 if containers remain (deep cleanup needed).
function try_cleanup_containers_via_ssh {
  local -a ssh_cmd=("$@")
  "${ssh_cmd[@]}" bash -lc '
    ids=$(docker ps -aq || true)
    if [ -n "$ids" ]; then
      docker rm -f $ids >/dev/null 2>&1 || true
    fi
    remaining=$(docker ps -aq || true)
    if [ -z "$remaining" ]; then
      echo "No containers remain; skipping docker-state cleanup"
      exit 0
    fi
    echo "CLEANUP REQUIRED: Containers still remain after docker rm -f" >&2
    docker ps -a >&2 || true
    exit 1
  '
}

# Helper: try to remove all containers via the local docker CLI.
# Returns 0 if containers are clean, 1 if containers remain (deep cleanup needed).
function try_cleanup_containers_native {
  local ids
  ids=$(docker ps -aq || true)
  if [ -n "$ids" ]; then
    docker rm -f "$ids" >/dev/null 2>&1 || true
  fi
  local remaining
  remaining=$(docker ps -aq || true)
  if [ -n "$remaining" ]; then
    echo "CLEANUP REQUIRED: Containers still remain after docker rm -f" >&2
    docker ps -a >&2 || true
    return 1
  fi
  echo "No containers remain; skipping docker-state cleanup"
  return 0
}

# Reset global config to defaults so no stale settings survive across runs.
# Provider-specific setup below can layer overrides on top (e.g. non-privileged ports for podman-rootless).
rm -f ~/.ddev/global_config.yaml
ddev config global >/dev/null

# On macOS, we can have several different docker providers, allow testing all
# In cleanup, stop everything we know of but leave either Orbstack or Docker Desktop running
if [ "${os:-}" = "darwin" ]; then
  function cleanup {
    command -v orb 2>/dev/null && echo "Stopping orbstack" && (nohup orb stop &)
    sleep 3 # Since we backgrounded orb stop, make sure it completes
    if [ -f /Applications/Docker.app ]; then echo "Stopping Docker Desktop" && (killall com.docker.backend || true); fi
    command -v colima 2>/dev/null && echo "Stopping colima_vz" && (colima stop -f vz || true)
    command -v limactl 2>/dev/null && echo "Stopping lima" && (limactl stop -f lima-vz || true)
    if [ -f ~/.rd/bin/rdctl ]; then echo "Stopping Rancher Desktop" && (~/.rd/bin/rdctl shutdown || true); fi
    command -v podman 2>/dev/null && echo "Stopping podman machine" && (podman machine stop || true)
    docker context use default
    # Leave orbstack running as the most likely to be reliable, otherwise Docker Desktop
    if command -v orb 2>/dev/null ; then
      docker context use orbstack
      echo "Starting orbstack" && (nohup orb start &)
    else
      docker context use desktop-linux
      open -a Docker
    fi
    sleep 5
  }
  trap cleanup EXIT

  # Start with a predictable docker provider running
  cleanup

  echo "initial docker context situation:"
  docker context ls

  # For Lima and Colima, as of Lima 1.0.4, having orbstack running
  # makes lima fail, see https://github.com/lima-vm/lima/issues/3145#issuecomment-2613728408
  command -v orb 2>/dev/null && echo "Stopping orbstack" && (nohup orb stop &)
  sleep 3 # Since we backgrounded orb stop, make sure it completes

  # Now start the docker provider we want
  case ${DOCKER_TYPE:=none} in
    "colima_vz")
      export COLIMA_INSTANCE=vz
      colima start ${COLIMA_INSTANCE}

      if ! try_cleanup_containers_via_ssh colima ssh -p "${COLIMA_INSTANCE}" --; then
        echo "Performing deep cleanup: removing container state and restarting colima"
        colima ssh -p "${COLIMA_INSTANCE}" -- sudo bash -lc 'rm -rf /var/lib/docker/containers/*'
        colima restart ${COLIMA_INSTANCE}

        # Verify cleanup was successful
        remaining_after_cleanup=$(colima ssh -p "${COLIMA_INSTANCE}" -- bash -lc 'docker ps -aq' || true)
        if [ -n "$remaining_after_cleanup" ]; then
          echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
          colima ssh -p "${COLIMA_INSTANCE}" -- bash -lc 'docker ps -a' >&2 || true
          exit 1
        fi
        echo "Deep cleanup succeeded: all containers removed"
      fi
      docker context use colima-${COLIMA_INSTANCE}
      ;;

    "lima")
      export LIMA_INSTANCE=lima-vz
      export HOMEDIR=/home/testbot.linux
      limactl start ${LIMA_INSTANCE}

      if ! try_cleanup_containers_via_ssh limactl shell "${LIMA_INSTANCE}"; then
        echo "Performing deep cleanup: removing container state and restarting docker"
        limactl shell lima-vz bash -lc "rm -rf ${HOMEDIR}/.local/share/docker/containers/*"
        limactl shell ${LIMA_INSTANCE} systemctl --user restart docker

        # Wait for docker to come back up inside lima
        for i in {1..30}; do
          if limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps >/dev/null 2>&1'; then
            break
          fi
          echo "Waiting for docker to restart in lima: $i"
          sleep 1
        done

        # Verify cleanup was successful
        remaining_after_cleanup=$(limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps -aq' || true)
        if [ -n "$remaining_after_cleanup" ]; then
          echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
          limactl shell ${LIMA_INSTANCE} bash -lc 'docker ps -a' >&2 || true
          exit 1
        fi
        echo "Deep cleanup succeeded: all containers removed"
      fi
      docker context use lima-${LIMA_INSTANCE}
      ;;

    "docker-desktop")
      open -a Docker
      docker context use desktop-linux
      ;;

    "orbstack")
      nohup orb start &
      sleep 3
      docker context use orbstack
      ;;

    "rancher-desktop")
      ~/.rd/bin/rdctl start
      for i in {1..120}; do
        if docker context use rancher-desktop >/dev/null 2>&1 ; then
          break
        fi
        echo "$(date): Waiting for rancher-desktop context to be available"
        sleep 1
      done
      docker context use rancher-desktop
      ;;

    "podman-rootless")
      podman machine start
      docker context use podman-rootless

      # Rootless podman cannot bind privileged ports (<1024); override the defaults.
      ddev config global --router-http-port=8080 --router-https-port=8443

      if ! try_cleanup_containers_native; then
        echo "ERROR: Containers remain after podman machine start; aborting" >&2
        docker ps -a >&2 || true
        exit 1
      fi
      ;;

    *)
      echo "no DOCKER_TYPE specified, exiting" && exit 10
      ;;
  esac
fi

# Handle docker-ce cleanup for WSL and other native Docker CE instances
if [ "${DOCKER_TYPE:-}" = "docker-ce" ] || [ "${DOCKER_TYPE:-}" = "wsl2dockerinside" ]; then
  if ! try_cleanup_containers_native; then
    echo "Performing deep cleanup: removing container state and restarting docker"
    sudo bash -c "rm -rf /var/lib/docker/containers/*"
    sudo systemctl restart docker

    # Wait for docker to come back up
    for i in {1..30}; do
      if docker ps >/dev/null 2>&1 ; then
        break
      fi
      echo "Waiting for docker to restart: $i"
      sleep 1
    done

    # Verify cleanup was successful
    remaining_after_cleanup=$(docker ps -aq || true)
    if [ -n "$remaining_after_cleanup" ]; then
      echo "ERROR: Cleanup failed, containers still remain after deep cleanup:" >&2
      docker ps -a >&2 || true
      exit 1
    fi
    echo "Deep cleanup succeeded: all containers removed"
  fi
fi

# Find a suitable timeout command for reliability and readability
if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT="gtimeout"
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT="timeout"
else
  echo "Error: Neither 'gtimeout' nor 'timeout' found in PATH." >&2
  exit 1
fi

# On Windows, start Docker Desktop if this job uses it, before waiting on docker.
# This is the Windows analogue of the lima/colima/orbstack provider startup above;
# it is provider setup, separate from the sanetestbot.sh sanity check. No-op on
# non-Windows and on docker-ce-inside cases.
bash "$(dirname "$0")/start-docker-desktop.sh" || true

# Make sure docker is working
echo "Waiting for docker provider to come up: $(date)"
date && ${TIMEOUT} 3m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
  sleep 10
  echo "Waiting: $(date)"
done'
echo "Testing again to make sure docker came up: $(date)"
if ! docker ps >/dev/null 2>&1 ; then
  echo "Docker is not running, exiting"
  exit 1
fi

echo
echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER:-unknown} for OS=${os:-} DOCKER_TYPE=${DOCKER_TYPE:-notset} in ${PWD} with GOTEST_SHORT=${GOTEST_SHORT:-notset} golang=$(go version | awk '{print $3}') ddev version=$(ddev --version | awk '{print $3}')"

echo
case ${DOCKER_TYPE:-none} in
  "docker-ce")
    echo "Running docker-ce (Docker CE)"
    ;;
  "docker-desktop")
    echo "docker-desktop for mac version=$(scripts/docker-desktop-version.sh)"
    ;;
  "colima")
    echo "colima version=$(colima version)"
    ;;
  "colima_vz")
    echo "colima version=$(colima version)"
    ;;
  "lima")
    echo "limactl --version=$(limactl --version)"
    ;;
  "orbstack")
    echo "orbstack version=$(orbctl version)"
    ;;
  "rancher-desktop")
    echo "rancher-desktop=$(~/.rd/bin/rdctl version)"
    ;;
  "podman-rootless")
    echo "podman version=$(podman --version)"
    ;;
  "wsl2dockerinside")
    echo "Running wsl2dockerinside"
    ;;
  "dockerforwindows")
    echo "Running Windows docker desktop for windows"
    ;;
  "wsl2-docker-desktop")
    echo "Running wsl2-docker-desktop"
    ;;
  *)
    echo "$DOCKER_TYPE not found"
    ;;
esac

echo "Docker version:"
docker version
if command -v ddev >/dev/null ; then
  echo "ddev version:"
  ddev version
else
  echo "ddev not installed"
fi
echo

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

# If this is a PR and the diff doesn't have code, skip it
set -x
if [ "${BUILDKITE_PULL_REQUEST:-false}" != "false" ]; then
  # Find the merge base between the PR branch and the base branch
  MERGE_BASE=$(git merge-base HEAD refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-})
  # Check if there are any changes in the specified directories or files since the merge base
  if ! git diff --name-only "$MERGE_BASE" | grep -E '^(\.buildkite/|Makefile$|pkg/|cmd/|vendor/|winpkg/|go\.)' >/dev/null; then
    echo "+++ SKIP: No relevant code changes found"
    echo "No changes in: .buildkite/, Makefile, pkg/, cmd/, vendor/, winpkg/, go.*"
    exit 0
  fi

fi

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"

${TIMEOUT} 10m bash "$(dirname "$0")/testbot_maintenance.sh"

# Our testbot should be sane, run the testbot checker to make sure.
echo "--- running sanetestbot.sh"
${TIMEOUT} 60s bash "$(dirname "$0")/sanetestbot.sh"

# Close the setup sections before starting tests
echo "~~~ Setup complete, starting tests"

# Make sure we start with mutagen daemon off.
unset MUTAGEN_DATA_DIRECTORY
if [ -f ~/.ddev/bin/mutagen -o -f ~/.ddev/bin/mutagen.exe ]; then
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen sync terminate -a || true
  MUTAGEN_DATA_DIRECTORY=~/.mutagen ~/.ddev/bin/mutagen daemon stop || true
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen daemon stop || true
fi
if command -v killall >/dev/null ; then
  killall mutagen || true
fi

echo "--- Running tests..."

# Note: Windows installer tests are no longer run here. They run in their own
# dedicated Buildkite pipeline (.buildkite/installer-test.sh +
# .buildkite/windows-installer.yml), decoupled from this suite so they can fan
# out across runners.

make ${MAKE_TARGET:-test} TESTARGS="${TESTARGS:-}" TESTPKG="${TESTPKG:-}" TESTFILE="${TESTFILE:-}" | sed -u 's/^--- FAIL:/+++ FAIL:/; /\//!s/^=== RUN /--- RUN /'
RV=$?
echo "test.sh completed with status=$RV"
ddev poweroff || true

exit $RV
