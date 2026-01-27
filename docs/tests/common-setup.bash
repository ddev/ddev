#!/usr/bin/env bash

_common_setup() {
    TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
    export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
    bats_load_library bats-support
    bats_load_library bats-assert
    bats_load_library bats-file
    mkdir -p ~/tmp
    tmpdir=$(mktemp -d ~/tmp/${PROJNAME}.XXXXXX)
    export DDEV_NO_INSTRUMENTATION=true
    export DDEV_NONINTERACTIVE=true
    mkdir -p ${tmpdir} && cd ${tmpdir} || exit 1
    ddev delete -Oy ${PROJNAME:-notset} >/dev/null
#    echo "# Starting test at $(date)" >&3
}

# Check if a test should be skipped based on DDEV_SKIP_QUICKSTART_TEST
# Set DDEV_SKIP_QUICKSTART_TEST to a comma-separated list of test identifiers to skip
# Examples:
#   DDEV_SKIP_QUICKSTART_TEST="symfony-composer" make quickstart-test
#   DDEV_SKIP_QUICKSTART_TEST="symfony-composer,symfony-cli,drupal10-composer" make quickstart-test
# Usage in test files: _skip_test_if_needed "test-identifier"
_skip_test_if_needed() {
    local test_id="$1"
    if [ -n "${DDEV_SKIP_QUICKSTART_TEST:-}" ]; then
        IFS=',' read -ra SKIP_TESTS <<< "${DDEV_SKIP_QUICKSTART_TEST}"
        local skip_id
        for skip_id in "${SKIP_TESTS[@]}"; do
            # Trim whitespace
            skip_id=$(echo "$skip_id" | xargs)
            if [ "$skip_id" = "$test_id" ]; then
                skip "Test skipped via DDEV_SKIP_QUICKSTART_TEST: ${test_id}"
            fi
        done
    fi
}

_extra_info() {
  HOST_HTTP_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.services.web.host_http_url)
  HOST_HTTPS_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.services.web.host_https_url)
  PRIMARY_HTTP_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.httpurl)
  PRIMARY_HTTPS_URL=$(ddev describe -j ${PROJNAME} | jq -r .raw.httpsurl)
}

_common_teardown() {
#  echo "# Ending test at $(date)" >&3
  ddev delete -Oy ${PROJNAME} >/dev/null
  rm -rf ${tmpdir}
}

# Curl wrapper that adds GITHUB_TOKEN auth if available
# Usage: _curl_github [curl options] <url>
_curl_github() {
  local auth_args=()
  [ -n "${GITHUB_TOKEN:-}" ] && auth_args=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
  curl --retry 5 --retry-delay 2 "${auth_args[@]}" "$@"
}

# Download the latest release asset from a GitHub repo
# Usage: _github_release_download <owner/repo> <asset_regex> <output_file>
# Example: _github_release_download "ExpressionEngine/ExpressionEngine" "^ExpressionEngine.*\\.zip$" "ee.zip"
# Uses GITHUB_TOKEN env var if available for authenticated requests
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
