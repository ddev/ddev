package ddevapp

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestConfirmSuggestedConfig checks the answers to "Use this configuration?",
// including input piped for the old questions, which must fail rather than
// take the default.
func TestConfirmSuggestedConfig(t *testing.T) {
	t.Cleanup(func() { util.SetInputScanner(bufio.NewScanner(os.Stdin)) })

	tests := []struct {
		input   string
		want    bool
		wantErr bool
	}{
		{input: "\n", want: true},
		{input: "", want: true},
		{input: "y\n", want: true},
		{input: "No\n", want: false},
		{input: "maybe\nno\n", want: false},
		{input: "maybe\n\n", want: true},
		{input: "myname\nweb\ndrupal10\n", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(strings.ReplaceAll(tc.input, "\n", `\n`), func(t *testing.T) {
			util.SetInputScanner(bufio.NewScanner(strings.NewReader(tc.input)))
			got, err := confirmSuggestedConfig()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
