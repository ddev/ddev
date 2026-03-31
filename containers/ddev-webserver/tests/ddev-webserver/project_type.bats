#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

# Requires bats-assert and bats-support
# brew tap bats-core/bats-core &&
# brew install bats-core bats-assert bats-support jq mkcert yq
setup() {
  load setup.sh
}

@test "verify that backdrop drush commands were added on backdrop and only backdrop ($project_type)" {
  if [ "$project_type" = "backdrop" ]; then
    # The .drush/commands/backdrop directory should only exist for backdrop apptype
    run docker exec -t $CONTAINER_NAME bash -c 'test -d ~/.drush/commands/backdrop'
    assert_success
  else
    run docker exec -t $CONTAINER_NAME bash -c 'test -d ~/.drush/commands/backdrop'
    assert_failure
  fi
}
