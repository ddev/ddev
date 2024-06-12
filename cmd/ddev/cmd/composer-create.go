package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// ComposerCreateCmd handles ddev composer create
var ComposerCreateCmd = &cobra.Command{
	DisableFlagParsing: true,
	Use:                "create [args] [flags]",
	Short:              "Executes 'composer create-project' within the web container with the arguments and flags provided",
	Long: `Directs basic invocations of 'composer create-project' within the context of the
web container. Projects will be installed to a temporary directory and moved to
the Composer root directory after install.`,
	Example: `ddev composer create drupal/recommended-project
ddev composer create drupal/recommended-project
ddev composer create "typo3/cms-base-distribution:^10"
ddev composer create drupal/recommended-project --no-install
ddev composer create --repository=https://repo.magento.com/ magento/project-community-edition
ddev composer create --prefer-dist --no-interaction --no-dev psr/log
ddev composer create --preserve-flags --no-interaction psr/log
`,
	ValidArgsFunction: getComposerCompletionFunc(true),
	Run: func(cmd *cobra.Command, _ []string) {
		preserveFlags, _ := cmd.Flags().GetBool("preserve-flags")

		// We only want to pass all flags and args to Composer
		// cobra does not seem to allow direct access to everything predictably
		osargs := []string{}
		if len(os.Args) > 3 {
			osargs = os.Args[3:]
			osargs = nodeps.RemoveItemFromSlice(osargs, "--preserve-flags")
			expandedOsargs := []string{}
			for _, osarg := range osargs {
				// If this is a combined short option like "-test", split it into "-t -e -s -t"
				// This is necessary, because each of these options will be checked for validity
				if strings.HasPrefix(osarg, "-") && !strings.HasPrefix(osarg, "--") && len(osarg) > 2 {
					for _, char := range osarg[1:] {
						expandedOsargs = append(expandedOsargs, "-"+string(char))
					}
				} else {
					expandedOsargs = append(expandedOsargs, osarg)
				}
			}
			osargs = expandedOsargs
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
				util.Failed("failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		composerRoot := app.GetComposerRoot(false, false)

		err = os.MkdirAll(composerRoot, 0755)
		if err != nil {
			util.Failed("Failed to create composerRoot: %v", err)
		}

		appRoot := app.GetAbsAppRoot(false)
		skipDirs := []string{".ddev", ".git", ".tarballs"}
		composerCreateAllowedPaths, _ := app.GetComposerCreateAllowedPaths()
		err = filepath.Walk(appRoot,
			func(walkPath string, walkInfo os.FileInfo, err error) error {
				if walkPath == appRoot {
					return nil
				}

				checkPath := app.GetRelativeDirectory(walkPath)

				if walkInfo.IsDir() && nodeps.ArrayContainsString(skipDirs, checkPath) {
					return filepath.SkipDir
				}
				if !nodeps.ArrayContainsString(composerCreateAllowedPaths, checkPath) {
					return fmt.Errorf("'%s' is not allowed to be present. composer create needs to be run on a clean/empty project with only the following paths: %v - please clean up the project before using 'ddev composer create'", filepath.Join(appRoot, checkPath), composerCreateAllowedPaths)
				}
				if err != nil {
					return err
				}
				return nil
			})

		if err != nil {
			util.Failed("Failed to create project: %v", err)
		}

		// Define a randomly named temp directory for install target
		tmpDir := util.RandString(6)
		containerInstallPath := path.Join("/tmp", tmpDir)

		// Function to check if a Composer option is valid for a given command
		isValidComposerOption := func(command, option string) bool {
			// We have to pass arguments and options to "create-project",
			// but only options for other composer commands.
			if command != "create-project" && !strings.HasPrefix(option, "-") {
				return false
			}
			// Try each option with --dry-run to see if it is valid.
			validateCmd := []string{"composer", command, option, "--dry-run"}
			userOutFunc := util.CaptureUserOut()
			_, _, _ = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Dir:     app.GetComposerRoot(true, false),
				RawCmd:  validateCmd,
			})
			out := userOutFunc()
			// If there is an error from symfony/console, it is an invalid option
			return !strings.Contains(out, fmt.Sprintf(`"%s" option does not exist`, option))
		}

		// Add some args to avoid troubles while cloning as long as
		// --preserve-flags is not set.
		createArgs := []string{}

		for _, osarg := range osargs {
			if isValidComposerOption("create-project", osarg) {
				createArgs = append(createArgs, osarg)
			}
		}

		if !preserveFlags && !nodeps.ArrayContainsString(createArgs, "--no-plugins") {
			createArgs = append(createArgs, "--no-plugins")
		}

		if !preserveFlags && !nodeps.ArrayContainsString(createArgs, "--no-scripts") {
			createArgs = append(createArgs, "--no-scripts")
		}

		// Remember if --no-install was provided by the user
		noInstallPresent := nodeps.ArrayContainsString(createArgs, "--no-install")
		if !noInstallPresent {
			// Add the --no-install option by default to avoid issues with
			// rsyncing many files afterwards to the project root.
			createArgs = append(createArgs, "--no-install")
		}

		// Build container Composer command
		composerCmd := []string{
			"composer",
			"create-project",
		}
		composerCmd = append(composerCmd, createArgs...)
		composerCmd = append(composerCmd, containerInstallPath)

		output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)
		stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			RawCmd:  composerCmd,
			Dir:     "/var/www/html",
			Tty:     isatty.IsTerminal(os.Stdin.Fd()),
		})

		if err != nil {
			util.Failed("failed to create project: %v\nstderr=%v", err, stderr)
		}

		if len(stdout) > 0 {
			output.UserOut.Println(stdout)
		}

		if len(stderr) > 0 {
			output.UserErr.Println(stderr)
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
			util.Failed("failed to create project: %v", err)
		}

		// Make sure composer.json is here with Mutagen enabled
		err = app.MutagenSyncFlush()
		if err != nil {
			util.Failed("Failed to flush Mutagen: %v", err)
		}

		composerManifest, _ := composer.NewManifest(path.Join(composerRoot, "composer.json"))

		if !preserveFlags && composerManifest != nil && composerManifest.HasPostRootPackageInstallScript() {
			// Try to run post-root-package-install.
			composerCmd = []string{
				"composer",
				"run-script",
				"post-root-package-install",
			}

			for _, osarg := range osargs {
				if isValidComposerOption("run-script", osarg) {
					composerCmd = append(composerCmd, osarg)
				}
			}

			output.UserOut.Printf("Executing composer command: %v\n", composerCmd)

			stdout, stderr, _ = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Dir:     app.GetComposerRoot(true, false),
				RawCmd:  composerCmd,
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
			})

			if len(stdout) > 0 {
				output.UserOut.Println(stdout)
			}

			if len(stderr) > 0 {
				output.UserErr.Println(stderr)
			}
		}

		// Do a spare restart, which will create any needed settings files
		// and also restart mutagen
		err = app.Restart()
		if err != nil {
			util.Warning("failed to restart project after composer create: %v", err)
		}

		// If --no-install was not provided by the user, call composer install
		// now to finish the installation in the project root folder.
		if !noInstallPresent {
			composerCmd = []string{
				"composer",
				"install",
			}

			for _, osarg := range osargs {
				if isValidComposerOption("install", osarg) {
					composerCmd = append(composerCmd, osarg)
				}
			}

			// Run install command.
			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

			stdout, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				RawCmd:  composerCmd,
				Dir:     app.GetComposerRoot(true, false),
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
			})

			if err != nil {
				util.Failed("failed to install project: %v\nstderr=%v", err, stderr)
			}

			if len(stdout) > 0 {
				output.UserOut.Println(stdout)
			}

			if len(stderr) > 0 {
				output.UserErr.Println(stderr)
			}

			// Reload composer.json if it has changed in the meantime.
			composerManifest, _ := composer.NewManifest(path.Join(composerRoot, "composer.json"))

			if !preserveFlags && composerManifest != nil && composerManifest.HasPostCreateProjectCmdScript() {
				// Try to run post-create-project-cmd.
				composerCmd = []string{
					"composer",
					"run-script",
					"post-create-project-cmd",
				}

				for _, osarg := range osargs {
					if isValidComposerOption("run-script", osarg) {
						composerCmd = append(composerCmd, osarg)
					}
				}

				output.UserOut.Printf("Executing composer command: %v\n", composerCmd)

				stdout, stderr, _ = app.Exec(&ddevapp.ExecOpts{
					Service: "web",
					Dir:     app.GetComposerRoot(true, false),
					RawCmd:  composerCmd,
					Tty:     isatty.IsTerminal(os.Stdin.Fd()),
				})

				if len(stdout) > 0 {
					output.UserOut.Println(stdout)
				}

				if len(stderr) > 0 {
					output.UserErr.Println(stderr)
				}
			}
		}
		// Do a spare restart, which will create any needed settings files
		// and also restart Mutagen
		err = app.Restart()
		if err != nil {
			util.Warning("Failed to restart project after composer create: %v", err)
		}

		util.Success("\nddev composer create was successful.\nConsider using `ddev config --update` to autodetect configuration for your project")

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
	Hidden:             true,
	Run: func(_ *cobra.Command, _ []string) {
		util.Failed(`'ddev composer create-project' is unsupported. Please use 'ddev composer create'
for basic project creation or 'ddev ssh' into the web container and execute
'composer create-project' directly.`)
	},
}

func init() {
	ComposerCreateCmd.Flags().Bool("preserve-flags", false, "Do not append `--no-plugins` and `--no-scripts` flags")
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCmd.AddCommand(ComposerCreateCmd)
}
