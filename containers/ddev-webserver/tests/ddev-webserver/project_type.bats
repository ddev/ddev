#!/usr/bin/env bats

# Run these tests from the repo root directory, for example
# bats tests

setup() {
  load setup.sh
}

@test "verify that backdrop drush commands are not baked into the shared image ($project_type)" {
  # Drush 8 and the Backdrop Drush extension are now installed only in the
  # per-project derived image (pkg/ddevapp/project_tools.go), never in the
  # shared image this test runs against, regardless of DDEV_PROJECT_TYPE.
  run docker exec -t $CONTAINER_NAME bash -c 'test -d ~/.drush/commands/backdrop'
  assert_failure
}

@test "verify legacy drush command does not resolve in the shared image ($project_type)" {
  run docker exec -t "$CONTAINER_NAME" bash -c 'test -e "$HOME/.local/bin/drush"'
  assert_failure
}
