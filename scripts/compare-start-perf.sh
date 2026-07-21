#!/usr/bin/env bash
# compare-start-perf.sh
#
# Compare `ddev start` and `ddev utility rebuild` performance between any two
# DDEV commits.
#
# It builds each commit in an isolated git worktree (your working tree is left
# untouched), creates a throwaway project for each, and times repeated
# `ddev start` and `ddev utility rebuild` runs. Results are printed as plain
# text.
#
# Works on macOS (bash 3.2 / BSD date) and Linux. It has no hard dependency on
# Mutagen and runs whether or not Mutagen is installed. By default it uses your
# global performance mode, so Mutagen sync time (if any) is part of the measured
# start time; use -m to force it on or off.
#
# Test projects are created under ~/tmp (not $TMPDIR) because some Docker
# providers (e.g. OrbStack) only share $HOME with containers, not $TMPDIR.
#
# Usage:
#   scripts/compare-start-perf.sh [options] <commit-a> <commit-b>
#
# Before each commit it runs `ddev poweroff` with that commit's own binary, so the
# shared global state (ddev-router, ddev-ssh-agent, Mutagen daemon) is torn down
# and recreated by the version under test. NOTE: this stops any other DDEV
# projects you have running. Use -P to skip it.
#
# Options:
#   -n NUM     number of timed (warm) start runs per commit     (default: 5)
#   -b NUM     number of timed 'ddev utility rebuild' runs      (default: 3)
#              per commit; 0 disables the rebuild comparison
#   -t TYPE    ddev project type for the test project          (default: php)
#   -m MODE    Mutagen: on | off | default (global config)     (default: default)
#   -P         do NOT run 'ddev poweroff' before each commit   (default: do it)
#   -k         keep the built binaries and test projects (skip cleanup)
#   -h         show this help
#
# Example:
#   scripts/compare-start-perf.sh upstream/main HEAD
#   scripts/compare-start-perf.sh -n 10 -m off v1.25.0 HEAD
#   scripts/compare-start-perf.sh -m on upstream/main HEAD
#   scripts/compare-start-perf.sh -b 5 upstream/main HEAD

set -u

RUNS=5
REBUILD_RUNS=3
PROJECT_TYPE=php
KEEP=0
MUTAGEN_MODE=default
POWEROFF=1

usage() {
  sed -n '2,42p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

while getopts ":n:b:t:m:Pkh" opt; do
  case "$opt" in
    n) RUNS=$OPTARG ;;
    b) REBUILD_RUNS=$OPTARG ;;
    t) PROJECT_TYPE=$OPTARG ;;
    m) MUTAGEN_MODE=$OPTARG ;;
    P) POWEROFF=0 ;;
    k) KEEP=1 ;;
    h) usage 0 ;;
    :) echo "Error: -$OPTARG requires an argument" >&2; usage 1 ;;
    \?) echo "Error: unknown option -$OPTARG" >&2; usage 1 ;;
  esac
done
shift $((OPTIND - 1))

if [ "$#" -ne 2 ]; then
  echo "Error: need exactly two commit refs" >&2
  usage 1
fi
REF_A=$1
REF_B=$2

case "$RUNS" in
  ''|*[!0-9]*) echo "Error: -n must be a positive integer" >&2; exit 1 ;;
esac
[ "$RUNS" -ge 1 ] || { echo "Error: -n must be >= 1" >&2; exit 1; }

case "$REBUILD_RUNS" in
  ''|*[!0-9]*) echo "Error: -b must be a non-negative integer" >&2; exit 1 ;;
esac

# Map Mutagen mode to the config flag value and the config.yaml key fallback.
case "$MUTAGEN_MODE" in
  on)      PERF_FLAG=mutagen; PERF_KEY=mutagen ;;
  off)     PERF_FLAG=none;    PERF_KEY=none ;;
  default) PERF_FLAG="";      PERF_KEY="" ;;
  *) echo "Error: -m must be one of: on, off, default" >&2; exit 1 ;;
esac

# --- locate the repo ---------------------------------------------------------
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || {
  echo "Error: must be run from inside the DDEV git repository" >&2; exit 1; }
[ -f "$REPO_ROOT/Makefile" ] || { echo "Error: no Makefile at $REPO_ROOT" >&2; exit 1; }

for ref in "$REF_A" "$REF_B"; do
  git -C "$REPO_ROOT" rev-parse --verify --quiet "$ref^{commit}" >/dev/null || {
    echo "Error: '$ref' is not a valid commit" >&2; exit 1; }
