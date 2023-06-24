package types

type RouterType = string

// Router Types
const (
	RouterTypeTraefik    RouterType = "traefik"
	RouterTypeNginxProxy RouterType = "nginx-proxy"
	RouterTypeDefault    RouterType = RouterTypeTraefik
)

// validRouterTypes is the list of valid router types
var validRouterTypes = map[RouterType]bool{
	RouterTypeTraefik:    true,
	RouterTypeNginxProxy: true,
}

// IsValidRouterType limits the choices for Router Type
func IsValidRouterType(router RouterType) bool {
	isValid, ok := validRouterTypes[router]
	return ok && isValid
}

// GetValidRouterTypes returns a list of valid router types
func GetValidRouterTypes() []string {
	s := make([]string, 0, len(validRouterTypes))
	for p := range validRouterTypes {
		s = append(s, p)
	}
	return s
}
