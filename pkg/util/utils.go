package util

import (
	"os"

	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uitable"
)

// Failed will print an red error message and exit with failure.
func Failed(format string, a ...interface{}) {
	color.Red(format, a...)
	os.Exit(1)
}

// Success will indicate an operation succeeded with colored confirmation text.
func Success(format string, a ...interface{}) {
	color.Cyan(format, a...)
}

// Warning will present the user with warning text.
func Warning(format string, a ...interface{}) {
	color.Yellow(format, a...)
}

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

// RenderAppTable will format a table for user display based on a list of apps.
func RenderAppTable(apps map[string]map[string]string, name string) {
	if len(apps) > 0 {
		fmt.Printf("%v %s %v found.\n", len(apps), name, FormatPlural(len(apps), "site", "sites"))
		table := uitable.New()
		table.MaxColWidth = 200
		table.AddRow("NAME", "TYPE", "URL", "DATABASE URL", "STATUS")

		for _, site := range apps {
			table.AddRow(
				site["name"],
				site["type"],
				site["url"],
				fmt.Sprintf("127.0.0.1:%s", site["DbPublicPort"]),
				site["status"],
			)
		}
		fmt.Println(table)
	}

}
