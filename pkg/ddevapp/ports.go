package ddevapp

import (
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/util"
	"strings"
)

// DBAPort defines the default phpmyadmin port used on the router.
var dbaPort = "8036"

// mailHogPort defines the default mailhog exposed by the router.
var mailhogPort = "8025"

// dbPort defines the default DB (MySQL) port.
var dbPort = "3306"

// webPort defines the internal web port
var webPort = "80"

var ports = map[string]string{
	"mailhog": mailhogPort,
	"dba":     dbaPort,
	"db":      dbPort,
	"web":     webPort,
}

// GetPort returns the external router port (as a string) for the given service.
// This can be used to find a given port for docker-compose manifests,
// or for automated testing.
func GetPort(app *DdevApp, service string) string {
	if app.Database.Type == nodeps.Postgres {
		ports["db"] = "5432"
	}
	service = strings.ToLower(service)
	val, ok := ports[service]
	if !ok {
		util.Failed("Could not find port for service %s", service)
	}

	return val
}
