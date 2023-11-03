package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/mattn/go-isatty"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var composerCreateYesFlag bool

// ComposerCreateCmd handles ddev composer create
var ComposerCreateCmd = &cobra.Command{
	Use: "create [args] [flags]",
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	Short: "Executes 'composer create-project' within the web container with the arguments and flags provided",
	Long: `Directs basic invocations of 'composer create-project' within the context of the
web container. Projects will be installed to a temporary directory and moved to
the Composer root directory after install. Any existing files in the
composer root will be deleted when creating a project.`,
	Example: `ddev composer create drupal/recommended-project
ddev composer create -y drupal/recommended-project
ddev composer create "typo3/cms-base-distribution:^10"
ddev composer create drupal/recommended-project --no-install
ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition
ddev composer create --prefer-dist --no-interaction --no-dev psr/log
`,
	Run: func(cmd *cobra.Command, args []string) {

		// We only want to pass all flags and args to Composer
		// cobra does not seem to allow direct access to everything predictably
		osargs := []string{}
		if len(os.Args) > 3 {
			osargs = os.Args[3:]
			osargs = nodeps.RemoveItemFromSlice(osargs, "--yes")
			osargs = nodeps.RemoveItemFromSlice(osargs, "-y")
		}
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		// Ensure project is running
		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		docRoot := app.GetDocroot()
		composerRoot := app.GetComposerRoot(false, false)

		err = os.MkdirAll(composerRoot, 0755)
		if err != nil {
			util.Failed("Failed to create composerRoot: %v", err)
		}

		// If composer root is not the app root, make sure it's empty
		if app.GetAbsAppRoot(false) != composerRoot {
			if !fileutil.IsDirectoryEmpty(composerRoot) {
				util.Failed("Failed to create project: '%v' has to be empty", composerRoot)
			}
		} else {
			objs, err := fileutil.ListFilesInDir(composerRoot)
			if err != nil {
				util.Failed("Failed to create project: %v", err)
			}

			allowedEmpty := []string{".ddev", ".git", ".tarballs"}
			allowed := allowedEmpty

			if len(docRoot) > 0 {
				allowed = append(allowed, docRoot)
			}

			for _, o := range objs {
				// Only allow specific things to be present.
				if !nodeps.ArrayContainsString(allowed, o) {
					var allowedString = ""
					if len(allowed) > 1 {
						allowedString = "'" + strings.Join(allowed[:len(allowed)-1], "', '") + "'"
						allowedString += " and "
						allowedString += "'" + allowed[len(allowed)-1] + "'"
					}
					util.Failed("Failed to create project: project has to be recently init, only %v are allowed to be present.", allowedString)
				}

				if !nodeps.ArrayContainsString(allowedEmpty, o) {
					if !fileutil.IsDirectoryEmpty(o) {
						util.Failed("Failed to create project: although '%v' is allowed to be present, it has to be empty", o)
					}
				}
			}
		}

		// Define a randomly named temp directory for install target
		tmpDir := util.RandString(6)
		containerInstallPath := path.Join("/tmp", tmpDir)

		// Remember if --no-install was provided by the user
		noInstallPresent := nodeps.ArrayContainsString(osargs, "--no-install")
		if !noInstallPresent {
			// Add the --no-install option by default to avoid issues with
			// rsyncing many files afterwards to the project root.
			osargs = append(osargs, "--no-install")
		}

		// Build container Composer command
		composerCmd := []string{
			"composer",
			"create-project",
		}
		composerCmd = append(composerCmd, osargs...)
		composerCmd = append(composerCmd, containerInstallPath)

		output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)
		stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			RawCmd:  composerCmd,
			Dir:     "/var/www/html",
			Tty:     isatty.IsTerminal(os.Stdin.Fd()),
		})
		if err != nil {
			util.Failed("Failed to create project:%v, stderr=%v", err, stderr)
		}

		if len(stdout) > 0 {
			fmt.Println(strings.TrimSpace(stdout))
		}

		output.UserOut.Printf("Moving install to Composer root")

		rsyncArgs := "-rltgopD" // Same as -a
		if runtime.GOOS == "windows" {
			rsyncArgs = "-rlD" // on windows can't do perms, owner, group, times
		}
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     fmt.Sprintf(`rsync %s "%s/" "%s/"`, rsyncArgs, containerInstallPath, app.GetComposerRoot(true, false)),
			Dir:     "/var/www/html",
		})

		if err != nil {
			util.Failed("Failed to create project: %v", err)
		}

		// If --no-install was not provided by the user, call composer install
		// now to finish the installation in the project root folder.
		if !noInstallPresent {
			composerCmd = []string{
				"composer",
				"install",
			}

			// Apply args supported by install
			supportedArgs := []string{
				"--prefer-source",
				"--prefer-dist",
				"--prefer-install",
				"--no-dev",
				"--no-progress",
				"--ignore-platform-req",
				"--ignore-platform-reqs",
				"-q",
				"--quiet",
				"--ansi",
				"--no-ansi",
				"-n",
				"--no-interaction",
				"--profile",
				"--no-plugins",
				"--no-scripts",
				"-d",
				"--working-dir",
				"--no-cache",
				"-v",
				"-vv",
				"-vvv",
				"--verbose",
			}

			for _, osarg := range osargs {
				for _, supportedArg := range supportedArgs {
					if strings.HasPrefix(osarg, supportedArg) {
						composerCmd = append(composerCmd, osarg)
					}
				}
			}

			// Run command
			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)
			stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				RawCmd:  composerCmd,
				Dir:     app.GetComposerRoot(true, false),
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
			})
			if err != nil {
				util.Failed("Failed to install project:%v, stderr=%v", err, stderr)
			}

			if len(stdout) > 0 {
				fmt.Println(strings.TrimSpace(stdout))
			}
		}

		// Do a spare restart, which will create any needed settings files
		// and also restart Mutagen
		err = app.Restart()
		if err != nil {
			util.Warning("Failed to restart project after composer create: %v", err)
		}

		if runtime.GOOS == "windows" {
			fileutil.ReplaceSimulatedLinks(app.AppRoot)
		}
	},
}

// ComposerCreateProjectCmd sends people to the right thing
// when they try ddev composer create-project
var ComposerCreateProjectCmd = &cobra.Command{
	Use:                "create-project",
	Short:              "Unsupported, use `ddev composer create` instead",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		util.Failed(`'ddev composer create-project' is unsupported. Please use 'ddev composer create'
for basic project creation or 'ddev ssh' into the web container and execute
'composer create-project' directly.`)
	},
}

func init() {
	ComposerCreateCmd.Flags().BoolVarP(&composerCreateYesFlag, "yes", "y", false, "Yes - skip confirmation prompt")
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCmd.AddCommand(ComposerCreateCmd)
}
