package main

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
	"os"
)

func main() {
	cmd.Execute()
}

func init() {
	noSentry := os.Getenv("DDEV_NO_SENTRY")
	if noSentry == "" {
		_ = raven.SetDSN(version.SentryDSN)
		ddevapp.SetRavenBaseTags()
	}
}
