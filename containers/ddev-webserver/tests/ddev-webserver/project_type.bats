#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

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

@test "verify legacy drush command resolves from user-owned bin directory ($project_type)" {
  if [ "$project_type" = "drupal6" ] || [ "$project_type" = "drupal7" ] || [ "$project_type" = "backdrop" ]; then
    run docker exec -t "$CONTAINER_NAME" bash -c 'test "$(command -v drush)" = "$HOME/.local/bin/drush"'
    assert_success
    run docker exec -t "$CONTAINER_NAME" bash -c 'test -L "$HOME/.local/bin/drush"'
    assert_success
  else
    run docker exec -t "$CONTAINER_NAME" bash -c 'test -e "$HOME/.local/bin/drush"'
    assert_failure
  fi
}
