package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

var updateStorage bool
var dataType string

// DebugDownloadJSONFileCmd implements the ddev debug download-json-file command
var DebugDownloadJSONFileCmd = &cobra.Command{
	Use:   "download-json-file",
	Short: "Download and display JSON/JSONC files with optional local storage update",
	Long: `Download and display JSON/JSONC files used by DDEV from remote sources.

This command can download various data types:
  - remote-config: DDEV remote configuration (from ddev/remote-config repository)
  - sponsorship-data: DDEV sponsorship information (from ddev/sponsorship-data repository)

The downloaded content is displayed as formatted JSON to stdout.
Optionally updates the local cached storage file (enabled by default).`,
	Example: `ddev debug download-json-file --type=remote-config
ddev debug download-json-file --type=sponsorship-data --update-storage=false`,
	Args: cobra.NoArgs,
	Run: func(_ *cobra.Command, args []string) {
		// Validate type parameter
		if dataType != "remote-config" && dataType != "sponsorship-data" {
			util.Failed("Invalid data type '%s'. Must be 'remote-config' or 'sponsorship-data'", dataType)
		}

		// Download and display the data
		err := downloadAndDisplayJSON("", dataType, updateStorage)
		if err != nil {
			util.Failed("Error downloading JSON file: %v", err)
		}
	},
}

// downloadAndDisplayJSON downloads JSON data from URL or default source and displays it
func downloadAndDisplayJSON(url, dataType string, updateLocalStorage bool) error {
	switch dataType {
	case "remote-config":
		return downloadRemoteConfig(url, updateLocalStorage)
	case "sponsorship-data":
		return downloadSponsorshipData(url, updateLocalStorage)
	default:
		return fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// downloadRemoteConfig downloads remote config data
func downloadRemoteConfig(url string, updateLocalStorage bool) error {
	if url != "" {
		return fmt.Errorf("custom URLs are not supported, use default remote config source")
	}

	// Use default remote config source
	d := downloader.NewGitHubJSONCDownloader(
		"ddev",
		"remote-config",
		"remote-config.jsonc",
		github.RepositoryContentGetOptions{Ref: "main"},
	)

	ctx := context.Background()
	var remoteConfig types.RemoteConfigData
	err := d.Download(ctx, &remoteConfig)
	if err != nil {
		return fmt.Errorf("downloading remote config: %w", err)
	}

	// Display as JSON
	jsonData, err := json.MarshalIndent(remoteConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	output.UserOut.Printf("%s\n", string(jsonData))

	// Update local storage if requested
	if updateLocalStorage {
		err = updateRemoteConfigStorage(remoteConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update local storage: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Local remote config storage updated successfully\n")
		}
	}

	return nil
}

// downloadSponsorshipData downloads sponsorship data
func downloadSponsorshipData(url string, updateLocalStorage bool) error {
	if url != "" {
		return fmt.Errorf("custom URLs are not supported, use default sponsorship data source")
	}

	// Use default sponsorship data source
	d := downloader.NewGitHubJSONCDownloader(
		"ddev",
		"sponsorship-data",
		"data/all-sponsorships.json",
		github.RepositoryContentGetOptions{Ref: "main"},
	)

	ctx := context.Background()
	var sponsorshipData types.SponsorshipData
	err := d.Download(ctx, &sponsorshipData)
	if err != nil {
		return fmt.Errorf("downloading sponsorship data: %w", err)
	}

	// Display as JSON
	jsonData, err := json.MarshalIndent(sponsorshipData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	output.UserOut.Printf("%s\n", string(jsonData))

	// Update local storage if requested
	if updateLocalStorage {
		err = updateSponsorshipDataStorage(sponsorshipData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update local storage: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Local sponsorship data storage updated successfully\n")
		}
	}

	return nil
}

// updateRemoteConfigStorage updates the local remote config cache
func updateRemoteConfigStorage(config types.RemoteConfigData) error {
	globalDir := globalconfig.GetGlobalDdevDir()
	// Ensure the directory exists
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}
	fileStorage := storage.NewFileStorage(filepath.Join(globalDir, ".remote-config"))
	return fileStorage.Write(config)
}

// updateSponsorshipDataStorage updates the local sponsorship data cache
func updateSponsorshipDataStorage(data types.SponsorshipData) error {
	globalDir := globalconfig.GetGlobalDdevDir()
	// Ensure the directory exists
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return err
	}
	sponsorshipStorage := storage.NewSponsorshipFileStorage(filepath.Join(globalDir, ".sponsorship-data"))
	return sponsorshipStorage.Write(&data)
}

func init() {
	DebugDownloadJSONFileCmd.Flags().StringVarP(&dataType, "type", "t", "remote-config", "Type of data to download (remote-config|sponsorship-data)")
	DebugDownloadJSONFileCmd.Flags().BoolVar(&updateStorage, "update-storage", true, "Update local cached storage file")
	DebugCmd.AddCommand(DebugDownloadJSONFileCmd)
}
