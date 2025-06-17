package cmd

import (
	"os"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// execDirArg allows a configurable container execution directory
var execDirArg string

// DdevExecCmd allows users to execute arbitrary sh commands within a container.
var DdevExecCmd = &cobra.Command{
	Use:     "exec [flags] [command] [command-flags]",
	Aliases: []string{"."},
	Short:   "Execute a shell command in the container for a service. Uses the web service by default.",
	Long:    `Execute a shell command in the container for a service. Uses the web service by default. To run your command in the container for another service, run "ddev exec --service <service> <cmd>". If you want to use raw, uninterpreted command inside container use --raw as in example.`,
	Example: `ddev exec ls /var/www/html
ddev exec --service db
ddev exec -s db
ddev exec -s solr (assuming an add-on service named 'solr')
ddev exec --raw -- ls -lR`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(1)
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed("Failed to exec command: %v", err)
		}

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not currently running. Try 'ddev start'.")
		}

		app.DockerEnv()

		opts := &ddevapp.ExecOpts{
			Service: serviceType,
			Dir:     execDirArg,
			Cmd:     quoteArgs(args),
			Tty:     true,
		}

		// If they've chosen raw, use the actual passed values.
		// Also, retrieve and preserve the current $PATH to ensure the environment is consistent.
		if cmd.Flag("raw").Changed {
			var env []string
			path, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: serviceType,
				Cmd:     "echo $PATH",
			})
			path = strings.Trim(path, "\n")
			if err == nil && path != "" {
				env = append(env, "PATH="+path)
			}
			// opts.RawCmd is used instead of opts.Cmd
			opts.RawCmd = args
			opts.Env = env
		}

		_, _, err = app.Exec(opts)
		verbose, _ := cmd.Flags().GetBool("verbose")

		if err != nil && verbose {
			util.Failed("Failed to execute command `%s`: %v", opts.Cmd, err)
		}
	},
}

// quoteArgs quotes any arguments that contain spaces.
// This avoids splitting quoted strings with spaces into separate arguments.
// The function is adapted from the internal quoteArgs golang function.
func quoteArgs(args []string) string {
	if len(args) < 2 {
		return strings.Join(args, " ")
	}
	var b strings.Builder
	for i, arg := range args {
		if i > 0 {
			b.WriteString(" ")
		}
		if strings.ContainsAny(arg, "\" \t\r\n#") {
			b.WriteString(`"`)
			b.WriteString(strings.ReplaceAll(arg, `"`, `\"`))
			b.WriteString(`"`)
		} else {
			b.WriteString(arg)
		}
	}
	return b.String()
}

func init() {
	DdevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Defines the service to connect to. [e.g. web, db]")
	_ = DdevExecCmd.RegisterFlagCompletionFunc("service", ddevapp.GetServiceNamesFunc(true))
	DdevExecCmd.Flags().StringVarP(&execDirArg, "dir", "d", "", "Defines the execution directory within the container")
	DdevExecCmd.Flags().Bool("raw", true, "Use raw exec (do not interpret with Bash inside container)")
	DdevExecCmd.Flags().BoolP("verbose", "v", false, "Add DDEV default error output")
	// This requires flags for exec to be specified prior to any arguments, allowing for
	// flags to be ignored by cobra for commands that are to be executed in a container.
	DdevExecCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(DdevExecCmd)
}
