package output

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColorsForced(t *testing.T) {
	for _, c := range []struct {
		force, noColor string
		want           bool
	}{
		{"", "", false},
		{"0", "", false},
		{"1", "", true},
		{"1", "1", false},
	} {
		t.Setenv("CLICOLOR_FORCE", c.force)
		t.Setenv("NO_COLOR", c.noColor)
		require.Equal(t, c.want, ColorsForced(), "CLICOLOR_FORCE=%q NO_COLOR=%q", c.force, c.noColor)
	}
}
