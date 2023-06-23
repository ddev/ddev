package globalconfigtypes

type RouterType = string

// Router Types
const (
	RouterTypeTraefik    RouterType = "traefik"
	RouterTypeNginxProxy RouterType = "nginx-proxy"
	RouterTypeDefault    RouterType = RouterTypeTraefik
)

// ValidRouterTypes is the list of valid router types
var ValidRouterTypes = map[RouterType]bool{
	RouterTypeTraefik:    true,
	RouterTypeNginxProxy: true,
}

// IsValidRouterType limits the choices for Router Type
func IsValidRouterType(router RouterType) bool {
	isValid, ok := ValidRouterTypes[router]
	return ok && isValid
}

// GetValidRouterTypes returns a list of valid router types
func GetValidRouterTypes() []string {
	s := make([]string, 0, len(ValidRouterTypes))
	for p := range ValidRouterTypes {
		s = append(s, p)
	}
	return s
}
