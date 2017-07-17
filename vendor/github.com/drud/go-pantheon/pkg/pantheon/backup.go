package pantheon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Backup represents a single backup in the pantheon system.
type Backup struct {
	ID              string    `json:"-"`
	ArchiveType     string    `json:"-"`
	DownloadURL     string    `json:"url,omitempty"`
	SiteID          string    `json:"-"`
	FileName        string    `json:"filename,omitempty"`
	EnvironmentName string    `json:"-"`
	BuildTag        string    `json:"BUILD_TAG"`
	BuildURL        string    `json:"BUILD_URL"`
	EndpointUUID    string    `json:"endpoint_uuid"`
	Folder          string    `json:"folder"`
	Size            jsonInt64 `json:"size"`
	Timestamp       jsonInt64 `json:"timestamp"`
	TotalDirs       jsonInt64 `json:"total_dirs"`
	TotalEntries    jsonInt64 `json:"total_entries"`
	TotalFiles      jsonInt64 `json:"total_files"`
	TotalSize       jsonInt64 `json:"total_size"`
	TTL             jsonInt64 `json:"ttl"`
}

// BackupList represents a list of all backups taken for a given site and environment combination.
type BackupList struct {
	EnvironmentName string
	SiteID          string
	Backups         map[string]Backup
}

// NewBackupList creates an BackupList for a given site. You are responsible for making the HTTP request.
func NewBackupList(siteID string, environmentName string) *BackupList {
	return &BackupList{
		SiteID:          siteID,
		EnvironmentName: environmentName,
		Backups:         make(map[string]Backup),
	}
}

// Path returns the API endpoint which can be used to get a BackupList for the current user.
func (bl BackupList) Path(method string, auth AuthSession) string {
	return fmt.Sprintf("/sites/%s/environments/%s/backups/catalog", bl.SiteID, bl.EnvironmentName)
}

// JSON prepares the BackupList for HTTP transport.
func (bl BackupList) JSON() ([]byte, error) {
	return json.Marshal(bl.Backups)
}

// Unmarshal is responsible for converting a HTTP response into a BackupList struct.
func (bl *BackupList) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, &bl.Backups)
	if err != nil {
		return err
	}

	if len(bl.Backups) > 0 {
		for name, backup := range bl.Backups {
			backup.ID = name
			backup.SiteID = bl.SiteID
			backup.EnvironmentName = bl.EnvironmentName

			/**
			 * The name field is always in the following forms:
			 * - 1489769609_backup_files
			 * - 1489769609_backup_database
			 * - 1489769609_backup_code
			 *
			 * Based on code in terminus, the canonical way to determine 'type' for backup is to parse the name.
			 **/
			nameParts := strings.Split(name, "_")
			backup.ArchiveType = nameParts[2]

			bl.Backups[name] = backup
		}
	}

	return nil
}

// Path returns the API endpoint which can be used to get a BackupList for the current user.
func (b Backup) Path(method string, auth AuthSession) string {
	return fmt.Sprintf("/sites/%s/environments/%s/backups/catalog/%s/%s/s3token", b.SiteID, b.EnvironmentName, b.Folder, b.ArchiveType)
}

// JSON prepares the BackupList for HTTP transport.
func (b Backup) JSON() ([]byte, error) {
	return []byte("{\"method\": \"get\"}"), nil
}

// Unmarshal is responsible for converting a HTTP response into a BackupList struct.
func (b *Backup) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &b)
}

// Download will download the backup to the location specified by downloadLocation.
func (b *Backup) Download(downloadLocation string) error {
	if b.DownloadURL == "" {
		return fmt.Errorf("download URL is unknown. Have you performed an HTTP GET on the backup entity?")
	}

	// Create the file
	out, err := os.Create(downloadLocation)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(b.DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
