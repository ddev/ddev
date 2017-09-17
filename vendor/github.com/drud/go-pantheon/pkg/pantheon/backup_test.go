package pantheon

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInvalidSiteBackupList tests getting a backup list for a site which does not exist.
func TestInvalidSiteBackupList(t *testing.T) {
	assert := assert.New(t)
	bl := NewBackupList("invalid-site-id", "dev")

	mux.HandleFunc(bl.Path("GET", *session), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		http.Error(w, "Not found.", http.StatusNotFound)
	})

	err := session.Request("GET", bl)
	assert.Error(err)
}

// TestInvalidEnvironmentBackupList tests getting a backup list for a valid site but an invalid environment.
func TestInvalidEnvironmentBackupList(t *testing.T) {
	assert := assert.New(t)
	bl := NewBackupList("invalid-site-id", "some-invalid-environment")

	mux.HandleFunc(bl.Path("GET", *session), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Write([]byte("{}"))
	})

	err := session.Request("GET", bl)
	assert.NoError(err)

	// If the environment is invalid, Pantheon still returns a empty JSON response with a HTTP 200.
	// Sort of odd behavior, but we have no choice but to roll with it.
	assert.Equal(len(bl.Backups), 0)
}

// TestEnvironmentList ensures EnvironmentLists can be retrieved as expected.
func TestBackupList(t *testing.T) {
	assert := assert.New(t)
	validSiteID := "some-site-id"
	backupEnvironment := "dev"
	bl := NewBackupList(validSiteID, backupEnvironment)
	mux.HandleFunc(bl.Path("GET", *session), func(w http.ResponseWriter, r *http.Request) {
		// Ensure a HTTP GET request was made with the proper authorization headers.
		testMethod(t, r, "GET")
		assert.Contains(r.Header.Get("Authorization"), session.Session)

		// Send JSON response back.
		contents, err := ioutil.ReadFile("testdata/backups.json")
		assert.NoError(err)
		w.Write(contents)
	})

	err := session.Request("GET", bl)
	assert.NoError(err)

	// Ensure we got a valid response and were able to unmarshal it as expected.
	assert.Equal(len(bl.Backups), 4)

	// Create a list of expected values in the backup list.
	backups := []struct {
		ID              string
		ArchiveType     string
		SiteID          string
		EnvironmentName string
	}{
		{
			"1489769609_backup_manifest",
			"manifest",
			validSiteID,
			backupEnvironment,
		},
		{
			"1489769609_backup_files",
			"files",
			validSiteID,
			backupEnvironment,
		},
		{
			"1489769609_backup_database",
			"database",
			validSiteID,
			backupEnvironment,
		},
		{
			"1489769609_backup_code",
			"code",
			validSiteID,
			backupEnvironment,
		},
	}

	// Loop over expected values and ensure they match what we got in our request.
	for _, backup := range backups {
		assert.Equal(bl.Backups[backup.ID].ID, backup.ID)
		assert.Equal(bl.Backups[backup.ID].ArchiveType, backup.ArchiveType)
		assert.Equal(bl.Backups[backup.ID].SiteID, backup.SiteID)
		assert.Equal(bl.Backups[backup.ID].EnvironmentName, backup.EnvironmentName)
	}
}
