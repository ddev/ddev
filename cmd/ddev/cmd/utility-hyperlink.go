package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// UtilityHyperlinkCmd lets host scripts print URLs as OSC 8 hyperlinks using
// the same terminal detection as the rest of DDEV. Scripts pipe text into it
// rather than capturing its output, because detection needs stdout to be the
// terminal.
var UtilityHyperlinkCmd = &cobra.Command{
	Use:         "hyperlink [url [text]]",
	Annotations: map[string]string{NoDockerCommand: "true"},
	Short:       "Print a URL, or link the URLs in stdin, as terminal hyperlinks when supported",
	Hidden:      true,
	Args:        cobra.MaximumNArgs(2),
	Run: func(_ *cobra.Command, args []string) {
		if len(args) > 0 {
			linkText := args[0]
			if len(args) > 1 {
				linkText = args[1]
			}
			fmt.Println(output.Hyperlink(args[0], linkText))
			return
		}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Println(output.LinkifyURLs(scanner.Text()))
		}
	},
}

func registerUtilityHyperlinkCmd() {
	DebugCmd.AddCommand(UtilityHyperlinkCmd)
}
