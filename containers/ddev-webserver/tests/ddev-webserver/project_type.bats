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
	if [ "$project_type" = "backdrop" ] ; then
	 	# The .drush/commands/backdrop directory should only exist for backdrop apptype
		docker exec -t $CONTAINER_NAME bash -c 'if [ ! -d  ~/.drush/commands/backdrop ] ; then echo "Failed to find expected backdrop drush commands"; exit 106; fi'
	else
		docker exec -t $CONTAINER_NAME bash -c 'if [ -d  ~/.drush/commands/backdrop ] ; then echo "Found unexpected backdrop drush commands"; exit 107; fi'
  fi
}
