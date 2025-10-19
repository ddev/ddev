package cmd

import (
	"embed"
)

//go:embed scripts/test_ddev.sh scripts/diagnose_ddev.sh
var bundledAssets embed.FS
