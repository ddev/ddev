package types

type RouterType = string

const (
	RouterTypeTraefik    RouterType = "traefik"
	RouterTypeNginxProxy RouterType = "nginx-proxy"
)
