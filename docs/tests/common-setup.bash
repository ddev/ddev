#!/usr/bin/env bash

_common_setup() {
    # Allows `run -0`, which reports the command and its exit code on failure
    bats_require_minimum_version 1.5.0
    TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
    export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
    bats_load_library bats-support
    bats_load_library bats-assert
    bats_load_library bats-file
    # Skip this test if it matches DDEV_EMBARGO_TESTS.
    # Values are separated by "|" or by line endings. Each value is matched
    # as a substring against:
    #   1. The bats file name without ".bats", e.g. "sveltekit"
    #   2. The @test description ($BATS_TEST_DESCRIPTION)
    if [ -n "${DDEV_EMBARGO_TESTS:-}" ]; then
        local _bats_basename _embargo_id _embargo_list
        _bats_basename=$(basename "${BATS_TEST_FILENAME:-}" .bats)
        _embargo_list=${DDEV_EMBARGO_TESTS//[$'\r\n']/|}
        IFS='|' read -ra _embargo_ids <<< "${_embargo_list}"
        for _embargo_id in "${_embargo_ids[@]}"; do
            # Remove spaces at both ends of the value
            read -r _embargo_id <<< "$_embargo_id"
            [ -z "$_embargo_id" ] && continue
            if [[ "$_bats_basename" == *"$_embargo_id"* ]] \
              || [[ "${BATS_TEST_DESCRIPTION:-}" == *"$_embargo_id"* ]]; then
                skip "Auto-skipped by DDEV_EMBARGO_TESTS match: ${_embargo_id}"
            fi
        done
    fi
    mkdir -p ~/tmp
    tmpdir=$(mktemp -d ~/tmp/${PROJNAME}.XXXXXX)
    export DDEV_NO_INSTRUMENTATION=true
    export DDEV_NONINTERACTIVE=true
    mkdir -p ${tmpdir} && cd ${tmpdir} || exit 1
    ddev delete -Oy ${PROJNAME:-notset} >/dev/null 2>&1
#    echo "# Starting test at $(date)" >&3
}

# Populates URL variables from `ddev describe -j`, so tests don't have to
# hand-build URLs and can stay correct under a custom router_http(s)_port or TLD.
_extra_info() {
  local json
  json=$(ddev describe -j ${PROJNAME})
  HOST_HTTP_URL=$(echo "${json}" | jq -r .raw.services.web.host_http_url)
  HOST_HTTPS_URL=$(echo "${json}" | jq -r .raw.services.web.host_https_url)
  PRIMARY_HTTP_URL=$(echo "${json}" | jq -r .raw.httpurl)
  PRIMARY_HTTPS_URL=$(echo "${json}" | jq -r .raw.httpsurl)
  PRIMARY_URL=$(echo "${json}" | jq -r .raw.primary_url)
  DDEV_HOSTNAME=$(echo "${json}" | jq -r .raw.hostname)
  PRIMARY_URL_ENCODED=$(printf '%s' "${PRIMARY_URL}" | jq -sRr '@uri')
}

_common_teardown() {
#  echo "# Ending test at $(date)" >&3
  ddev delete -Oy ${PROJNAME} >/dev/null 2>&1
  rm -rf ${tmpdir}
}

# Print the first GitHub token set, in the order ddev itself checks them
_github_token() {
  echo "${DDEV_GITHUB_TOKEN:-${GH_TOKEN:-${GITHUB_TOKEN:-}}}"
}

# Curl wrapper that adds GitHub token auth if available
# Usage: _curl_github [curl options] <url>
_curl_github() {
  local auth_args=() token
  token=$(_github_token)
  [ -n "${token}" ] && auth_args=(-H "Authorization: Bearer ${token}")
  curl --retry 5 --retry-delay 2 "${auth_args[@]}" "$@"
}

# Run a command up to 3 times, for steps that download from upstream hosts
# whose outages (GitHub 5xx and 429, packagist timeouts) outlast the tool's own retries.
# Usage: run -0 _retry <command> [args...]
_retry() {
  local attempt
  for attempt in 1 2 3; do
    "$@" && return 0
    [ "${attempt}" -lt 3 ] || return 1
    echo "# Attempt ${attempt} of '$*' failed, retrying in $((attempt * 15))s" >&3
    sleep $((attempt * 15))
  done
}

# Fetch a page and require each string in its body. A failure reports curl's
# error, or the missing string and the page title, rather than the whole page.
# Usage: _assert_page_contains [curl options] <url> <string>...
_assert_page_contains() {
  local curl_args=() url body s
  while [[ "${1:-}" == -* ]]; do curl_args+=("$1"); shift; done
  url=$1; shift
  if ! body=$(curl -sSf "${curl_args[@]}" "${url}" 2>&1); then
    echo "Fetching ${url} failed: ${body}" >&2
    return 1
  fi
  for s in "$@"; do
    if [[ "${body}" != *"${s}"* ]]; then
      echo "Page ${url} does not contain: ${s}" >&2
      echo "Page title: $(grep -o '<title>[^<]*</title>' <<<"${body}" | head -1)" >&2
      return 1
    fi
  done
}

# Download the latest release asset from a GitHub repo
# Usage: _github_release_download <owner/repo> <asset_regex> <output_file>
# Example: _github_release_download "ExpressionEngine/ExpressionEngine" "^ExpressionEngine.*\\.zip$" "ee.zip"
# Authenticates with _github_token if one is available
_github_release_download() {
  local repo="$1"
  local asset_pattern="$2"
  local output_file="$3"

  # Get latest release info with retries
  local release_json
  release_json=$(_curl_github -sfL "https://api.github.com/repos/${repo}/releases/latest")

  if [ -z "${release_json}" ]; then
    echo "# Failed to fetch release info from ${repo}" >&3
    return 1
  fi

  # Extract download URL - escape backslashes for jq regex
  local download_url
  local escaped_pattern="${asset_pattern//\\/\\\\}"
  download_url=$(echo "${release_json}" | docker run -i --rm ddev/ddev-utilities jq -r ".assets | map(select(.name | test(\"${escaped_pattern}\")))[0].browser_download_url")

  if [ -z "${download_url}" ] || [ "${download_url}" = "null" ]; then
    echo "# No asset matching '${asset_pattern}' found in ${repo}" >&3
    return 1
  fi

  echo "# Downloading ${download_url}" >&3

  # Download the asset
  _curl_github -fL -o "${output_file}" "${download_url}"
}
