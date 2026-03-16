package dockerutil

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/compose/v5/cmd/display"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestSanitizeServiceNameEdges covers empty input, leading/trailing hyphens, runs of separators.
func TestSanitizeServiceNameEdges(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"simple", "simple"},
		{"hello/world", "hello-world"},
		{"---leading", "leading"},
		{"trailing---", "trailing"},
		{"a///b", "a-b"},
		{"docker.io/library/alpine:3.18", "docker.io-library-alpine-3.18"},
		{"_underscore_", "_underscore_"},
	}
	for _, tt := range tests {
		got := sanitizeServiceName(tt.input)
		require.Equal(t, tt.want, got, "input=%q", tt.input)
	}
}

// TestSanitizeServiceNameCollisionMatrix pins inputs that DO collide vs
// inputs that look similar but DO NOT, so PullImages's "skip on collision"
// fallback in docker_compose.go is exercised against a known-stable set.
//
// The collision pairs document a real footgun: `a/b/c` and `a-b-c` collapse
// to the same sanitized form. DDEV's image set keeps registry+tag distinct
// per image, so collisions don't occur in practice — but a future addition
// could regress this silently. If this test starts failing, audit the new
// image set rather than relaxing the assertion.
func TestSanitizeServiceNameCollisionMatrix(t *testing.T) {
	colliding := []struct {
		a, b string
	}{
		{"a/b/c", "a-b-c"},
		{"docker.io/library/alpine", "docker.io-library-alpine"},
		{"foo:bar", "foo-bar"},
		{"foo bar", "foo-bar"},
	}
	for _, c := range colliding {
		require.Equal(t, sanitizeServiceName(c.a), sanitizeServiceName(c.b),
			"expected %q and %q to collide", c.a, c.b)
	}

	distinct := []struct {
		a, b string
	}{
		// Different tags must remain distinct (DDEV relies on this).
		{"docker.io/library/alpine:3.18", "docker.io/library/alpine:3.19"},
		// Different repositories on the same registry remain distinct.
		{"docker.io/library/alpine", "docker.io/library/busybox"},
		// Different registries remain distinct.
		{"ghcr.io/foo/bar", "docker.io/foo/bar"},
	}
	for _, d := range distinct {
		require.NotEqual(t, sanitizeServiceName(d.a), sanitizeServiceName(d.b),
			"expected %q and %q to remain distinct after sanitization", d.a, d.b)
	}
}

// TestProgressOptsPlainWriterRouting verifies that display.ModePlain routes
// EventProcessor output to a buffer wired through progressOpts.
//
// We exercise display.Plain directly with a known buffer and feed it a
// resource event; this is the same construction path progressOpts takes.
// Failure here would indicate that compose's display.Plain has changed
// behavior and DDEV's progress output is no longer reaching its expected
// destination.
func TestProgressOptsPlainWriterRouting(t *testing.T) {
	var buf bytes.Buffer

	ep := display.Plain(&buf)
	require.NotNil(t, ep)

	ep.On(api.Resource{
		ID:     "ddev-progress-routing-test",
		Status: api.Working,
		Text:   "routing-check",
	})

	require.Contains(t, buf.String(), "routing-check",
		"display.Plain must write events to the writer it was constructed with")
}

// TestSuppressLogrusFormatterDirect exercises the formatter installed at
// package init by feeding it entries directly. This is the deterministic
// counterpart to TestSuppressedLogrusNoise (which goes through the global
// logger) and is robust against parallel tests touching logrus.
func TestSuppressLogrusFormatterDirect(t *testing.T) {
	f, ok := logrus.StandardLogger().Formatter.(*suppressLogrusFormatter)
	require.True(t, ok, "package init must install suppressLogrusFormatter on logrus.StandardLogger")

	// Matching message: nil bytes, no error.
	out, err := f.Format(&logrus.Entry{Message: "Warning: No resource found to remove for project \"foo\"."})
	require.NoError(t, err)
	require.Empty(t, out)

	// Matching substring at any position is suppressed.
	out, err = f.Format(&logrus.Entry{Message: "prefix No resource found to remove suffix"})
	require.NoError(t, err)
	require.Empty(t, out)

	// Non-matching message: underlying formatter runs and produces output.
	out, err = f.Format(&logrus.Entry{
		Message: "real warning",
		Level:   logrus.WarnLevel,
		Time:    time.Unix(0, 0),
	})
	require.NoError(t, err)
	require.NotEmpty(t, out)
	require.Contains(t, string(out), "real warning")
}

// TestSetupLogrusSuppressionIdempotent verifies that calling setupLogrusSuppression
// twice does not stack wrappers — a real concern if init runs more than once
// (e.g. via test reordering) or if a future caller invokes it explicitly.
func TestSetupLogrusSuppressionIdempotent(t *testing.T) {
	logger := logrus.New()
	originalFormatter := logger.Formatter

	setupLogrusSuppression(logger)
	first, ok := logger.Formatter.(*suppressLogrusFormatter)
	require.True(t, ok)
	require.Same(t, originalFormatter, first.underlying,
		"first install must wrap the original formatter")

	setupLogrusSuppression(logger)
	second, ok := logger.Formatter.(*suppressLogrusFormatter)
	require.True(t, ok)
	require.Same(t, first, second,
		"second install must be a no-op; otherwise we'd double-wrap and lose the original formatter")
}
