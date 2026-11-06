package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// postForm POSTs form values with the bearer token.
func postForm(s *Server, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", target, strings.NewReader(form.Encode()))
	req.Host = s.host()
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

// jobOutput waits for the job a 202 reply names and returns its output.
func jobOutput(t *testing.T, s *Server, rec *httptest.ResponseRecorder) string {
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	var summary jobSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
	j := s.jobs.get(summary.ID)
	require.NotNil(t, j)
	require.Eventually(t, func() bool { return j.summary().Done }, 10*time.Second, 50*time.Millisecond)
	require.Empty(t, j.summary().Err)
	lines, _, _, _, _ := j.since(0)
	return strings.Join(lines, "\n")
}

func TestAddProjectValidation(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "0")
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))

	for name, tc := range map[string]struct {
		form url.Values
		want string
	}{
		"empty dir":    {url.Values{}, "absolute path"},
		"relative dir": {url.Values{"dir": {"my-site"}}, "absolute path"},
		"missing dir":  {url.Values{"dir": {filepath.Join(dir, "nope")}}, "no such directory"},
		"file":         {url.Values{"dir": {file}}, "no such directory"},
		"unknown type": {url.Values{"dir": {dir}, "type": {"not-a-type"}}, "unknown project type"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := postForm(s, "/api/projects", tc.form)
			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), tc.want)
		})
	}
	require.Empty(t, s.jobs.summaries(), "no ddev config may run for a refused request")
}

func TestAddProjectRunsConfig(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "0")
	dir := t.TempDir()

	out := jobOutput(t, s, postForm(s, "/api/projects", url.Values{
		"dir":     {dir},
		"type":    {"php"},
		"name":    {" my-site "},
		"docroot": {"web"},
	}))
	require.Contains(t, out, "args: config --project-type=php --project-name=my-site --docroot=web")
	realDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.Contains(t, out, "dir: "+realDir)
}

func TestAddProjectExpandsHome(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "0")
	home := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(home, "site"), 0o755))
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	out := jobOutput(t, s, postForm(s, "/api/projects", url.Values{"dir": {"~/site"}}))
	require.Contains(t, out, "args: config\n", "no flags without a type, name, or docroot")
	realHome, err := filepath.EvalSymlinks(home)
	require.NoError(t, err)
	require.Contains(t, out, "dir: "+filepath.Join(realHome, "site"))
}

func TestFormArgRefusesFlags(t *testing.T) {
	for value, ok := range map[string]bool{
		"ddev/ddev-redis":  true,
		"  redis  ":        true,
		"":                 true,
		"--help":           false,
		" -y":              false,
		"-":                false,
		"snapshot-name-1x": true,
	} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(url.Values{"v": {value}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		got, gotOK := formArg(r, "v")
		require.Equal(t, ok, gotOK, "value %q", value)
		require.Equal(t, strings.TrimSpace(value), got)
	}
}

func TestUnknownProjectAndAction(t *testing.T) {
	s := newTestServer(t)
	fakeDdev(t, s, "0")

	rec := postForm(s, "/api/projects/no-such-project-webui-test/actions/start", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "no project named")

	rec = postForm(s, "/api/actions/nothing", nil)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, s.jobs.summaries())
}

func TestProjectTypes(t *testing.T) {
	s := newTestServer(t)
	rec := request(s, "GET", "/api/project-types", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+s.token) })
	require.Equal(t, http.StatusOK, rec.Code)
	var types []string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &types))
	require.Contains(t, types, "php")
	require.Contains(t, types, "drupal11")
}
