package types

import "github.com/ddev/ddev/pkg/nodeps"

type RouterType = string

// Router Types
const (
	RouterTypeTraefik RouterType = "traefik"
)

// validRouterTypes is the list of valid router types
var validRouterTypes = []RouterType{
	RouterTypeTraefik,
}

// IsValidRouterType limits the choices for Router Type
func IsValidRouterType(router RouterType) bool {
	return nodeps.ArrayContainsString(validRouterTypes, router)
}

// GetValidRouterTypes returns a list of valid router types
func GetValidRouterTypes() []RouterType {
	return validRouterTypes
}
