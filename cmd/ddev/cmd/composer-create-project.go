package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

var composerDirectoryArg = ""

// ComposerCreateProjectCmd handles ddev composer create-project
var ComposerCreateProjectCmd = &cobra.Command{
	DisableFlagParsing: true,
	Use:                "create-project [args] [flags]",
	Short:              "Executes 'composer create-project' within the web container with the arguments and flags provided",
	Long: `Directs basic invocations of 'composer create-project' within the context of the
web container. Projects will be installed to a temporary directory and moved to
the Composer root directory after install.`,
	Example: `ddev composer create-project drupal/recommended-project
ddev composer create-project drupal/recommended-project .
ddev composer create-project typo3/cms-base-distribution . "^10"
ddev composer create-project drupal/recommended-project --no-install .
ddev composer create-project --repository=https://repo.magento.com/ magento/project-community-edition .
ddev composer create-project --prefer-dist --no-interaction --no-dev psr/log .
`,
	ValidArgsFunction: getComposerCompletionFunc(true),
	Run: func(cmd *cobra.Command, args []string) {
		inputData := readPipedInput()

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
			err = app.CreateDocroot()
			if err != nil {
				util.Failed("Could not create docroot at %s: %v", app.GetAbsDocroot(false), err)
			}
			err = app.Start()
			if err != nil {
				util.Failed("Failed to start app %s to run create-project: %v", app.Name, err)
			}
		}

		composerRoot := app.GetComposerRoot(false, false)

		err = os.MkdirAll(composerRoot, 0755)
		if err != nil {
			util.Failed("Failed to create composerRoot: %v", err)
		}

		// Define a randomly named temp directory for install target
		tmpDir := util.RandString(6)
		containerInstallPath := path.Join("/tmp", tmpDir)

		// Add some args to avoid troubles while cloning the project.
		// We add the three options to "composer create-project": --no-plugins, --no-scripts, --no-install
		// These options make the difference between "composer create-project" and "ddev composer create-project".
		var createArgs []string

		for _, arg := range args {
			if isValidComposerOption(app, "create-project", arg) {
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
		composerCmd = appendAllArgsAtTheEnd(composerCmd, containerInstallPath, app)

		checkForComposerCreateAllowedPaths(app)

		output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)
		stdout, stderr, err := app.Exec(&ddevapp.ExecOpts{
			Service: "web",
			RawCmd:  composerCmd,
			Dir:     "/var/www/html",
			Tty:     isatty.IsTerminal(os.Stdin.Fd()),
			Env:     []string{"XDEBUG_MODE=off"},
		})

		if err != nil {
			util.Failed("Failed to create project: %v\nstderr=%v", err, stderr)
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
			util.Failed("Failed to create project: %v", err)
		}

		prepareAppForComposer(app)

		composerManifest, err := composer.NewManifest(path.Join(composerRoot, composerDirectoryArg, "composer.json"))
		if err != nil {
			util.Failed("Failed to read composer.json: %v", err)
		}
		var validRunScriptArgs []string

		if !noScriptsPresent && composerManifest.HasPostRootPackageInstallScript() {
			// Try to run post-root-package-install.
			composerCmd = []string{
				"composer",
				"run-script",
				"post-root-package-install",
			}

			for i, validCreateArg := range validCreateArgs {
				if isValidComposerOption(app, "run-script", validCreateArg) {
					validRunScriptArgs = append(validRunScriptArgs, validCreateArg)
				} else if strings.HasPrefix(validCreateArg, "-") && i+1 < len(validCreateArgs) && !strings.HasPrefix(validCreateArgs[i+1], "-") {
					// If this is an option with a value, add it.
					if isValidComposerOption(app, "run-script", validCreateArg+" "+validCreateArgs[i+1]) {
						validRunScriptArgs = append(validRunScriptArgs, validCreateArg, validCreateArgs[i+1])
					}
				}
			}

			composerCmd = append(composerCmd, validRunScriptArgs...)
			composerCmd = wrapTTYCommandWithStdin(inputData, composerCmd)

			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

			stdout, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Dir:     getComposerRootInContainer(app),
				RawCmd:  composerCmd,
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
				Env:     []string{"XDEBUG_MODE=off"},
			})

			if err != nil {
				util.Failed("Failed to run post-root-package-install: %v\nstderr=%v", err, stderr)
			}

			if len(stdout) > 0 {
				output.UserOut.Println(stdout)
			}

			if len(stderr) > 0 {
				output.UserErr.Println(stderr)
			}

			prepareAppForComposer(app)
		}

		// If --no-install was not provided by the user, call composer install
		// now to finish the installation in the project root folder.
		if !noInstallPresent {
			composerCmd = []string{
				"composer",
				"install",
			}

			for i, validCreateArg := range validCreateArgs {
				if isValidComposerOption(app, "install", validCreateArg) {
					composerCmd = append(composerCmd, validCreateArg)
				} else if strings.HasPrefix(validCreateArg, "-") && i+1 < len(validCreateArgs) && !strings.HasPrefix(validCreateArgs[i+1], "-") {
					// If this is an option with a value, add it.
					if isValidComposerOption(app, "install", validCreateArg+" "+validCreateArgs[i+1]) {
						composerCmd = append(composerCmd, validCreateArg, validCreateArgs[i+1])
					}
				}
			}

			composerCmd = wrapTTYCommandWithStdin(inputData, composerCmd)
			// Run install command.
			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

			stdout, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				RawCmd:  composerCmd,
				Dir:     getComposerRootInContainer(app),
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
				Env:     []string{"XDEBUG_MODE=off"},
			})

			if err != nil {
				util.Failed("Failed to install project: %v\nstderr=%v", err, stderr)
			}

			if len(stdout) > 0 {
				output.UserOut.Println(stdout)
			}

			if len(stderr) > 0 {
				output.UserErr.Println(stderr)
			}

			prepareAppForComposer(app)
		}

		// Reload composer.json if it has changed in the meantime.
		composerManifest, err = composer.NewManifest(path.Join(composerRoot, composerDirectoryArg, "composer.json"))
		if err != nil {
			util.Failed("Failed to read composer.json: %v", err)
		}

		if !noScriptsPresent && composerManifest.HasPostCreateProjectCmdScript() {
			// Try to run post-create-project-cmd.
			composerCmd = []string{
				"composer",
				"run-script",
				"post-create-project-cmd",
			}

			// If the flags for "run-script" were already validated, don't validate them again.
			if validRunScriptArgs == nil {
				for i, validCreateArg := range validCreateArgs {
					if isValidComposerOption(app, "run-script", validCreateArg) {
						validRunScriptArgs = append(validRunScriptArgs, validCreateArg)
					} else if strings.HasPrefix(validCreateArg, "-") && i+1 < len(validCreateArgs) && !strings.HasPrefix(validCreateArgs[i+1], "-") {
						// If this is an option with a value, add it.
						if isValidComposerOption(app, "run-script", validCreateArg+" "+validCreateArgs[i+1]) {
							validRunScriptArgs = append(validRunScriptArgs, validCreateArg, validCreateArgs[i+1])
						}
					}
				}
			}

			composerCmd = append(composerCmd, validRunScriptArgs...)
			composerCmd = wrapTTYCommandWithStdin(inputData, composerCmd)

			output.UserOut.Printf("Executing Composer command: %v\n", composerCmd)

			stdout, stderr, err = app.Exec(&ddevapp.ExecOpts{
				Service: "web",
				Dir:     getComposerRootInContainer(app),
				RawCmd:  composerCmd,
				Tty:     isatty.IsTerminal(os.Stdin.Fd()),
				Env:     []string{"XDEBUG_MODE=off"},
			})

			if err != nil {
				util.Failed("Failed to run post-create-project-cmd: %v\nstderr=%v", err, stderr)
			}

			if len(stdout) > 0 {
				output.UserOut.Println(stdout)
			}

			if len(stderr) > 0 {
				output.UserErr.Println(stderr)
			}

			prepareAppForComposer(app)
		}

		util.Success("\nddev composer create-project was successful.")

		if runtime.GOOS == "windows" {
			fileutil.ReplaceSimulatedLinks(app.AppRoot)
		}
	},
}