done

command -v make >/dev/null 2>&1 || { echo "Error: 'make' not found in PATH" >&2; exit 1; }

# --- portable high-resolution clock ------------------------------------------
# Pick the best available sub-second timer. Falls back to whole seconds.
TIMER=seconds
if command -v perl >/dev/null 2>&1 && perl -MTime::HiRes -e 1 >/dev/null 2>&1; then
  TIMER=perl
elif command -v gdate >/dev/null 2>&1 && case "$(gdate +%N 2>/dev/null)" in ''|*[!0-9]*) false ;; *) true ;; esac; then
  TIMER=gdate
elif case "$(date +%N 2>/dev/null)" in ''|*[!0-9]*) false ;; *) true ;; esac; then
  TIMER=gnudate
elif [ -n "${EPOCHREALTIME:-}" ]; then
  TIMER=epochrealtime
fi

now() {
  case "$TIMER" in
    perl)          perl -MTime::HiRes=time -e 'printf "%.6f\n", time' ;;
    gdate)         gdate +%s.%N ;;
    gnudate)       date +%s.%N ;;
    epochrealtime) printf '%s\n' "${EPOCHREALTIME/,/.}" ;;
    *)             date +%s ;;
  esac
}

elapsed() { awk -v a="$1" -v b="$2" 'BEGIN { printf "%.2f", b - a }'; }
median()  {
  printf '%s\n' "$@" | sort -n | awk '
    { v[NR] = $1 }
    END {
      n = NR; if (n == 0) { printf "0"; exit }
      if (n % 2) printf "%.2f", v[(n + 1) / 2]; else printf "%.2f", (v[n / 2] + v[n / 2 + 1]) / 2
    }'
}
stats() {
  printf '%s\n' "$@" | sort -n | awk '
    { v[NR] = $1; sum += $1 }
    END {
      n = NR
      if (n == 0) { printf "no data"; exit }
      if (n % 2) med = v[(n + 1) / 2]; else med = (v[n / 2] + v[n / 2 + 1]) / 2
      printf "min=%.2fs  median=%.2fs  mean=%.2fs  max=%.2fs  (n=%d)", v[1], med, sum / n, v[n], n
    }'
}
# A run is "anomalous" if it exceeds OUTLIER_FACTOR x median. On the always-build
# code path a single start can trigger a full `docker build --no-cache` (minutes),
# which would otherwise wreck the arithmetic mean.
OUTLIER_FACTOR=3
# trimmed_mean <median> <values...> : mean of the non-anomalous runs.
trimmed_mean() {
  med=$1; shift
  printf '%s\n' "$@" | awk -v med="$med" -v f="$OUTLIER_FACTOR" '
    { if (med <= 0 || $1 <= f * med) { s += $1; c++ } }
    END { if (c) printf "%.2f", s / c; else printf "0" }'
}
# outlier_runs <median> <values...> : prints "run N (Xs)" for each anomalous run.
outlier_runs() {
  med=$1; shift
  i=0
  for v in "$@"; do
    i=$((i + 1))
    if awk -v v="$v" -v med="$med" -v f="$OUTLIER_FACTOR" 'BEGIN { exit !(med > 0 && v > f * med) }'; then
      printf 'run %d (%ss) ' "$i" "$v"
    fi
  done
}
# report_stats <name> <values...> : per-commit stats line for a named metric
# (e.g. "start", "rebuild"), with an outlier note when a run looks anomalous.
report_stats() {
  name=$1; shift
  med=$(median "$@")
  echo "  $name: $(stats "$@")"
  outs=$(outlier_runs "$med" "$@")
  if [ -n "$outs" ]; then
    echo "  NOTE: anomalous run(s) excluded from trimmed mean: $outs"
    echo "        (a one-time 'docker build --no-cache' rebuild or a Docker/network stall"
    echo "         can cause this; median and trimmed mean are the reliable numbers)"
    echo "  $name (trimmed): median=${med}s  mean=$(trimmed_mean "$med" "$@")s"
  fi
}

# --- workspace + cleanup -----------------------------------------------------
# Build artifacts (worktrees, binaries, logs) can live in $TMPDIR; they are not
# mounted into containers.
WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/ddev-perf.XXXXXX") || exit 1
# Test projects must live under $HOME so the Docker provider can bind-mount them;
# some providers (e.g. OrbStack) only share $HOME, not $TMPDIR. ~/tmp is used by
# convention.
mkdir -p "$HOME/tmp" || { echo "Error: cannot create $HOME/tmp" >&2; exit 1; }
PROJ_BASE=$(mktemp -d "$HOME/tmp/ddev-perf.XXXXXX") || exit 1
PROJ_A="ddevperf-a-$$"
PROJ_B="ddevperf-b-$$"
export DDEV_NO_INSTRUMENTATION=true

