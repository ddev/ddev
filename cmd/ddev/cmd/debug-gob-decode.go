package cmd

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddev/ddev/pkg/output"
	"github.com/ddev/ddev/pkg/util"
	"github.com/spf13/cobra"
)

// Local type definitions for gob decoding (copied from internal packages)
type Message struct {
	Message    string   `json:"message"`
	Title      string   `json:"title,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Versions   string   `json:"versions,omitempty"`
}

type Notifications struct {
	Interval int       `json:"interval"`
	Infos    []Message `json:"infos"`
	Warnings []Message `json:"warnings"`
}

type Ticker struct {
	Interval int       `json:"interval"`
	Messages []Message `json:"messages"`
}

type Messages struct {
	Notifications Notifications `json:"notifications"`
	Ticker        Ticker        `json:"ticker"`
}

type Remote struct {
	Owner    string `json:"owner,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Ref      string `json:"ref,omitempty"`
	Filepath string `json:"filepath,omitempty"`
}

type RemoteConfig struct {
	UpdateInterval int      `json:"update-interval,omitempty"`
	Remote         Remote   `json:"remote,omitempty"`
	Messages       Messages `json:"messages,omitempty"`
}

type fileStorageData struct {
	RemoteConfig RemoteConfig
}

// DebugGobDecodeCmd implements the ddev debug gob-decode command
var DebugGobDecodeCmd = &cobra.Command{
	Use:   "gob-decode [file]",
	Short: "Decode and display contents of a gob-encoded file",
	Long: `Decode and display the contents of Go gob-encoded binary files.

This command can decode various gob files used by DDEV, including:
  - .remote-config files (remote configuration cache)
  - Other gob-encoded state files

The output is displayed as formatted JSON for readability.`,
	Example: `ddev debug gob-decode ~/.ddev/.remote-config
ddev debug gob-decode /path/to/some/file.gob`,
	Args: cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		filename := args[0]

		// Expand ~ to home directory if needed
		if strings.HasPrefix(filename, "~/") {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				util.Failed("Error getting home directory: %v", err)
			}
			filename = filepath.Join(homeDir, filename[2:])
		}

		// Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			util.Failed("File does not exist: %s", filename)
		}

		// Try to decode the file
		err := decodeGobFile(filename)
		if err != nil {
			util.Failed("Error decoding gob file %s: %v", filename, err)
		}
	},
}

// decodeGobFile attempts to decode various known gob file types
func decodeGobFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// First, try to decode as remote config (most common case)
	if err := tryDecodeRemoteConfig(filename); err == nil {
		return nil
	}

	// Try to decode as generic interface{}
	_, err = file.Seek(0, 0) // Reset file pointer
	if err != nil {
		return fmt.Errorf("seeking to start of file: %w", err)
	}
	decoder := gob.NewDecoder(file)

	var data interface{}
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("decoding gob data: %w", err)
	}

	// Convert to JSON for readable output
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling to JSON: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Decoded gob file contents:\n")
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

func init() {
	DebugCmd.AddCommand(DebugGobDecodeCmd)
}
