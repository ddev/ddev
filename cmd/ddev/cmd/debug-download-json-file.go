package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ddev/ddev/pkg/config/remoteconfig/downloader"
	"github.com/ddev/ddev/pkg/config/remoteconfig/storage"
	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
	"muzzammil.xyz/jsonc"
)

var updateStorage bool
var dataType string

// DebugDownloadJSONFileCmd implements the ddev debug download-json-file command
var DebugDownloadJSONFileCmd = &cobra.Command{
	Use:   "download-json-file [URL]",
	Short: "Download and display JSON/JSONC files with optional local storage update",
	Long: `Download and display JSON/JSONC files used by DDEV from remote sources.

This command can download various data types:
  - remote-config: DDEV remote configuration (default from ddev/remote-config repository)
  - sponsorship-data: DDEV sponsorship information (default from ddev/sponsorship-data repository)

The downloaded content is displayed as formatted JSON to stdout.
Optionally updates the local cached storage file (enabled by default).

Custom URLs can be provided, or use the defaults by specifying only the --type flag.`,
	Example: `ddev debug download-json-file --type=remote-config
ddev debug download-json-file --type=sponsorship-data --update-storage=false
ddev debug download-json-file https://raw.githubusercontent.com/ddev/remote-config/main/remote-config.jsonc --type=remote-config`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var url string
		if len(args) > 0 {
			url = args[0]
		}

		// Validate type parameter
		if dataType != "remote-config" && dataType != "sponsorship-data" {
			util.Failed("Invalid data type '%s'. Must be 'remote-config' or 'sponsorship-data'", dataType)
		}

		// Download and display the data
		err := downloadAndDisplayJSON(url, dataType, updateStorage)
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
	var remoteConfig types.RemoteConfigData
	var err error

	if url != "" {
		// Download from custom URL
		err = downloadFromURL(url, &remoteConfig)
		if err != nil {
			return fmt.Errorf("downloading from custom URL: %w", err)
		}
	} else {
		// Use default remote config source
		d := downloader.NewGitHubJSONCDownloader(
			"ddev",
			"remote-config",
			"remote-config.jsonc",
			github.RepositoryContentGetOptions{Ref: "main"},
		)

		ctx := context.Background()
		err = d.Download(ctx, &remoteConfig)
		if err != nil {
			return fmt.Errorf("downloading remote config: %w", err)
		}
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
	var sponsorshipData types.SponsorshipData
	var err error

	if url != "" {
		// Download from custom URL
		err = downloadFromURL(url, &sponsorshipData)
		if err != nil {
			return fmt.Errorf("downloading from custom URL: %w", err)
		}
	} else {
		// Use default sponsorship data source
		d := downloader.NewGitHubJSONCDownloader(
			"ddev",
			"sponsorship-data",
			"data/all-sponsorships.json",
			github.RepositoryContentGetOptions{Ref: "main"},
		)

		ctx := context.Background()
		err = d.Download(ctx, &sponsorshipData)
		if err != nil {
			return fmt.Errorf("downloading sponsorship data: %w", err)
		}
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
	fileStorage := storage.NewFileStorage(globalDir + "/.remote-config")
	return fileStorage.Write(config)
}

// updateSponsorshipDataStorage updates the local sponsorship data cache
func updateSponsorshipDataStorage(data types.SponsorshipData) error {
	globalDir := globalconfig.GetGlobalDdevDir()
	sponsorshipStorage := storage.NewSponsorshipFileStorage(globalDir + "/.sponsorship-data")
	return sponsorshipStorage.Write(&data)
}

// downloadFromURL downloads JSON/JSONC content from a custom URL
func downloadFromURL(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Try JSONC first (which handles both JSON and JSONC)
	err = jsonc.Unmarshal(body, target)
	if err != nil {
		// If JSONC fails, try regular JSON
		err = json.Unmarshal(body, target)
		if err != nil {
			return fmt.Errorf("unmarshaling JSON/JSONC: %w", err)
		}
	}

	return nil
}

func init() {
	DebugDownloadJSONFileCmd.Flags().StringVarP(&dataType, "type", "t", "remote-config", "Type of data to download (remote-config|sponsorship-data)")
	DebugDownloadJSONFileCmd.Flags().BoolVar(&updateStorage, "update-storage", true, "Update local cached storage file")
	DebugCmd.AddCommand(DebugDownloadJSONFileCmd)
}
