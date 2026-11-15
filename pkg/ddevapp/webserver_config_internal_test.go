package ddevapp

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// nginxLocation is one location block directly inside the server block.
type nginxLocation struct {
	line  int
	regex bool
	match string
	body  string
}

// TestNginxDenyRulesPrecedePHPLocation checks that every regex location that
// denies access comes before the regex location that passes .php requests to
// php-fpm. nginx uses the first regex location that matches, so a deny rule
// after the PHP location never applies to .php files, see ddev/ddev#9127.
func TestNginxDenyRulesPrecedePHPLocation(t *testing.T) {
	files, err := fs.Glob(webserverConfigAssets, "webserver_config_assets/nginx*.conf")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		name := path.Base(file) //nolint:forbidigo // embed path
		t.Run(name, func(t *testing.T) {
			content, err := webserverConfigAssets.ReadFile(file)
			require.NoError(t, err)
			locations := parseNginxServerLocations(string(content))

			phpIndex := -1
			for i, loc := range locations {
				if loc.regex && strings.Contains(loc.body, "fastcgi_pass") {
					phpIndex = i
					break
				}
			}
			require.NotEqual(t, -1, phpIndex, "%s has no regex location that runs PHP", name)

			for _, loc := range locations[phpIndex+1:] {
				if !loc.regex || !isNginxDenyBody(loc.body) {
					continue
				}
				require.Failf(t, "deny rule after the PHP location",
					"%s line %d: location %q denies access but comes after the PHP location on line %d, so it never applies to .php requests. Move it above the PHP location.",
					name, loc.line, loc.match, locations[phpIndex].line)
			}
		})
	}
}

// isNginxDenyBody reports whether a location body refuses the request.
func isNginxDenyBody(body string) bool {
	return strings.Contains(body, "deny all;") || strings.Contains(body, "return 403;")
}

// parseNginxServerLocations returns the location blocks directly inside the
// first server block, ignoring comments and nested locations.
func parseNginxServerLocations(content string) []nginxLocation {
	var locations []nginxLocation
	var current *nginxLocation
	depth := 0
	inServer := false
	for i, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !inServer && strings.HasPrefix(line, "server ") && strings.HasSuffix(line, "{") {
			inServer = true
			depth = 1
			continue
		}
		if !inServer {
			continue
		}
		if depth == 1 && strings.HasPrefix(line, "location ") && strings.HasSuffix(line, "{") {
			fields := strings.Fields(strings.TrimSuffix(line, "{"))
			loc := nginxLocation{line: i + 1}
			if len(fields) >= 3 && (fields[1] == "~" || fields[1] == "~*") {
				loc.regex = true
				loc.match = strings.Trim(strings.Join(fields[2:], " "), "'\"")
			} else {
				loc.match = strings.Join(fields[1:], " ")
			}
			locations = append(locations, loc)
			current = &locations[len(locations)-1]
		} else if current != nil {
			current.body += line + "\n"
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth == 1 {
			current = nil
		}
		if depth == 0 {
			break
		}
	}
	return locations
}