// checkForComposerCreateAllowedPaths ensures that the project does not contain any paths that are not allowed to be present in the composer create-project command
func checkForComposerCreateAllowedPaths(app *ddevapp.DdevApp) {
	appRoot := app.GetAbsAppRoot(false)
	composerRoot := filepath.Join(app.GetComposerRoot(false, false), composerDirectoryArg)
	skipDirs := []string{".ddev", ".DS_Store", ".git", ".idea", ".tarballs", ".vscode"}
	composerCreateAllowedPaths, _ := app.GetComposerCreateAllowedPaths()
	err := filepath.Walk(composerRoot,
		func(walkPath string, walkInfo os.FileInfo, err error) error {
			if walkPath == composerRoot {
				return nil
			}

			checkPath := app.GetRelativeDirectory(walkPath)

			if walkInfo.IsDir() && appRoot == composerRoot && slices.Contains(skipDirs, checkPath) {
				return filepath.SkipDir
			}
			if !slices.Contains(composerCreateAllowedPaths, checkPath) {
				return fmt.Errorf("'%s' is not allowed to be present. composer create-project needs to be run on a clean/empty project with only the following paths: %v - please clean up the project before using 'ddev composer create-project'", filepath.Join(appRoot, checkPath), composerCreateAllowedPaths)
			}
			return err
		})
	if err != nil {
		util.Failed("Failed to create project: %v", err)
	}
}

