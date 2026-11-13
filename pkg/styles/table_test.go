package styles

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFitColumnWidths(t *testing.T) {
	tests := []struct {
		name      string
		natural   []int
		available int
		want      []int
	}{
		{"everything fits", []int{12, 7, 30, 22, 8}, 100, []int{12, 7, 30, 22, 8}},
		{"widest columns capped evenly", []int{45, 7, 69, 22, 8}, 107, []int{35, 7, 35, 22, 8}},
		{"short columns keep their width", []int{20, 7, 69, 22, 8}, 83, []int{20, 7, 26, 22, 8}},
		{"all wide columns shrink together", []int{45, 7, 69, 22, 8}, 63, []int{16, 7, 16, 16, 8}},
		{"minimum width when nothing fits", []int{45, 7, 69, 40, 8}, 20, []int{7, 7, 7, 7, 7}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, FitColumnWidths(tc.natural, tc.available))
		})
	}
}

func TestTableRender(t *testing.T) {
	link := "\x1b]8;;https://example.ddev.site\x1b\\https://example.ddev.site\x1b]8;;\x1b\\"
	tbl := NewTable("Name", "Location", "URL")
	tbl.SetSnip(1)
	tbl.AppendRow("a-long-project-name", "~/workspace/some/deep/path", link)
	tbl.AppendRow("b", "~/b")
	tbl.SetFooter("Router", "~/.ddev/some/long/path\nport 80 is already in use", "line one\nline two")
	tbl.SetMaxWidths(10, 12, 12)
	out := tbl.Render()

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, line := range lines {
		require.Equal(t, ansi.StringWidth(lines[0]), ansi.StringWidth(line), "every line is as wide as the first:\n%s", out)
	}
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "~/workspace…", "a one-line path is snipped")
	require.NotContains(t, out, "deep/path")
	require.Contains(t, out, "https://exa…", "a link is snipped, not wrapped")
	require.Equal(t, 1, strings.Count(out, "\x1b]8;;https://example.ddev.site"), "the link is not split across lines")
	require.Contains(t, out, "~/.ddev/som…", "a path in a multi-line cell is snipped")
	require.Contains(t, out, "port 80 is", "text with spaces wraps")
	require.Contains(t, out, "line one")
	require.Contains(t, out, "line two")
	require.Contains(t, out, "a-long-")
}

func TestTableUnlimited(t *testing.T) {
	tbl := NewTable("Name")
	tbl.AppendRow("a-long-project-name")
	tbl.SetMaxWidths(5)
	tbl.SetUnlimited(true)
	require.Contains(t, tbl.Render(), "a-long-project-name")
}
