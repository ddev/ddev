#!/usr/bin/env bash

# This script wraps commands to catch and save their stderr output.
# When you run a command through it, stderr gets saved to /tmp/ddev-log-stderr-*.txt
# but only if the command fails.
#
# There are two different contexts where this is used:
#
# 1. During image build (additional layers in `.ddev/.webimageBuild/Dockerfile`)
#    - Use: log-stderr.sh <command>
#    - Errors saved here will trigger a rebuild without cache on next `ddev start`
#    - This catches build-time problems like missing dependencies or network issues
#
# 2. During container startup (using /start.sh script)
#    - Use: log-stderr.sh [--timeout <seconds>] <command>
#    - Errors are shown to the user but DON'T trigger rebuild (image is already built)
#    - Optional --timeout can be used for commands that might hang (like network calls)
#    - Timeout logs are saved separately and cleaned up by --show
#    - This catches runtime issues like slow networks or unavailable services
#
# During `ddev start`, we call "log-stderr.sh --show" from app.Start() to display
# all collected warnings to the user.

# Function to display usage information
usage() {
  echo "Usage: $(basename "$0") [-t timeout] [-r] [-s] command"
  echo "  -d, --debug          Enables debug mode"
  echo "  -s, --show           Shows stderr log (no command required)"
  echo "  -t, --timeout <int>  Timeout in seconds (default: no timeout)"
  exit 1
}

# Defaults
debug=false
show=false
timeout=0

# Parse command line options
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -d|--debug)
      debug=true
      shift
      ;;
    -s|--show)
      show=true
      shift
      ;;
    -t|--timeout)
      timeout="$2"
      shift 2
      ;;
    -*)
      usage
      ;;
    *)
      command=("$@")
      break
      ;;
  esac
done

[ "${debug}" = "true" ] && set -x

# show stderr output
if [ "${show}" = "true" ]; then
  # find /tmp -maxdepth 1 -name 'ddev-log-stderr-*.txt': Searches for files matching the pattern.
  # -printf "%C@ %p\n": Outputs the creation time (%C@) in seconds since the epoch followed by the file path (%p).
  # sort -n: Sorts the output numerically (-n).
  # awk '{print $2}': Extracts the file paths from the sorted output.
  # xargs cat --squeeze-blank: Concatenates content by collapsing multiple blank lines into a single blank line.
  find /tmp -maxdepth 1 -name 'ddev-log-stderr-*.txt' -printf "%C@ %p\n" | sort -n | awk '{print $2}' | xargs --no-run-if-empty cat --squeeze-blank
  exit_code=$?
  # Logs are used to check if we want to run `docker-compose build` without cache in `ddev start`.
  # But if these are timeout related logs, we don't want them to trigger build without cache,
  # because timeout is only used in /start.sh, which happens AFTER the web container is built.
  rm -f /tmp/ddev-log-stderr-timeout-*.txt || true
  exit "${exit_code}"
fi

# If no command is provided, show usage
if [ -z "${command[*]}" ]; then
  usage
fi

if whoami &>/dev/null; then
  whoami=$(whoami)
else
  whoami=$(id -u)
fi

# get a unique identifier for a filename
identifier="$(echo -n "${whoami}-${command[*]}" | md5sum | awk '{print $1}')"

error_file="/tmp/ddev-log-stderr-${identifier}.txt"

# Timeout is used only in /start.sh, we want to log to a different file
if [ "${timeout}" = "true" ]; then
  error_file="/tmp/ddev-log-stderr-timeout-${identifier}-$(date +%s).txt"
fi

tmp_error_file="/tmp/tmp-ddev-log-stderr-${identifier}-$(date +%s).txt"

# Run the command with timeout if specified
if [ "${timeout}" -gt 0 ]; then
  timeout "${timeout}" "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
else
  "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
fi

exit_code=$?

# Wait for background process substitution (tee) to complete writing to file
wait

# Exit on success
if [ "${exit_code}" -eq 0 ]; then
  rm -f "${tmp_error_file}" "${error_file}"
  exit "${exit_code}"
fi

# If it is a timeout error (see 'timeout --help')
# Timeout should only be used for commands in /start.sh that might take a long time due to network issues
if [ "${timeout}" -gt 0 ] && [ "${exit_code}" -eq 124 ]; then
  echo | tee -a "${tmp_error_file}" >&2
  echo "Command '${command[*]}' timed out after ${timeout} seconds" | tee -a "${tmp_error_file}" >&2
  echo "This may be due to a slow internet connection" | tee -a "${tmp_error_file}" >&2
  echo "You can rerun it inside the web container with:" | tee -a "${tmp_error_file}" >&2
  echo "  - ddev exec ${command[*]}" | tee -a "${tmp_error_file}" >&2
  echo | tee -a "${tmp_error_file}" >&2
fi

# If stderr is empty
if [ ! -s "${tmp_error_file}" ]; then
  echo "Command '${command[*]}' didn't return stderr output" | tee -a "${tmp_error_file}" >&2
fi

# Write to the log
{ printf "Warning: command '%s' run as '%s' failed with exit code %s:\n" "${command[*]}" "${whoami}" "${exit_code}"; cat "${tmp_error_file}" --squeeze-blank; echo; } >> "${error_file}"
rm -f "${tmp_error_file}"

exit "${exit_code}"
