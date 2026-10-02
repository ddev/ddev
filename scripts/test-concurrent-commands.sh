#!/usr/bin/env bash
# test-concurrent-commands.sh
#
# Runs random `ddev` commands against several projects at the same time, then
# checks the state they share: the ddev_default network, ddev-router and
# ddev-ssh-agent. It exists to exercise the global lock that serializes setup
# of those resources (https://github.com/ddev/ddev/issues/5527).
#
# Each wave starts one command per project at once (start, stop, restart,
# a hostname change plus start, or a read-only list), and sometimes a
# `ddev poweroff` alongside them. Individual commands may legitimately fail
# when they race a stop or poweroff, so a wave only fails when a log shows a
# known race signature. After every wave, each project is started again one
# at a time and the shared state must be intact.
#
# Works on Linux, macOS, and Windows (Git Bash). Test projects are created
# under ~/tmp because some Docker providers only share $HOME.
#
# Usage: scripts/test-concurrent-commands.sh
#   PROJECTS=4  number of projects
#   WAVES=5     number of waves
#   SEED=n      seed for the random command choice (default: random, printed)
#
# NOTE: this runs `ddev poweroff`, which stops all your running DDEV projects.

set -u
PROJECTS=${PROJECTS:-4}
WAVES=${WAVES:-5}
SEED=${SEED:-$((RANDOM * 32768 + RANDOM))}
BASE="$HOME/tmp/ddev-concurrent-test"
LOGS="$BASE/logs"
# Output that shows two ddev processes stepped on each other.
RACE_PATTERN='is already in use by container|Conflict\.|already exists|network .* not found|No such container'
failures=0

fail() {
  echo "FAIL: $*"
  failures=$((failures + 1))
}

cleanup() {
  ddev poweroff >/dev/null 2>&1
  for i in $(seq 1 "$PROJECTS"); do
    (cd "$BASE/proj$i" 2>/dev/null && ddev delete -Oy "ddevconc$i" >/dev/null 2>&1)
  done
  rm -rf "$BASE"
}
trap cleanup EXIT

RANDOM=$SEED
echo "ddev: $(command -v ddev), seed=$SEED, projects=$PROJECTS, waves=$WAVES"
rm -rf "$BASE"
mkdir -p "$LOGS"
for i in $(seq 1 "$PROJECTS"); do
  mkdir -p "$BASE/proj$i/web"
  echo "<?php echo 'ok';" >"$BASE/proj$i/web/index.php"
  (cd "$BASE/proj$i" && ddev config --project-type=php --docroot=web --project-name="ddevconc$i" >/dev/null 2>&1) || fail "config proj$i"
done

# run_op <wave> <project number> <op>: runs in the background, logging to $LOGS.
run_op() {
  local log="$LOGS/wave$1-proj$2-$3.log"
  cd "$BASE/proj$2" || return 1
  case "$3" in
    start) ddev start -y ;;
    stop) ddev stop ;;
    restart) ddev restart -y ;;
    hostname) ddev config --additional-hostnames="extra$RANDOM" && ddev start -y ;;
    list) ddev list && ddev describe ;;
  esac >"$log" 2>&1
}

ops=(start start stop restart hostname list)
for wave in $(seq 1 "$WAVES"); do
  echo "=== Wave $wave ==="
  pids=()
  for i in $(seq 1 "$PROJECTS"); do
    op=${ops[$((RANDOM % ${#ops[@]}))]}
    echo "proj$i: $op"
    run_op "$wave" "$i" "$op" &
    pids+=($!)
  done
  if [ $((RANDOM % 4)) -eq 0 ]; then
    echo "global: poweroff"
    ddev poweroff >"$LOGS/wave$wave-poweroff.log" 2>&1 &
    pids+=($!)
  fi
  for p in "${pids[@]}"; do wait "$p"; done

  if grep -lE "$RACE_PATTERN" "$LOGS"/wave"$wave"-*.log >/dev/null 2>&1; then
    fail "wave $wave: race signature in $(grep -lE "$RACE_PATTERN" "$LOGS"/wave"$wave"-*.log | tr '\n' ' ')"
  fi

  # Converge one at a time, then check the shared state.
  for i in $(seq 1 "$PROJECTS"); do
    (cd "$BASE/proj$i" && ddev start -y >"$LOGS/wave$wave-proj$i-converge.log" 2>&1) || fail "wave $wave: converge start proj$i failed"
  done
  [ "$(docker network ls --format '{{.Name}}' | grep -cx ddev_default)" -eq 1 ] || fail "wave $wave: not exactly one ddev_default network"
  [ "$(docker ps --format '{{.Names}}' | grep -cx ddev-router)" -eq 1 ] || fail "wave $wave: ddev-router is not running"
  [ "$(docker ps --format '{{.Names}}' | grep -cx ddev-ssh-agent)" -eq 1 ] || fail "wave $wave: ddev-ssh-agent is not running"
  for i in $(seq 1 "$PROJECTS"); do
    curl -sk --max-time 15 "https://ddevconc$i.ddev.site" | grep -q ok || fail "wave $wave: ddevconc$i not reachable through the router"
  done
  echo "Waits for the lock this wave: $(cat "$LOGS"/wave"$wave"-*.log | grep -c 'Waiting for another ddev process')"
done

if [ "$failures" -eq 0 ]; then
  echo "PASS"
else
  echo "FAILED ($failures failures, seed=$SEED); logs kept in $LOGS"
  trap - EXIT
  ddev poweroff >/dev/null 2>&1
fi
exit "$failures"
