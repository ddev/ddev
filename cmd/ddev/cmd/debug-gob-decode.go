package cmd

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ddev/ddev/pkg/config/remoteconfig/types"
	"github.com/ddev/ddev/pkg/output"
	"github.com/spf13/cobra"
)

// fileStorageData is used for gob decoding of remote config storage
type fileStorageData struct {
	RemoteConfig types.RemoteConfigData
}

// sponsorshipFileStorageData is used for gob decoding of sponsorship data storage
type sponsorshipFileStorageData struct {
	SponsorshipData types.SponsorshipData `json:"sponsorship_data"`
}

// Amplitude event cache structures (not available in remoteconfig/types)
type StorageEvent struct {
	EventType  string                 `json:"event_type,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	DeviceID   string                 `json:"device_id,omitempty"`
	Time       int64                  `json:"time,omitempty"`
	EventProps map[string]interface{} `json:"event_properties,omitempty"`
	UserProps  map[string]interface{} `json:"user_properties,omitempty"`
}

type eventCache struct {
	LastSubmittedAt time.Time       `json:"last_submitted_at"`
	Events          []*StorageEvent `json:"events"`
}

// DebugGobDecodeCmd implements the ddev debug gob-decode command
var DebugGobDecodeCmd = &cobra.Command{
	Use:    "gob-decode [file]",
	Short:  "Decode and display contents of a gob-encoded file",
	Hidden: true,
	Long: `Decode and display the contents of Go gob-encoded binary files.

This command can decode various gob files used by DDEV, including:
  - .remote-config files (remote configuration cache)
  - .amplitude.cache files (analytics event cache)
  - sponsorship data files (contributor sponsorship information)

The decoder automatically detects the file type and uses the appropriate structure.
The output is displayed as formatted JSON for readability.

Note: Generic gob files with unknown concrete types may not be decodable due to
Go's gob encoding limitations.`,
	Example: `ddev debug gob-decode ~/.ddev/.remote-config
ddev debug gob-decode ~/.ddev/.amplitude.cache
ddev debug gob-decode /path/to/some/file.gob`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		filename := args[0]

		// Convert relative paths to absolute paths safely
		if !filepath.IsAbs(filename) {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %w", err)
			}
			fullPath, err := filepath.Abs(filepath.Join(cwd, filename))
			if err != nil {
				return fmt.Errorf("failed to derive absolute path for %s: %w", filename, err)
			}
			filename = fullPath
		}

		// Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filename)
		}

		// Try to decode the file
		err := decodeGobFile(filename)
		if err != nil {
			return fmt.Errorf("error decoding gob file %s: %w", filename, err)
		}

		return nil
	},
}

// decodeGobFile attempts to decode various known gob file types
func decodeGobFile(filename string) error {
	// Try known specific types first
	decoders := []struct {
		name    string
		decoder func(string) error
	}{
		{"remote config", tryDecodeRemoteConfig},
		{"sponsorship data", tryDecodeSponsorshipData},
		{"amplitude event cache", tryDecodeAmplitudeCache},
	}

	for _, d := range decoders {
		if err := d.decoder(filename); err == nil {
			return nil
		}
	}

	// Fall back to generic interface{} decoding
	return tryDecodeGeneric(filename)
}

// tryDecodeGeneric attempts to decode as generic interface{}
func tryDecodeGeneric(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	var data interface{}
	err = decoder.Decode(&data)
	if err != nil {
		// If we can't decode as interface{}, return the error
		// This means the gob file contains a concrete type that wasn't registered
		return fmt.Errorf("decoding gob data: %w", err)
	}

	// Convert to JSON for readable output
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Generic gob file contents:\n")
	output.UserOut.Printf("%s\n", string(jsonData))
	return nil
}

// tryDecodeRemoteConfig attempts to decode the file as a remote config
func tryDecodeRemoteConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var data fileStorageData
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	// Convert to JSON for readable output
	jsonData, err := json.MarshalIndent(data.RemoteConfig, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Remote config file contents:\n")
	output.UserOut.Printf("%s\n", string(jsonData))
	return nil
}

// tryDecodeSponsorshipData attempts to decode the file as sponsorship data
func tryDecodeSponsorshipData(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var data sponsorshipFileStorageData
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	// Convert to JSON for readable output
	jsonData, err := json.MarshalIndent(data.SponsorshipData, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Sponsorship data file contents:\n")
	output.UserOut.Printf("%s\n", string(jsonData))
	return nil
}

// tryDecodeAmplitudeCache attempts to decode the file as amplitude event cache
func tryDecodeAmplitudeCache(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var data eventCache
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	// Convert to JSON for readable output
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Amplitude event cache contents:\n")
	output.UserOut.Printf("%s\n", string(jsonData))
	return nil
}

func init() {
	DebugCmd.AddCommand(DebugGobDecodeCmd)
}
