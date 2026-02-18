package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "share [project]",
	Short:             "Share project on the internet via tunnel provider (ngrok, cloudflared, or custom).",
	Long: `Share your project on the internet using a tunnel provider.
Built-in providers: ngrok (default), cloudflared.
Custom providers can be added to .ddev/share-providers/`,
	Example: `ddev share
ddev share --provider=cloudflared
ddev share --provider-args "--basic-auth username:pass1234"
ddev share --provider=cloudflared --provider-args="--tunnel my-tunnel --hostname mysite.example.com"
ddev share myproject`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			util.Failed("Too many arguments provided. Please use 'ddev share' or 'ddev share [projectname]'")
		}
		apps, err := getRequestedProjects(args, false)
		if err != nil {
			util.Failed("Failed to describe project(s): %v", err)
		}
		app := apps[0]

		status, _ := app.SiteStatus()
		if status != ddevapp.SiteRunning {
			util.Failed("Project is not yet running. Use 'ddev start' first.")
		}

		// Determine which provider to use: flag > project config > global config > default
		providerName := "ngrok" // default
		if globalconfig.DdevGlobalConfig.ShareDefaultProvider != "" {
			providerName = globalconfig.DdevGlobalConfig.ShareDefaultProvider
		}
		if app.ShareDefaultProvider != "" {
			providerName = app.ShareDefaultProvider
		}
		if cmd.Flags().Changed("provider") {
			providerName, err = cmd.Flags().GetString("provider")
			if err != nil {
				util.Failed("Unable to get --provider flag: %v", err)
			}
		}

		// Get provider script path
		scriptPath, err := app.GetShareProviderScript(providerName)
		if err != nil {
			util.Error("Failed to find share provider '%s': %v\n\nAvailable providers:", providerName, err)
			if providers, listErr := app.ListShareProviders(); listErr == nil && len(providers) > 0 {
				for _, p := range providers {
					util.Error("  - %s", p)
				}
			}
			os.Exit(1)
		}

		// Get provider args override from command line
		var providerArgsOverride string
		if cmd.Flags().Changed("provider-args") {
			providerArgsOverride, _ = cmd.Flags().GetString("provider-args")
		}

		// Get environment for provider
		env := app.GetShareProviderEnvironment(providerName, providerArgsOverride)

		// Create pipe to capture stdout (for URL)
		stdoutReader, stdoutWriter, err := os.Pipe()
		if err != nil {
			util.Failed("Failed to create stdout pipe: %v", err)
		}

		// Show what script is being run
		util.Success("Using share provider script: %s", scriptPath)

		// Extract key environment variables for display
		var localURL, shareArgs string
		for _, e := range env {
			if strings.HasPrefix(e, "DDEV_LOCAL_URL=") {
				localURL = strings.TrimPrefix(e, "DDEV_LOCAL_URL=")
			} else if strings.HasPrefix(e, "DDEV_SHARE_ARGS=") {
				shareArgs = strings.TrimPrefix(e, "DDEV_SHARE_ARGS=")
			}
		}
		if shareArgs != "" {
			util.Success("Sharing %s with args: %s", localURL, shareArgs)
		} else {
			util.Success("Sharing %s", localURL)
		}

		// Execute provider script via bash
		bashPath := util.FindBashPath()
		if bashPath == "" {
			util.Failed("Unable to find bash to run share provider script. Please install bash.")
		}
		providerCmd := exec.Command(bashPath, scriptPath)
		providerCmd.Env = env
		providerCmd.Stdout = stdoutWriter
		providerCmd.Stderr = os.Stderr
		setProcessGroupAttr(providerCmd)

		// Set up signal handling for SIGINT/SIGTERM
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start provider
		err = providerCmd.Start()
		if err != nil {
			util.Failed("Failed to start share provider '%s': %v", providerName, err)
		}

		// Close write end immediately after Start - child has its own copy
		// This ensures reader sees EOF when child exits (fixes hang on provider failure)
		_ = stdoutWriter.Close()

		// Capture URL from first line of stdout
		urlChan := make(chan string, 1)
		go func() {
			scanner := bufio.NewScanner(stdoutReader)
			if scanner.Scan() {
				urlChan <- scanner.Text()
			} else {
				urlChan <- ""
			}
			// Drain remaining stdout to prevent provider from blocking
			_, _ = io.Copy(io.Discard, stdoutReader)
		}()

		// Wait for URL with timeout
		var shareURL string
		select {
		case shareURL = <-urlChan:
			if shareURL == "" {
				killProcessTree(providerCmd)
				util.Failed("Provider '%s' did not output a URL", providerName)
			}
		case <-sigChan:
			// Signal received before URL captured
			killProcessTree(providerCmd)
			util.Failed("Interrupted before tunnel URL was established")
		case <-time.After(60 * time.Second):
			killProcessTree(providerCmd)
			util.Failed("Provider '%s' did not output a URL within 60 seconds", providerName)
		}

		// Validate URL
		shareURL = strings.TrimSpace(shareURL)
		parsedURL, err := url.Parse(shareURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			killProcessTree(providerCmd)
			util.Failed("Provider '%s' output invalid URL: %s", providerName, shareURL)
		}

		util.Success("Tunnel URL: %s", shareURL)

		// Set DDEV_SHARE_URL environment variable for hooks
		_ = os.Setenv("DDEV_SHARE_URL", shareURL)

		// Process pre-share hooks NOW (after URL is captured)
		// This fixes issue #7784 - hooks can now access DDEV_SHARE_URL
		err = app.ProcessHooks("pre-share")
		if err != nil {
			util.Warning("Failed to process pre-share hooks: %v", err)
		}

		// Wait for either provider to exit or signal to be received
		done := make(chan error, 1)
		go func() {
			done <- providerCmd.Wait()
		}()

		select {
		case err = <-done:
			// Provider exited on its own
		case <-sigChan:
			// Signal received, kill provider process group
			killProcessTree(providerCmd)
			err = <-done
		}

		// Process post-share hooks
		hookErr := app.ProcessHooks("post-share")
		if hookErr != nil {
			util.Warning("Failed to process post-share hooks: %v", hookErr)
		}

		// Report provider exit status if non-zero and not killed by signal
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Don't warn about exit code -1 which is from Kill()
				if exitErr.ExitCode() != -1 {
					util.Error("Provider '%s' exited with code %d", providerName, exitErr.ExitCode())
					os.Exit(exitErr.ExitCode())
				}
			}
		}

		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().String("provider", "", "share provider to use (ngrok, cloudflared, or custom)")
	_ = DdevShareCommand.RegisterFlagCompletionFunc("provider", configCompletionFunc([]string{"ngrok", "cloudflared"}))
	DdevShareCommand.Flags().String("provider-args", "", "arguments to pass to the share provider")
	DdevShareCommand.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "ngrok-args" {
			newName := "provider-args"
			_, _ = fmt.Fprintf(os.Stderr, "Flag --%s has been deprecated, use --%s instead\n", name, newName)
			name = newName
		}
		return pflag.NormalizedName(name)
	})
}