cleanup() {
  if [ "$KEEP" -eq 1 ]; then
    echo; echo "Kept binaries in $WORKDIR and projects in $PROJ_BASE"; return
  fi
  # Tear down test projects with whichever binary built them.
  if [ -x "$WORKDIR/bin-a/ddev" ]; then
    PATH="$WORKDIR/bin-a:$PATH" "$WORKDIR/bin-a/ddev" delete -Oy "$PROJ_A" >/dev/null 2>&1
  fi
  if [ -x "$WORKDIR/bin-b/ddev" ]; then
    PATH="$WORKDIR/bin-b:$PATH" "$WORKDIR/bin-b/ddev" delete -Oy "$PROJ_B" >/dev/null 2>&1
  fi
  git -C "$REPO_ROOT" worktree prune >/dev/null 2>&1
  rm -rf "$WORKDIR" "$PROJ_BASE"
}
trap cleanup EXIT INT TERM

# --- build one commit into $WORKDIR/bin-<label> ------------------------------
build_commit() {
  ref=$1; label=$2
  wt="$WORKDIR/wt-$label"
  bindir="$WORKDIR/bin-$label"
  echo "Building $ref ..."
  git -C "$REPO_ROOT" worktree add --quiet --detach "$wt" "$ref" || {
    echo "Error: failed to create worktree for $ref" >&2; exit 1; }
  if ! ( cd "$wt" && make ) >"$WORKDIR/build-$label.log" 2>&1; then
    echo "Error: build failed for $ref. Last lines of build log:" >&2
    tail -n 20 "$WORKDIR/build-$label.log" >&2
    exit 1
  fi
  mkdir -p "$bindir"
  # The default `make` target builds for the host os/arch into .gotmp/bin/<os>_<arch>/.
  hostbin=$(ls "$wt"/.gotmp/bin/*/ddev 2>/dev/null | head -1)
  [ -n "$hostbin" ] || { echo "Error: could not find built ddev binary for $ref" >&2; exit 1; }
  cp "$hostbin" "$bindir/ddev"
  # ddev-hostname is a separate binary only in newer DDEV; older releases embed
  # hostname handling in the ddev binary itself. Copy it only if it was built.
  hostnamebin="$(dirname "$hostbin")/ddev-hostname"
  if [ -f "$hostnamebin" ]; then
    cp "$hostnamebin" "$bindir/ddev-hostname"
  else
    echo "  Note: $ref has no separate ddev-hostname binary (older layout); continuing"
  fi
  git -C "$REPO_ROOT" worktree remove --force "$wt" >/dev/null 2>&1
  ver=$("$bindir/ddev" --version 2>/dev/null)
  eval "RESULT_VERSION_$label=\$ver"
  # `ddev debug` was renamed to `ddev utility` (with `debug` kept as an alias),
  # so this only matters for commits that predate the rename.
  if "$bindir/ddev" utility --help >/dev/null 2>&1; then
    eval "REBUILD_CMD_$label=utility"
  else
    eval "REBUILD_CMD_$label=debug"
  fi
}

# --- benchmark one binary ----------------------------------------------------
# Sets globals: RESULT_VERSION, RESULT_COLD, and fills the named arrays via eval.
benchmark() {
  label=$1; proj=$2; warm_arrname=$3; rebuild_arrname=$4
  bindir="$WORKDIR/bin-$label"
  projdir="$PROJ_BASE/proj-$label"
  log="$WORKDIR/run-$label.log"
  REF_VAL=$(eval "echo \$RESULT_VERSION_$label")
  rebuild_cmd=$(eval "echo \$REBUILD_CMD_$label")
  export PATH="$bindir:$PATH_ORIG"
  : >"$log"

  # diagnostics: dump container logs so a failed start is debuggable.
  dump_diagnostics() {
    echo "------- ddev list -------" >&2
    ( "$bindir/ddev" list 2>&1 | tail -n 20 ) >&2
    for svc in web db; do
      echo "------- ddev logs -s $svc (tail) -------" >&2
      ( cd "$projdir" && "$bindir/ddev" logs -s "$svc" 2>&1 | tail -n 30 ) >&2
      echo "------- docker logs ddev-$proj-$svc (tail) -------" >&2
      docker logs --tail 30 "ddev-$proj-$svc" >&2 2>&1
    done
    echo "------- docker ps -a (project containers) -------" >&2
    docker ps -a --filter "name=ddev-$proj" --format '{{.Names}}\t{{.Status}}' >&2 2>&1
  }

  # run_ddev <error-description> <ddev args...> : run quietly, logging output;
  # on failure print the captured output and container diagnostics, then exit.
  run_ddev() {
    desc=$1; shift
    if ! ( cd "$projdir" && "$bindir/ddev" "$@" ) >>"$log" 2>&1; then
      echo >&2
      echo "Error: $desc (commit $label, $REF_VAL)" >&2
      echo "------- last output of 'ddev $*' -------" >&2
      tail -n 40 "$log" >&2
      case "$1" in start|restart) dump_diagnostics ;; esac
      echo "----------------------------------------" >&2
      echo "Artifacts kept for inspection: project=$projdir log=$log" >&2
      KEEP=1
      exit 1
    fi
  }

  rm -rf "$projdir"; mkdir -p "$projdir"

  # Power off first so the shared global state (router, ssh-agent, Mutagen daemon)
  # is recreated by the version under test, not left over from another version.
  if [ "$POWEROFF" -eq 1 ]; then
    echo "  ddev poweroff (clean slate; stops any other running projects) ..."
    ( "$bindir/ddev" poweroff ) >>"$log" 2>&1 \
      || echo "  Warning: 'ddev poweroff' failed for $label (continuing)" >&2
  fi

  # Configure the project. Mutagen mode comes from -m (default: global config),
  # so by default Mutagen sync time is included in the measured start time.
  if [ -n "$PERF_FLAG" ] \
    && ( cd "$projdir" && "$bindir/ddev" config --project-name="$proj" --project-type="$PROJECT_TYPE" --docroot=. --performance-mode="$PERF_FLAG" ) >>"$log" 2>&1; then
    : # configured with the requested performance mode
  else
    # Either -m default, or an older release lacking --performance-mode: configure
    # plainly, then set the key directly if a specific mode was requested.
    run_ddev "'ddev config' failed" config --project-name="$proj" --project-type="$PROJECT_TYPE" --docroot=.
    [ -n "$PERF_KEY" ] && printf '\nperformance_mode: %s\n' "$PERF_KEY" >>"$projdir/.ddev/config.yaml"
  fi

  # Pre-download all images this build needs (run inside the project dir so it
  # fetches that project's exact images), so start/build timing measures the
  # build itself and not one-time image pulls. The subcommand was renamed from
  # `debug download-images` to `utility download-images`, so try both.
  echo "  pre-downloading images ..."
  if ! ( cd "$projdir" && "$bindir/ddev" utility download-images ) >>"$log" 2>&1; then
    ( cd "$projdir" && "$bindir/ddev" debug download-images ) >>"$log" 2>&1 \
      || echo "  Warning: image pre-download failed for $label (continuing anyway)" >&2
  fi

  # Prime the project: the very first build for this project is always
  # uncached, so use `ddev utility rebuild` (build the web image + restart)
  # rather than `ddev start`. This also means the warm-start runs below never
  # need to build anything, since nothing changes between them.
  echo "  cold build (ddev $rebuild_cmd rebuild; may take a few minutes) ..."
  t0=$(now)
  run_ddev "cold 'ddev $rebuild_cmd rebuild' failed" "$rebuild_cmd" rebuild
  t1=$(now)
  cold=$(elapsed "$t0" "$t1")
  eval "RESULT_COLD_$label=\$cold"
  echo "    cold build: ${cold}s"

  # Each timed run measures `ddev start` from a STOPPED state: an untimed
  # `ddev stop` first, then time `ddev start`. This is the representative
  # "start a stopped project" operation and is deterministic -- it avoids the
  # race that back-to-back `ddev start` calls on a live project can hit (the web
  # container gets recreated mid-healthcheck and exits).
  echo "  warmup stop/start (untimed) ..."
  run_ddev "warmup 'ddev stop' failed" stop
  run_ddev "warmup 'ddev start' failed" start -y

  echo "  timed starts from stopped state ($RUNS):"
  eval "$warm_arrname=()"
  i=1
  while [ "$i" -le "$RUNS" ]; do
    run_ddev "'ddev stop' before run $i failed" stop
    t0=$(now)
    run_ddev "'ddev start' run $i failed" start -y
    t1=$(now)
    e=$(elapsed "$t0" "$t1")
    eval "$warm_arrname+=(\"\$e\")"
    printf "    run %d: %ss\n" "$i" "$e"
    i=$((i + 1))
  done

  # Timed `ddev utility rebuild` runs: this is a full `docker build --no-cache`
  # of the web image followed by a project restart, and is the operation that
  # regressed in https://github.com/ddev/ddev/issues/8600 (a slow `chmod -R` in
  # the webimage Dockerfile). `ddev start` alone does not exercise this once the
  # image is cached, so it is timed separately here.
  eval "$rebuild_arrname=()"
  if [ "$REBUILD_RUNS" -gt 0 ]; then
    echo "  timed '$rebuild_cmd rebuild' runs ($REBUILD_RUNS; may take a few minutes) ..."
    i=1
    while [ "$i" -le "$REBUILD_RUNS" ]; do
      t0=$(now)
      run_ddev "'ddev $rebuild_cmd rebuild' run $i failed" "$rebuild_cmd" rebuild
      t1=$(now)
      e=$(elapsed "$t0" "$t1")
      eval "$rebuild_arrname+=(\"\$e\")"
      printf "    run %d: %ss\n" "$i" "$e"
      i=$((i + 1))
    done
  fi
}

# --- run ---------------------------------------------------------------------
PATH_ORIG=$PATH
SHA_A=$(git -C "$REPO_ROOT" rev-parse --short "$REF_A")
SHA_B=$(git -C "$REPO_ROOT" rev-parse --short "$REF_B")

echo "==================================================================="
echo "DDEV  ddev start / ddev utility rebuild  performance comparison"
echo "==================================================================="
echo "Commit A    : $REF_A ($SHA_A)"
echo "Commit B    : $REF_B ($SHA_B)"
echo "Warm runs   : $RUNS    Rebuild runs: $REBUILD_RUNS    Project type: $PROJECT_TYPE    Timer: $TIMER"
echo "Mutagen     : $MUTAGEN_MODE (start time includes Mutagen sync when enabled)"
echo

build_commit "$REF_A" a
build_commit "$REF_B" b
echo

echo "Commit A  $REF_A ($SHA_A)  $(eval echo \$RESULT_VERSION_a 2>/dev/null)"
benchmark a "$PROJ_A" WARM_A REBUILD_A
report_stats start "${WARM_A[@]}"
[ "$REBUILD_RUNS" -gt 0 ] && report_stats rebuild "${REBUILD_A[@]}"
echo

echo "Commit B  $REF_B ($SHA_B)  $(eval echo \$RESULT_VERSION_b 2>/dev/null)"
benchmark b "$PROJ_B" WARM_B REBUILD_B
report_stats start "${WARM_B[@]}"
[ "$REBUILD_RUNS" -gt 0 ] && report_stats rebuild "${REBUILD_B[@]}"
echo

# print_summary <title> <arrname-a> <arrname-b> : median/trimmed-mean comparison
# for one metric. Arrays are passed by name (not value) for bash 3.2 compatibility.
print_summary() {
  title=$1; an=$2; bn=$3
  eval "aset=(\"\${$an[@]}\")"
  eval "bset=(\"\${$bn[@]}\")"
  med_a=$(median "${aset[@]}"); tm_a=$(trimmed_mean "$med_a" "${aset[@]}")
  med_b=$(median "${bset[@]}"); tm_b=$(trimmed_mean "$med_b" "${bset[@]}")
  echo "Summary ($title)"
  echo "-------------------------------------------------------------------"
  printf "  A (%s): median=%ss  trimmed-mean=%ss\n" "$SHA_A" "$med_a" "$tm_a"
  printf "  B (%s): median=%ss  trimmed-mean=%ss\n" "$SHA_B" "$med_b" "$tm_b"
  awk -v a="$med_a" -v b="$med_b" 'BEGIN {
    d = b - a
    pct = (a != 0) ? d / a * 100 : 0
    if (d > 0)      printf "  B is SLOWER than A by %.2fs (%+.1f%%) on median\n", d, pct
    else if (d < 0) printf "  B is FASTER than A by %.2fs (%+.1f%%) on median\n", -d, pct
    else            printf "  No median difference\n"
  }'
}

echo "==================================================================="
print_summary "ddev start from stopped state" WARM_A WARM_B
if [ "$REBUILD_RUNS" -gt 0 ]; then
  echo
  print_summary "ddev utility rebuild" REBUILD_A REBUILD_B
fi
echo "==================================================================="
