#!/bin/bash

# This script is used to run a command with optional timeout
# and save stderr to /tmp/ddev-log-stderr-{whoami}-{command-md5sum}.txt

# Function to display usage information
usage() {
  echo "Usage: $(basename "$0") [-t timeout] [-r] [-s] command"
  echo "  -d, --debug          Enables debug mode"
  echo "  -s, --show           Shows stderr log (no command required)"
  echo "  -t, --timeout <int>  Timeout in seconds (default: no timeout)"
  echo "  -r, --remove         Removes stderr log for specified command"
  exit 1
}

# Defaults
debug=false
show=false
remove=false
timeout=0

# Parse command line options
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -d|--debug)
      debug=true
      shift
      ;;
    -r|--remove)
      remove=true
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
  exit $?
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

if [ "${remove}" = "true" ]; then
  rm -f "${error_file}"
  exit $?
fi

tmp_error_file="/tmp/tmp-ddev-log-stderr-${identifier}-$(date +%s).txt"

# Run the command with timeout if specified
if [ "${timeout}" -gt 0 ]; then
  timeout "${timeout}" "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
else
  "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
fi

exit_code=$?

# Exit on success
if [ "${exit_code}" -eq 0 ]; then
  rm -f "${tmp_error_file}" "${error_file}"
  exit "${exit_code}"
fi

# If it is a timeout error (see 'timeout --help')
if [ "${timeout}" -gt 0 ] && [ "${exit_code}" -eq 124 ]; then
  echo "Command '${command[*]}' timed out after ${timeout} seconds" | tee -a "${tmp_error_file}" >&2
fi

# If stderr is empty
if [ ! -s "${tmp_error_file}" ]; then
  echo "Command '${command[*]}' didn't return stderr output" | tee -a "${tmp_error_file}" >&2
fi

# Write to the log
{ printf "Warning: command '%s' run as '%s' failed with exit code %s:\n" "${command[*]}" "${whoami}" "${exit_code}"; cat "${tmp_error_file}" --squeeze-blank; echo; } >> "${error_file}"
rm -f "${tmp_error_file}"

exit "${exit_code}"
