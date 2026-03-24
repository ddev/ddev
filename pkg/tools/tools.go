//go:build tools

// This file uses blank imports to keep new dependencies in go.mod and vendor
// without affecting normal builds. go mod tidy respects these imports and will
// not remove the dependencies. This allows splitting large PRs into two parts
// for easier review: one PR for vendor dependency updates, and a separate PR
// for the logic that uses them. Once the real code lands, remove the
// corresponding imports from this file.
// See https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

// Example:
//
//import (
//	_ "github.com/some/new/library"
//	_ "github.com/some/new/library/pkg/api"
//)

// For new Docker Compose SDK https://github.com/ddev/ddev/pull/8234
import (
	_ "github.com/docker/cli/cli"
	_ "github.com/docker/cli/cli/streams"
	_ "github.com/docker/compose/v5/cmd/display"
	_ "github.com/docker/compose/v5/pkg/api"
	_ "github.com/docker/compose/v5/pkg/compose"
)

// For Viper https://github.com/ddev/ddev/pull/8181
import (
	_ "github.com/spf13/viper"
)
