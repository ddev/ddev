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
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

var updateStorage bool
var dataType string

// DebugRemoteDataCmd implements the ddev utility remote-data command
var DebugRemoteDataCmd = &cobra.Command{
	Use:    "remote-data",
	Short:  "Download and display remote configuration and sponsorship data",
	Hidden: true,
	Long: `Download and display remote data used by DDEV from GitHub repositories.

This command can download various data types:
  - remote-config: DDEV remote configuration (from ddev/remote-config repository)
  - sponsorship-data: DDEV sponsorship information (from ddev/sponsorship-data repository)

The downloaded content is displayed as formatted JSON to stdout.
Optionally updates the local cached storage file (enabled by default).

This is a developer/debugging tool and is hidden from normal help output.`,
	Example: `ddev utility remote-data --type=remote-config
ddev utility remote-data --type=sponsorship-data --update-storage=false`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		// Ensure global config is loaded
		globalconfig.EnsureGlobalConfig()

		// Validate type parameter
		if dataType != "remote-config" && dataType != "sponsorship-data" {
			return fmt.Errorf("invalid data type. Must be 'remote-config' or 'sponsorship-data'")
		}

		// Download and display the data
		err := downloadAndDisplayJSON("", dataType, updateStorage)
		if err != nil {
			return fmt.Errorf("error downloading JSON file: %w", err)
		}

		return nil
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

	// Use configured remote config source from global config
	config := globalconfig.DdevGlobalConfig.RemoteConfig
	d := downloader.NewURLJSONCDownloader(config.RemoteConfigURL)

	output.UserErr.Printf("Downloading remote config from: %s\n", config.RemoteConfigURL)

	ctx := context.Background()
	var remoteConfigData types.RemoteConfigData
	err := d.Download(ctx, &remoteConfigData)
	if err != nil {
		return fmt.Errorf("downloading remote config: %w", err)
	}

	// Display as JSON
	jsonData, err := json.MarshalIndent(remoteConfigData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	output.UserOut.Printf("%s\n", string(jsonData))

	// Update local storage if requested
	if updateLocalStorage {
		err = updateRemoteConfigStorage(remoteConfigData)
		if err != nil {
			output.UserErr.Printf("Warning: Failed to update local storage: %v\n", err)
		} else {
			output.UserErr.Printf("Local remote config storage updated successfully\n")
		}
	}

	return nil
}

// downloadSponsorshipData downloads sponsorship data
func downloadSponsorshipData(url string, updateLocalStorage bool) error {
	if url != "" {
		return fmt.Errorf("custom URLs are not supported, use default sponsorship data source")
	}

	// Use configured sponsorship data source from global config
	config := globalconfig.DdevGlobalConfig.RemoteConfig
	d := downloader.NewURLJSONCDownloader(config.SponsorshipDataURL)

	output.UserErr.Printf("Downloading sponsorship data from: %s\n", config.SponsorshipDataURL)

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
			output.UserErr.Printf("Warning: Failed to update local storage: %v\n", err)
		} else {
			output.UserErr.Printf("Local sponsorship data storage updated successfully\n")
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
	DebugRemoteDataCmd.Flags().StringVarP(&dataType, "type", "t", "remote-config", "Type of data to download (remote-config|sponsorship-data)")
	_ = DebugRemoteDataCmd.RegisterFlagCompletionFunc("type", configCompletionFunc([]string{"remote-config", "sponsorship-data"}))
	DebugRemoteDataCmd.Flags().BoolVar(&updateStorage, "update-storage", true, "Update local cached storage file")
	_ = DebugRemoteDataCmd.RegisterFlagCompletionFunc("update-storage", configCompletionFunc([]string{"true", "false"}))
	DebugCmd.AddCommand(DebugRemoteDataCmd)
}
