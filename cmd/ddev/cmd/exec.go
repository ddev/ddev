package cmd

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
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
ddev exec --raw -- ls -lR
ddev exec -s db -u root ls -la /root`,
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

		container, err := app.FindContainerByType(serviceType)
		if err != nil {
			util.Failed("Failed to find container for service '%s' in '%s' project: %v", serviceType, app.Name, err)
		}
		if container == nil {
			util.Failed("No running container found for service '%s' in '%s' project", serviceType, app.Name)
		}

		_ = app.DockerEnv()

		opts := &ddevapp.ExecOpts{
			Service: serviceType,
			Dir:     execDirArg,
			Cmd:     quoteArgs(args),
			Tty:     true,
			User:    serviceUser,
		}

		// If they've chosen raw, use the actual passed values.
		// Also, retrieve and preserve the current $PATH to ensure the environment is consistent.
		if cmd.Flag("raw").Changed {
			var env []string
			path, _, err := app.Exec(&ddevapp.ExecOpts{
				Service: serviceType,
				Cmd:     "echo $PATH",
				User:    serviceUser,
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
		quiet, _ := cmd.Flags().GetBool("quiet")

		if err != nil {
			exitCode := 1
			var exiterr *exec.ExitError
			if errors.As(err, &exiterr) {
				exitCode = exiterr.ExitCode()
			}
			if !quiet {
				util.Error("Failed to execute command `%s`: %v", opts.Cmd, err)
			}
			output.UserErr.Exit(exitCode)
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
	DdevExecCmd.Flags().StringVarP(&serviceType, "service", "s", "web", "Define the service to connect to. [e.g. web, db]")
	_ = DdevExecCmd.RegisterFlagCompletionFunc("service", ddevapp.GetServiceNamesFunc(true))
	DdevExecCmd.Flags().StringVarP(&execDirArg, "dir", "d", "", "Define the execution directory within the container")
	DdevExecCmd.Flags().Bool("raw", true, "Use raw exec (do not interpret with Bash inside container)")
	DdevExecCmd.Flags().BoolP("quiet", "q", false, "Suppress detailed error output")
	DdevExecCmd.Flags().StringVarP(&serviceUser, "user", "u", "", "Defines the user to use within the container")
	// This requires flags for exec to be specified prior to any arguments, allowing for
	// flags to be ignored by cobra for commands that are to be executed in a container.
	DdevExecCmd.Flags().SetInterspersed(false)
	RootCmd.AddCommand(DdevExecCmd)
}
