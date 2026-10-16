package ddevapp

import (
	"os/exec"
	"testing"

	"github.com/ddev/ddev/pkg/util"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderInfoField tests that the InfoCommand field exists in ProviderInfo
func TestProviderInfoField(t *testing.T) {
	assert := asrt.New(t)

	// Test that InfoCommand field is properly defined in ProviderInfo
	p := Provider{
		InfoCommand: ProviderCommand{
			Command: "echo test",
			Service: "web",
		},
	}

	assert.Equal("echo test", p.InfoCommand.Command)
	assert.Equal("web", p.InfoCommand.Service)
}

// TestProviderInjectedEnvironment tests that provider values with shell
// characters reach the command intact while $VAR and ~ still expand
func TestProviderInjectedEnvironment(t *testing.T) {
	require.Equal(t, "true", (&Provider{}).injectedEnvironment())

	t.Setenv("DDEV_TEST_PROVIDER_VAR", "site")
	p := &Provider{EnvironmentVariables: map[string]string{
		"project_id":   "abc) def; it's",
		"token":        `p@ss"word\`,
		"environment":  "${DDEV_TEST_PROVIDER_VAR}-main",
		"checkout_dir": "~/tmp/repo",
	}}
	script := p.injectedEnvironment() + `; printf '%s|%s|%s|%s' "$project_id" "$token" "$environment" "$checkout_dir"`
	out, err := exec.Command(util.FindBashPath(), "-c", script).Output()
	require.NoError(t, err, "script=%s", script)
	home, err := exec.Command(util.FindBashPath(), "-c", `printf '%s' "$HOME"`).Output()
	require.NoError(t, err)
	require.Equal(t, `abc) def; it's|p@ss"word\|site-main|`+string(home)+"/tmp/repo", string(out))
}