// appendAllArgsAtTheEnd appends all the arguments at the end of the "composer create-project"
// This command also adjusts the directory properly (the second argument)
func appendAllArgsAtTheEnd(args []string, containerInstallPath string, app *ddevapp.DdevApp) []string {
	optionsWithValues := getListOfComposerOptionsThatCanHaveValues(app)
	var composerArgs []string
	// Start from the third argument, because the first two are "composer create-project"
	for i := 2; i < len(args); i++ {
		arg := args[i]
		// Skip if this is an option
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// Skip if this is a value for an option
		if strings.HasPrefix(args[i-1], "-") && slices.Contains(optionsWithValues, args[i-1]) {
			continue
		}
		// Add the second arg here, which is a directory
		if len(composerArgs) == 1 {
			appRoot := app.GetAbsAppRoot(false)
			absComposerDirectory, err := filepath.Abs(arg)
			if err != nil {
				util.Failed("Failed to get absolute path for '%s': %v", arg, err)
			}
			if !strings.HasPrefix(absComposerDirectory, appRoot) {
				util.Failed("Failed to create project: directory '%s' is outside the project root '%s'", absComposerDirectory, appRoot)
			}
			composerDirectoryArg = strings.TrimPrefix(absComposerDirectory, appRoot)
			composerDirectoryArg = strings.TrimPrefix(composerDirectoryArg, "/")
			// Don't allow to create a project in a subdirectory
			if composerDirectoryArg != "" {
				util.Warning("Installing the project in the '%s' subdirectory is unsupported.", composerDirectoryArg)
				util.Warning("Replace the subdirectory with a dot '.' and try again.")
				util.Warning("Or use 'ddev ssh' to run 'composer create-project' in the web container.")
				os.Exit(1)
			}
			composerArgs = append(composerArgs, path.Join(containerInstallPath, composerDirectoryArg))
		} else {
			// Else add it without changes
			composerArgs = append(composerArgs, arg)
		}
		// Set the arg to empty string, so we can filter it out
		args[i] = ""
	}
	// If there was no directory argument, add one
	if len(composerArgs) == 1 {
		composerArgs = append(composerArgs, path.Join(containerInstallPath, composerDirectoryArg))
	}
	// Filter out the empty arguments
	var filteredArgs []string
	for _, arg := range args {
		if arg != "" {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	return append(filteredArgs, composerArgs...)
}

// getListOfComposerOptionsThatCanHaveValues returns slice of options that can have values in the "composer create-project"
// This is needed to properly filter out arguments.
func getListOfComposerOptionsThatCanHaveValues(app *ddevapp.DdevApp) []string {
	stdout, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     app.GetComposerRoot(true, false),
		RawCmd:  []string{"composer", "create-project", "--help"},
		Env:     []string{"XDEBUG_MODE=off"},
	})
	if err != nil {
		return []string{}
	}
	// Search for lines like:
	// -s, --stability=STABILITY
	// --prefer-install=PREFER-INSTALL
	re := regexp.MustCompile(`(?m)(?:(-\w), )?(--\w[\w-]*)=`)
	// Use map to avoid duplicates
	optionMap := make(map[string]struct{})
	matches := re.FindAllStringSubmatch(stdout, -1)
	for _, match := range matches {
		if match[1] != "" {
			// Add the short option if present
			optionMap[match[1]] = struct{}{}
		}
		if match[2] != "" {
			// Add the long option
			optionMap[match[2]] = struct{}{}
		}
	}
	// Convert the map to a slice
	var options []string
	for option := range optionMap {
		options = append(options, option)
	}
	return options
}

