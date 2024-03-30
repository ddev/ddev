package cmd

import (
	"reflect"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DebugConfigYamlCmd implements the ddev debug configyaml command
var DebugConfigYamlCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "configyaml [project]",
	Short:             "Prints the project config.*.yaml usage",
	Example:           "ddev debug configyaml, ddev debug configyaml <projectname>",
	Run: func(_ *cobra.Command, args []string) {
		projectName := ""

		if len(args) > 1 {
			util.Failed("This command only takes one optional argument: project-name")
		}

		if len(args) == 1 {
			projectName = args[0]
		}

		app, err := ddevapp.GetActiveApp(projectName)
		if err != nil {
			util.Failed("Failed to get active project: %v", err)
		}
		// Get fresh version of app so we don't recreate it
		app, err = ddevapp.NewApp(app.AppRoot, false)
		if err != nil {
			util.Failed("NewApp() failed: %v", err)
		}

		configFiles, err := app.ReadConfig(true)
		if err != nil {
			util.Error("failed reading config for project %s: %v", app.Name, err)
		}
		output.UserOut.Printf("These config files were loaded for project %s: %v", app.Name, configFiles)

		// strategy from https://stackoverflow.com/a/47457022/215713
		fields := reflect.TypeOf(*app)
		values := reflect.ValueOf(*app)

		num := fields.NumField()

		for i := 0; i < num; i++ {
			field := fields.Field(i)
			v := values.Field(i)

			yaml := field.Tag.Get("yaml")
			key := strings.Split(yaml, ",")
			if v.CanInterface() && key[0] != "-" && !isZero(v) {
				output.UserOut.Printf("%s: %v", key[0], v)
			}
		}

	},
}

// isZero() takes a reflect value and determines whether it's zero/nil
//
//	from https://stackoverflow.com/a/23555352/215713
//
// When this needs to get promoted for other uses it should be promoted.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZero(v.Field(i))
		}
		return z
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func init() {
	DebugCmd.AddCommand(DebugConfigYamlCmd)
}
