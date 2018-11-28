package main

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
)

func main() {
	cmd.Execute()
}

func init() {
	_ = raven.SetDSN(version.SentryDSN)
	ddevapp.SetRavenBaseTags()
}