// isValidComposerOption checks if a Composer option is valid for a given command
func isValidComposerOption(app *ddevapp.DdevApp, command string, option string) bool {
	// All arguments are valid for "create-project" and not valid for other commands.
	if !strings.HasPrefix(option, "-") {
		return command == "create-project"
	}
	// Try each option with --dry-run to see if it is valid.
	validateCmd := []string{"composer", command}
	validateCmd = append(validateCmd, strings.Split(option, " ")...)
	validateCmd = append(validateCmd, "--dry-run")
	userOutFunc := util.CaptureUserOut()
	_, _, err := app.Exec(&ddevapp.ExecOpts{
		Service: "web",
		Dir:     getComposerRootInContainer(app),
		RawCmd:  validateCmd,
		Env:     []string{"XDEBUG_MODE=off"},
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

// getComposerRootInContainer returns the composer root in the container
func getComposerRootInContainer(app *ddevapp.DdevApp) string {
	return path.Join(app.GetComposerRoot(true, false), composerDirectoryArg)
}

// prepareAppForComposer creates docroot, if needed, and restarts the app.
// Otherwise, it flushes mutagen, creates settings file, runs post start action
func prepareAppForComposer(app *ddevapp.DdevApp) {
	// Check if there is a docroot. If not, create one.
	if !fileutil.IsDirectory(app.GetAbsDocroot(false)) {
		err := app.CreateDocroot()
		if err != nil {
			util.Failed("Could not create docroot at %s: %v", app.GetAbsDocroot(false), err)
		}
		// Restart the project after creating docroot
		if err := app.Restart(); err != nil {
			util.Failed("Could not restart %s project: %v", app.Name, err)
		}
		return
	}
	// Flush Mutagen to make sure composer.json is in sync after moving it from /tmp
	if err := app.MutagenSyncFlush(); err != nil {
		util.Warning("Could not flush Mutagen: %v", err)
	}
	// Important because Composer can run some scripts depending on this
	if _, err := app.CreateSettingsFile(); err != nil {
		util.Warning("Could not write settings file: %v", err)
	}
	// Important because Composer can run some scripts depending on this
	if err := app.PostStartAction(); err != nil {
		util.Warning("Could not run PostStartAction: %v", err)
	}
}

// readPipedInput reads from os.Stdin if it's not a terminal.
// Returns nil if stdin is a terminal or unreadable.
// Required because os.Stdin is consumed by the first app.Exec() call
// and not forwarded to subsequent executions.
func readPipedInput() []byte {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		return nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil
	}
	return data
}

// wrapTTYCommandWithStdin wraps the given command in `script` to simulate a TTY,
// allowing interactive prompts to work with piped stdin (e.g., inside containers).
// If stdin is already a terminal or input is empty, returns the original command.
func wrapTTYCommandWithStdin(data []byte, cmd []string) []string {
	if isatty.IsTerminal(os.Stdin.Fd()) || len(data) == 0 {
		return cmd
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return cmd
	}
	go func() {
		defer stdinWriter.Close()
		_, _ = stdinWriter.Write(data)
	}()
	os.Stdin = stdinReader
	// 'script' forces execution in a pseudo-terminal (PTY)
	// '-q' suppresses script's start and end messages (quiet mode)
	// '-c' specifies the command to run in the PTY
	// '/dev/null' discards the session log
	return []string{"script", "-q", "-c", strings.Join(cmd, " "), "/dev/null"}
}

// ComposerCreateCmd does the same thing as "ddev composer create-project".
// This command was introduced a while ago and has always caused some confusion:
// why "composer create-project" but "ddev composer create"?
// As of DDEV v1.24.2, "ddev composer create-project" works almost the same as "composer create-project",
// so we can now hide this command.
var ComposerCreateCmd = &cobra.Command{
	Use:                "create [args] [flags]",
	Short:              `Use "ddev composer create-project" instead`,
	DisableFlagParsing: true,
	Hidden:             true,
	Deprecated:         `please start using the identical "ddev composer create-project" instead`,
	Run: func(cmd *cobra.Command, args []string) {
		ComposerCreateProjectCmd.Run(cmd, args)
	},
}

func init() {
	ComposerCmd.AddCommand(ComposerCreateProjectCmd)
	ComposerCmd.AddCommand(ComposerCreateCmd)
}
