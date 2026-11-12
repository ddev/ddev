package ddevapp

import (
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/require"
)

func TestGetExternalURL(t *testing.T) {
	app := &DdevApp{}
	require.False(t, app.HasExternalURLs())
	require.Empty(t, app.GetExternalURL("8080"))
	require.Empty(t, app.GetPrimaryExternalURL())

	app.ExternalURLTemplate = "https://{{port}}--main--ws--me.example.com"
	app.ExternalURLs = map[string]string{"8080": "https://proj--ws--me.example.com"}
	require.True(t, app.HasExternalURLs())
	require.Equal(t, "https://8036--main--ws--me.example.com", app.GetExternalURL("8036"))
	require.Equal(t, "https://proj--ws--me.example.com", app.GetExternalURL("8080"))
	require.Empty(t, app.GetExternalURL(""))
	require.Empty(t, app.GetExternalURL("0"))

	app.ExternalURLTemplate = ""
	require.Empty(t, app.GetExternalURL("8036"))
}

func TestGetExternalURLMap(t *testing.T) {
	httpExpose := "8036:80"
	httpsExpose := "8037:80"
	app := &DdevApp{
		Name:                 "exturls",
		RouterHTTPPort:       "8080",
		RouterHTTPSPort:      "8443",
		MailpitHTTPPort:      "8025",
		MailpitHTTPSPort:     "8026",
		WebExtraExposedPorts: []WebExposedPort{{Name: "vite", WebContainerPort: 5173, HTTPPort: 5172, HTTPSPort: 5173}},
		ExternalURLTemplate:  "https://{{port}}--main--ws--me.example.com",
		ExternalURLs:         map[string]string{"8080": "https://exturls--ws--me.example.com"},
		ComposeYaml: &composeTypes.Project{Services: composeTypes.Services{
			"phpmyadmin": {Name: "phpmyadmin", Environment: composeTypes.MappingWithEquals{
				"HTTP_EXPOSE":  &httpExpose,
				"HTTPS_EXPOSE": &httpsExpose,
			}},
		}},
	}
	m := app.GetExternalURLMap()
	require.Equal(t, "https://exturls--ws--me.example.com", m["8080"])
	require.Equal(t, "https://exturls--ws--me.example.com", m["8443"])
	require.Equal(t, "https://8025--main--ws--me.example.com", m["8026"])
	require.Equal(t, "https://5172--main--ws--me.example.com", m["5173"])
	require.Equal(t, "https://8036--main--ws--me.example.com", m["8037"])
	require.Equal(t, "https://8036--main--ws--me.example.com", app.serviceExternalURL("phpmyadmin"))
	require.Equal(t, "https://exturls--ws--me.example.com", app.serviceExternalURL("web"))
	require.Equal(t, "https://exturls--ws--me.example.com", app.GetPrimaryDisplayURL())
}

func TestValidateExternalURLs(t *testing.T) {
	for _, tc := range []struct {
		template string
		urls     map[string]string
		ok       bool
	}{
		{"", nil, true},
		{"https://{{port}}--main.example.com", map[string]string{"8080": "https://a.example.com"}, true},
		{"https://main.example.com", nil, false},
		{"{{port}}.example.com", nil, false},
		{"", map[string]string{"web": "https://a.example.com"}, false},
		{"", map[string]string{"8080": "a.example.com"}, false},
	} {
		app := &DdevApp{ExternalURLTemplate: tc.template, ExternalURLs: tc.urls}
		err := app.validateExternalURLs()
		if tc.ok {
			require.NoError(t, err, tc)
		} else {
			require.Error(t, err, tc)
		}
	}
}
