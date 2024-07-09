package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/composer"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/fileutil"
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
`,
	ValidArgsFunction: getComposerCompletionFunc(true),
	Run: func(cmd *cobra.Command, args []string) {
		app, err := ddevapp.GetActiveApp("")
		if err != nil {
			util.Failed(err.Error())
		}

		if len(args) < 1 {
			_ = cmd.Help()
			return
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

				if walkInfo.IsDir() && slices.Contains(skipDirs, checkPath) {
					return filepath.SkipDir
				}
				if !slices.Contains(composerCreateAllowedPaths, checkPath) {
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
			// All arguments are valid for "create-project" and not valid for other commands.
			if !strings.HasPrefix(option, "-") {
				return command == "create-project"
			}
			// Try each option with --dry-run to see if it is valid.
			validateCmd := []string{"composer", command, option, "--dry-run"}
			userOutFunc := util.CaptureUserOut()
			_, _, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Dir:     app.GetComposerRoot(true, false),
				RawCmd:  validateCmd,
			})
			out := userOutFunc()
			if err == nil {
				return true
			}
			// If it's an error for the "--dry-run" we use in validateCmd, then the option is valid.
			if option != "--dry-run" && strings.Contains(out, `"--dry-run" option does not exist`) {
				return true
			}
			// We only care about the "option does not exist" error for "create-project",
			// and if there are other errors, the user should see them.
			if command == "create-project" {
				return !strings.Contains(out, fmt.Sprintf(`"%s" option does not exist`, option))
			}
			// The option is not valid for other commands on any error.
			return false
		}

		// Add some args to avoid troubles while cloning the project.
		// We add the three options to "composer create-project": --no-plugins, --no-scripts, --no-install
		// These options make the difference between "composer create-project" and "ddev composer create".
		var createArgs []string

		for _, arg := range args {
			if isValidComposerOption("create-project", arg) {
				createArgs = append(createArgs, arg)
			}
		}

		// this slice will be used for nested composer commands
		validCreateArgs := createArgs

		if !slices.Contains(createArgs, "--no-plugins") {
			// Don't run plugin events for "composer create-project", but run them for "composer run-script" and "composer install"
			createArgs = append(createArgs, "--no-plugins")
		}

		// Remember if --no-scripts was provided by the user
		noScriptsPresent := slices.Contains(createArgs, "--no-scripts")
		if !noScriptsPresent {
			// Don't run scripts for "composer create-project", but run them for "composer install"
			createArgs = append(createArgs, "--no-scripts")
		}

		// Remember if --no-install was provided by the user
		noInstallPresent := slices.Contains(createArgs, "--no-install")
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

		// If options contain help or version flags, do not continue
		if slices.Contains(createArgs, "-h") || slices.Contains(createArgs, "--help") || slices.Contains(createArgs, "-V") || slices.Contains(createArgs, "--version") {
			return
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
		var validRunScriptArgs []string

		if !noScriptsPresent && composerManifest != nil && composerManifest.HasPostRootPackageInstallScript() {
			// Try to run post-root-package-install.
			composerCmd = []string{
				"composer",
				"run-script",
				"post-root-package-install",
			}

			for _, validCreateArg := range validCreateArgs {
				if isValidComposerOption("run-script", validCreateArg) {
					validRunScriptArgs = append(validRunScriptArgs, validCreateArg)
				}
			}

			composerCmd = append(composerCmd, validRunScriptArgs...)

			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

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

			for _, validCreateArg := range validCreateArgs {
				if isValidComposerOption("install", validCreateArg) {
					composerCmd = append(composerCmd, validCreateArg)
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
		}

		// Reload composer.json if it has changed in the meantime.
		composerManifest, _ = composer.NewManifest(path.Join(composerRoot, "composer.json"))

		if !noScriptsPresent && composerManifest != nil && composerManifest.HasPostCreateProjectCmdScript() {
			// Try to run post-create-project-cmd.
			composerCmd = []string{
				"composer",
				"run-script",
				"post-create-project-cmd",
			}

			// If the flags for "run-script" were already validated, don't validate them again.
			if validRunScriptArgs == nil {
				for _, validCreateArg := range validCreateArgs {
					if isValidComposerOption("run-script", validCreateArg) {
						validRunScriptArgs = append(validRunScriptArgs, validCreateArg)
					}
				}
			}

			composerCmd = append(composerCmd, validRunScriptArgs...)

			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

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
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCmd.AddCommand(ComposerCreateCmd)
}
