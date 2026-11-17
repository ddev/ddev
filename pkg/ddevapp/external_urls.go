package ddevapp

import (
	"fmt"
	"strconv"
	"strings"
)

// externalURLPortPlaceholder is replaced by a router HTTP port in external_url_template.
const externalURLPortPlaceholder = "{{port}}"

// HasExternalURLs reports whether the project maps router ports to external URLs,
// as in a cloud workspace whose proxy forwards each port under its own hostname.
func (app *DdevApp) HasExternalURLs() bool {
	return app.ExternalURLTemplate != "" || len(app.ExternalURLs) > 0
}

// GetExternalURL returns the URL a browser outside the workspace uses to reach
// the given router HTTP port, or "" when none is configured.
// An external_urls entry for the port wins over external_url_template.
func (app *DdevApp) GetExternalURL(routerHTTPPort string) string {
	if routerHTTPPort == "" || routerHTTPPort == "0" {
		return ""
	}
	if u := app.ExternalURLs[routerHTTPPort]; u != "" {
		return u
	}
	if app.ExternalURLTemplate == "" {
		return ""
	}
	return strings.ReplaceAll(app.ExternalURLTemplate, externalURLPortPlaceholder, routerHTTPPort)
}

// GetPrimaryExternalURL returns the external URL of the primary web app, or "".
func (app *DdevApp) GetPrimaryExternalURL() string {
	if !app.HasExternalURLs() {
		return ""
	}
	if IsRouterDisabled(app) {
		port, err := app.GetWebContainerDirectHTTPPort()
		if err != nil {
			return ""
		}
		return app.GetExternalURL(strconv.Itoa(port))
	}
	return app.GetExternalURL(app.GetPrimaryRouterHTTPPort())
}

// GetPrimaryDisplayURL returns the URL to show a person for the project:
// the external URL when one is configured, otherwise the primary URL.
// Code that runs inside the workspace or containers should keep GetPrimaryURL.
func (app *DdevApp) GetPrimaryDisplayURL() string {
	if u := app.GetPrimaryExternalURL(); u != "" {
		return u
	}
	return app.GetPrimaryURL()
}

// GetExternalURLMap maps every router port the project uses, HTTP and HTTPS,
// to its external URL. An HTTPS port maps to the URL of its paired HTTP port,
// because the external proxy terminates TLS and forwards to the HTTP port.
func (app *DdevApp) GetExternalURLMap() map[string]string {
	m := map[string]string{}
	if !app.HasExternalURLs() || IsRouterDisabled(app) {
		return m
	}
	add := func(httpPort, httpsPort string) {
		u := app.GetExternalURL(httpPort)
		if u == "" {
			return
		}
		m[httpPort] = u
		if _, ok := m[httpsPort]; !ok && httpsPort != "" && httpsPort != "0" {
			m[httpsPort] = u
		}
	}
	add(app.GetPrimaryRouterHTTPPort(), app.GetPrimaryRouterHTTPSPort())
	add(app.GetMailpitHTTPPort(), app.GetMailpitHTTPSPort())
	add(app.GetXHGuiHTTPPort(), app.GetXHGuiHTTPSPort())
	for _, p := range app.WebExtraExposedPorts {
		add(strconv.Itoa(p.HTTPPort), strconv.Itoa(p.HTTPSPort))
	}
	if app.ComposeYaml != nil {
		for name := range app.ComposeYaml.Services {
			for _, pair := range app.serviceRouterPorts(name) {
				add(pair[0], pair[1])
			}
		}
	}
	return m
}

// serviceExternalURL returns the external URL for the first HTTP_EXPOSE port
// of a compose service, the same port describe shows its router URL for.
func (app *DdevApp) serviceExternalURL(serviceName string) string {
	if serviceName == "web" {
		return app.GetPrimaryExternalURL()
	}
	pairs := app.serviceRouterPorts(serviceName)
	if len(pairs) == 0 || IsRouterDisabled(app) {
		return ""
	}
	return app.GetExternalURL(pairs[0][0])
}

// serviceRouterPorts returns the [HTTP, HTTPS] router port pairs a compose service
// declares in HTTP_EXPOSE, paired with HTTPS_EXPOSE by container port.
func (app *DdevApp) serviceRouterPorts(serviceName string) [][2]string {
	if app.ComposeYaml == nil {
		return nil
	}
	svc, ok := app.ComposeYaml.Services[serviceName]
	if !ok {
		return nil
	}
	env := func(name string) string {
		if v, ok := svc.Environment[name]; ok && v != nil {
			return *v
		}
		return ""
	}
	httpsExpose := env("HTTPS_EXPOSE")
	var pairs [][2]string
	for mapping := range strings.SplitSeq(env("HTTP_EXPOSE"), ",") {
		parts := strings.Split(strings.TrimSpace(mapping), ":")
		if len(parts) != 2 || parts[0] == "" {
			continue
		}
		pairs = append(pairs, [2]string{parts[0], app.TargetPortFromExposeVariable(httpsExpose, parts[1])})
	}
	return pairs
}

// validateExternalURLs checks external_url_template and external_urls.
func (app *DdevApp) validateExternalURLs() error {
	isHTTP := func(u string) bool {
		return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
	}
	if t := app.ExternalURLTemplate; t != "" {
		if !isHTTP(t) || !strings.Contains(t, externalURLPortPlaceholder) {
			return fmt.Errorf("external_url_template %q must start with http:// or https:// and contain %s", t, externalURLPortPlaceholder)
		}
	}
	for port, u := range app.ExternalURLs {
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("external_urls key %q must be a router port number", port)
		}
		if !isHTTP(u) {
			return fmt.Errorf("external_urls value %q for port %s must start with http:// or https://", u, port)
		}
	}
	return nil
}
