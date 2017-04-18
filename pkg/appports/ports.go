package appports

import (
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Define the DBA and MailHog ports as variables so that we can override them with ldflags if required.

// DBAPort defines the default phpmyadmin port used on the router.
var dbaPort = "8036"

// mailHogPort defines the default mailhog exposed by the router.
var mailhogPort = "8025"

var ports = map[string]string{
	"mailhog": mailhogPort,
	"dba":     dbaPort,
}

// GetPort returns the external router (as a string) for the given service. This can be used to find a given port for docker-compose manifests,
// or for automated testing.
func GetPort(service string) string {
	service = strings.ToLower(service)
	val, ok := ports[service]
	if !ok {
		log.Fatalf("Could not find port for service %s", service)
	}

	return val
}
