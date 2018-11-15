package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/drud/ddev/pkg/fileutil"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var (
	// Allows a user to pass the --dev flag to composer create-project
	devArg bool

	// Allows a user to pass the --no-dev flag to composer create-project
	noDevArg bool

	// Allows the user to pass a --stability <arg> option to composer create-project
	stabilityArg string

	// Allows a user to specify that Composer shouldn't require user interaction
	noInteractionArg bool

	// Allows a user to pass the --prefer-dist flag to composer create-project
	preferDistArg bool
)

var ComposerCreateCmd = &cobra.Command{
	Use:   "create [flags] <package> [<version>]",
	Short: "Executes 'composer create-project' within the web container",
	Long: `Directs basic invocations of 'composer create-project' within the context of the
web container. Projects will be installed to a temporary directory and moved to
the project root directory after installation. Any existing files in the
project root will be deleted when creating a project.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			err := cmd.Usage()
			util.CheckErr(err)
			os.Exit(-1)
		}

		var pkg, ver string
		if len(args) == 2 {
			pkg = args[len(args)-2]
			ver = args[len(args)-1]
		} else if len(args) == 1 {
			pkg = args[len(args)-1]
		}

		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		// Ensure project is running
		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		// Make the user confirm that existing contents will be deleted
		util.Warning("Warning: Any existing contents of the project root (%s) will be removed", app.AppRoot)
		if !noInteractionArg {
			if !util.Confirm("Would you like to continue?") {
				util.Failed("create-project cancelled")
			}
		}

		// Remove any contents of project root
		util.Warning("Removing any existing files in project root")
		objs, err := fileutil.ListFilesInDir(app.AppRoot)
		if err != nil {
			util.Failed("Failed to create project: %v", err)
		}

		for _, o := range objs {
			// Preserve .ddev/
			if o == ".ddev" {
				continue
			}

			if err = os.RemoveAll(filepath.Join(app.AppRoot, o)); err != nil {
				util.Failed("Failed to create project: %v", err)
			}
		}

		// Define a randomly named temp directory for install target
		tmpDir := fmt.Sprintf(".tmp_%s", util.RandString(6))
		containerInstallPath := path.Join("/var/www/html", tmpDir)
		hostInstallPath := filepath.Join(app.AppRoot, tmpDir)
		defer cleanupTmpDir(hostInstallPath)

		// Build container composer command
		composerCmd := []string{
			"composer",
			"create-project",
			pkg,
			containerInstallPath,
		}

		if ver != "" {
			composerCmd = append(composerCmd, ver)
		}

		if devArg {
			composerCmd = append(composerCmd, "--dev")
		}

		if noDevArg {
			composerCmd = append(composerCmd, "--no-dev")
		}

		if stabilityArg != "" {
			composerCmd = append(composerCmd, "--stability", stabilityArg)
		}

		if noInteractionArg {
			composerCmd = append(composerCmd, "--no-interaction")
		}

		if preferDistArg {
			composerCmd = append(composerCmd, "--prefer-dist")
		}

		composerCmdString := strings.TrimSpace(strings.Join(composerCmd, " "))
		output.UserOut.Printf("Executing composer command: %s\n", composerCmdString)
		stdout, _, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     composerCmd,
		})
		if err != nil {
			util.Failed("Failed to create project")
		}

		if len(stdout) > 0 {
			fmt.Println(strings.TrimSpace(stdout))
		}

		output.UserOut.Printf("Moving installation to project root")
		bashCmdString := fmt.Sprintf("if [ -d %s ]; then mv %s /var/www/html/; fi", containerInstallPath, path.Join(containerInstallPath, "*"))
		_, _, err = app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     []string{"sh", "-c", bashCmdString},
		})
		if err != nil {
			util.Failed("Failed to create project: %v", err)
		}
	},
}

var ComposerCreateProjectCmd = &cobra.Command{
	Use: "create-project",
	Run: func(cmd *cobra.Command, args []string) {
		util.Failed(`'ddev composer create-project' is unsupported. Please use 'ddev composer create'
for basic project creation or 'ddev ssh' into the web container and execute
'composer create-project' directly.`)
	},
}

func init() {
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCmd.AddCommand(ComposerCreateCmd)
	ComposerCreateCmd.Flags().BoolVar(&devArg, "dev", false, "Pass the --dev flag to composer create-project")
	ComposerCreateCmd.Flags().BoolVar(&noDevArg, "no-dev", false, "Pass the --no-dev flag to composer create-project")
	ComposerCreateCmd.Flags().StringVar(&stabilityArg, "stability", "", "Pass the --stability <arg> option to composer create-project")
	ComposerCreateCmd.Flags().BoolVar(&noInteractionArg, "no-interaction", false, "Pass the --no-interaction flag to composer create-project")
	ComposerCreateCmd.Flags().BoolVar(&preferDistArg, "prefer-dist", false, "Pass the --prefer-dist flag to composer create-project")
}

func cleanupTmpDir(hostTmpDir string) {
	output.UserOut.Println("Removing temporary install directory")
	if err := fileutil.PurgeDirectory(hostTmpDir); err != nil {
		util.Warning("Failed to purge the temporary install directory %s: %v", hostTmpDir, err)
		return
	}

	if err := os.Remove(hostTmpDir); err != nil {
		util.Warning("Failed to remove temporary install directory %v: %v", hostTmpDir, err)
		return
	}
}
