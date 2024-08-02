#!/bin/bash

# This script is used to run a command with optional timeout
# and save stderr to /tmp/ddev-log-stderr-${whoami}.txt
set +eu

# Function to display usage information
usage() {
  echo "Usage: $(basename "$0") [-t timeout] command"
  echo "  -t, --timeout: Timeout in seconds (default: no timeout)"
  exit 1
}

# Default timeout value
timeout=0

if whoami &>/dev/null; then
  whoami=$(whoami)
else
  whoami=$(id -u)
fi

# Parse command line options
while [[ "$#" -gt 0 ]]; do
  case "$1" in
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

# If no command is provided, show usage
if [ -z "${command[*]}" ]; then
  usage
fi

tmp_error_file="/tmp/tmp-ddev-log-stderr-${whoami}-$(echo -n "${command[*]} $(date +%s)" | md5sum | awk '{print $1}').txt"

# Run the command with timeout if specified
if [ "${timeout}" -gt 0 ]; then
  timeout "${timeout}" "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
  exit_code=$?
else
  "${command[@]}" 2> >(tee -a "${tmp_error_file}" >&2)
  exit_code=$?
fi

# Exit on success
if [ "${exit_code}" -eq 0 ]; then
  rm -f "${tmp_error_file}"
  exit "${exit_code}"
fi

# If it is a timeout error (see 'timeout --help')
if [ "${timeout}" -gt 0 ] && [ "${exit_code}" -eq 124 ]; then
  echo "Warning: '${command[*]}' timed out after ${timeout} seconds" | tee -a "${tmp_error_file}" >&2
fi

# Remove blank lines from begin and end of a file
sed -i -e '/./,$!d' -e :a -e '/^\n*$/{$d;N;ba' -e '}' "${tmp_error_file}"

# If stderr is empty
if [ ! -s "${tmp_error_file}" ]; then
  echo "Warning: '${command[*]}' didn't return stderr output" | tee -a "${tmp_error_file}" >&2
fi

# Write to the log
{ printf "Warning: Command '%s' run as '%s' failed with exit code %s:\n" "${command[*]}" "${whoami}" "${exit_code}"; cat "${tmp_error_file}"; } >> "/tmp/ddev-log-stderr-${whoami}.txt"
rm -f "${tmp_error_file}"

exit "${exit_code}"
