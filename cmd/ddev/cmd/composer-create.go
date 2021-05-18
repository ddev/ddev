package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/mattn/go-isatty"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/fileutil"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
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
the project root directory after installation. Any existing files in the
project root will be deleted when creating a project.`,
	Example: `ddev composer create drupal/recommended-project
ddev composer create -y drupal/recommended-project
ddev composer create "typo3/cms-base-distribution:^10"
ddev composer create drupal/recommended-project --no-install
ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition
ddev composer create --prefer-dist --no-interaction --no-dev psr/log
`,
	Run: func(cmd *cobra.Command, args []string) {

		// We only want to pass all flags and args to composer
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
		if app.SiteStatus() != ddevapp.SiteRunning {
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		// Make the user confirm that existing contents will be deleted
		util.Warning("Warning: ALL EXISTING CONTENT of the project root (%s) will be deleted", app.AppRoot)
		if !composerCreateYesFlag {
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
		tmpDir := fmt.Sprintf(".tmp_ddev_composer_create_%s", util.RandString(6))
		containerInstallPath := path.Join("/var/www/html", tmpDir)
		hostInstallPath := filepath.Join(app.AppRoot, tmpDir)

		// Build container composer command
		composerCmd := []string{
			"composer",
			"create-project",
		}
		composerCmd = append(composerCmd, osargs...)
		composerCmd = append(composerCmd, containerInstallPath)

		composerCmdString := strings.TrimSpace(strings.Join(composerCmd, " "))
		output.UserOut.Printf("Executing composer command: %s\n", composerCmdString)
		stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			Cmd:     composerCmdString,
			Dir:     "/var/www/html",
			Tty:     isatty.IsTerminal(os.Stdin.Fd()),
		})
		if err != nil {
			util.Failed("Failed to create project:%v, stderr=%v", err, stderr)
		}

		if len(stdout) > 0 {
			fmt.Println(strings.TrimSpace(stdout))
		}

		output.UserOut.Printf("Moving installation to project root")

		// Windows has serious problems with performance.
		// If not NFSMountEnabled,
		// we will move the contents of the temp installation
		// using host-side manipulation, but can't do that with a cached filesystem.
		if runtime.GOOS == "windows" && !(app.NFSMountEnabled || app.NFSMountEnabledGlobal) {
			// If traditional windows mount
			err = filepath.Walk(hostInstallPath, func(path string, info os.FileInfo, err error) error {
				// Skip the initial tmp install directory
				if path == hostInstallPath {
					return nil
				}

				elements := strings.Split(path, tmpDir)
				newPath := filepath.Join(elements...)

				// Dirs must be created, not renamed
				if info.IsDir() {
					if err := os.MkdirAll(newPath, info.Mode()); err != nil {
						return fmt.Errorf("unable to move %s to %s: %v", path, newPath, err)
					}

					return nil
				}

				// Rename files to to a path excluding the tmpDir
				if err := os.Rename(path, newPath); err != nil {
					return fmt.Errorf("unable to move %s to %s: %v", path, newPath, err)
				}

				return nil
			})
		} else {
			// All other cases than Windows, we can move the contents easily and quickly inside the container.
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Cmd:     fmt.Sprintf("shopt -s dotglob && mv %s/* /var/www/html && rmdir %s", containerInstallPath, containerInstallPath),
				Dir:     "/var/www/html",
			})
		}
		// This err check picks up either of the above: The filepath.Walk and the mv
		if err != nil {
			util.Failed("Failed to create project: %v", err)
		}
		if runtime.GOOS == "windows" {
			fileutil.ReplaceSimulatedLinks(app.AppRoot)
		}
		// Do a spare start, which will create any needed settings files
		err = app.Stop(false, false)
		if err != nil {
			util.Warning("Failed to stop project after composer create: %v", err)
		}
		err = app.Start()
		if err != nil {
			util.Failed("Failed to start project after composer create: %v", err)
		}
	},
}

// ComposerCreateProjectCmd just sends people to the right thing
// when they try ddev composer create-project
var ComposerCreateProjectCmd = &cobra.Command{
	Use: "create-project",
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
