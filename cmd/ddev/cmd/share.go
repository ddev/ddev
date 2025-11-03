package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// DdevShareCommand contains the "ddev share" command
var DdevShareCommand = &cobra.Command{
	ValidArgsFunction: ddevapp.GetProjectNamesFunc("all", 1),
	Use:               "share [project]",
	Short:             "Share project on the internet via tunnel provider (ngrok, cloudflared).",
	Long: `Share your project on the internet using a tunnel provider.
Built-in providers: ngrok (default), cloudflared.
Custom providers can be added to .ddev/share-providers/`,
	Example: `ddev share
ddev share --provider=cloudflared
ddev share --ngrok-args "--basic-auth username:pass1234"
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

		// Determine which provider to use: flag > config > default
		providerName := "ngrok" // default
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
			util.Failed("Failed to find share provider '%s': %v\n\nAvailable providers:", providerName, err)
			if providers, listErr := app.ListShareProviders(); listErr == nil && len(providers) > 0 {
				for _, p := range providers {
					util.Failed("  - %s", p)
				}
			}
			os.Exit(1)
		}

		// Get environment for provider
		env := app.GetShareProviderEnvironment(providerName)

		// Create pipe to capture stdout (for URL)
		stdoutReader, stdoutWriter, err := os.Pipe()
		if err != nil {
			util.Failed("Failed to create stdout pipe: %v", err)
		}

		// Execute provider script
		providerCmd := exec.Command(scriptPath)
		providerCmd.Env = env
		providerCmd.Stdout = stdoutWriter
		providerCmd.Stderr = os.Stderr

		// Set up signal handling for SIGINT/SIGTERM
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start provider
		err = providerCmd.Start()
		if err != nil {
			util.Failed("Failed to start share provider '%s': %v", providerName, err)
		}

		// Capture URL from first line of stdout
		urlChan := make(chan string, 1)
		go func() {
			scanner := bufio.NewScanner(stdoutReader)
			if scanner.Scan() {
				urlChan <- scanner.Text()
			} else {
				urlChan <- ""
			}
			// Keep reading to prevent provider from blocking on stdout
			for scanner.Scan() {
				// Discard additional stdout
			}
		}()

		// Wait for URL with timeout
		var shareURL string
		select {
		case shareURL = <-urlChan:
			if shareURL == "" {
				_ = providerCmd.Process.Kill()
				util.Failed("Provider '%s' did not output a URL", providerName)
			}
		case <-sigChan:
			// Signal received before URL captured
			if providerCmd.Process != nil {
				_ = providerCmd.Process.Kill()
			}
			util.Failed("Interrupted before tunnel URL was established")
		}

		// Validate URL
		shareURL = strings.TrimSpace(shareURL)
		parsedURL, err := url.Parse(shareURL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			_ = providerCmd.Process.Kill()
			util.Failed("Provider '%s' output invalid URL: %s", providerName, shareURL)
		}

		util.Success("Tunnel URL: %s", shareURL)

		// Set DDEV_SHARE_URL environment variable for hooks
		err = os.Setenv("DDEV_SHARE_URL", shareURL)
		if err != nil {
			util.Warning("Failed to set DDEV_SHARE_URL: %v", err)
		}

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
			// Signal received, kill provider process
			if providerCmd.Process != nil {
				_ = providerCmd.Process.Kill()
			}
			err = <-done
		}

		// Process post-share hooks
		hookErr := app.ProcessHooks("post-share")
		if hookErr != nil {
			util.Warning("Failed to process post-share hooks: %v", hookErr)
		}

		// Report provider exit status if non-zero
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				util.Warning("Provider '%s' exited with code %d", providerName, exitErr.ExitCode())
			}
		}

		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(DdevShareCommand)
	DdevShareCommand.Flags().String("provider", "", "share provider to use (ngrok, cloudflared, or custom)")
	DdevShareCommand.Flags().String("ngrok-args", "", `accepts any flag from "ngrok http --help" (deprecated: use share_ngrok_args in config.yaml)`)
}
