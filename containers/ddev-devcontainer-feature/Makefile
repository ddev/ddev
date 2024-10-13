VERSION := $(shell git describe --tags --always --dirty)

# Requires GITHUB_TOKEN with power to manage packages
push:
	devcontainer features publish -n ddev/ddev .