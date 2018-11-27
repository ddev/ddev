package main

import (
	"github.com/drud/ddev/cmd/ddev/cmd"
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/getsentry/raven-go"
)

func main() {
	cmd.Execute()
}

func init() {
	raven.SetDSN("https://ad3abb1deb8447398c5a2ad8f4287fad:70e11b442a9243719f150e4d922cfde6@sentry.io/160826")

	ddevapp.SetRavenBaseTags()
}
