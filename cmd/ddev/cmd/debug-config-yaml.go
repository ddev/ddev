package cmd

import (
	"reflect"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var fullYAMLOutput bool
var omitKeys string

// DebugConfigYamlCmd implements the ddev debug configyaml command
var DebugConfigYamlCmd = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "configyaml [project]",
	Short:             "Prints the project config.*.yaml usage",
	Example:           "ddev debug configyaml, ddev debug configyaml <projectname>, ddev debug configyaml --full-yaml, ddev debug configyaml --omit-keys=web_environment",
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
		output.UserErr.Printf("These config files were loaded for project %s: %v", app.Name, configFiles)

		// Parse omit keys
		var omitKeyList []string
		omitKeyMap := make(map[string]bool)
		if omitKeys != "" {
			omitKeyList = strings.Split(omitKeys, ",")
			for _, key := range omitKeyList {
				omitKeyMap[strings.TrimSpace(key)] = true
			}
		}

		if fullYAMLOutput {
			// Output complete processed YAML configuration
			configYAML, err := app.GetProcessedProjectConfigYAML(omitKeyList...)
			if err != nil {
				util.Failed("Failed to get processed project configuration YAML: %v", err)
			}
			output.UserOut.Printf("# Complete processed project configuration:\n%s", string(configYAML))
		} else {
			// strategy from https://stackoverflow.com/a/47457022/215713
			fields := reflect.TypeOf(*app)
			values := reflect.ValueOf(*app)

			num := fields.NumField()

			for i := 0; i < num; i++ {
				field := fields.Field(i)
				v := values.Field(i)

				yaml := field.Tag.Get("yaml")
				key := strings.Split(yaml, ",")
				if v.CanInterface() && key[0] != "-" && !isZero(v) && !omitKeyMap[key[0]] {
					output.UserOut.Printf("%s: %v", key[0], v)
				}
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
	DebugConfigYamlCmd.Flags().BoolVar(&fullYAMLOutput, "full-yaml", false, "Output complete processed YAML configuration instead of individual fields")
	DebugConfigYamlCmd.Flags().StringVar(&omitKeys, "omit-keys", "", "Comma-separated list of keys to omit from output (e.g., web_environment)")
	DebugCmd.AddCommand(DebugConfigYamlCmd)
}
