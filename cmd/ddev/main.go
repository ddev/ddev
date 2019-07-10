package main

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/globalconfig"
	"github.com/drud/ddev/pkg/version"
	"github.com/getsentry/raven-go"
)

func main() {
	cmd.Execute()
}

func init() {
	if !globalconfig.DdevNoSentry {
		_ = raven.SetDSN(version.SentryDSN)
		ddevapp.SetInstrumentationBaseTags()
	}
}
