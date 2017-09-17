package main

// This simple demonstration app explores a users sites and environments, then gives
// download links to any database or files downloads.

import (
	"fmt"
	"os"

	"github.com/drud/go-pantheon/pkg/pantheon"
	"log"
)

func main() {
	// Do some basic sanity checking around correct usage.
	if len(os.Args) != 4 {
		fmt.Println("usage: getbackups [sitename] [environment_name] [files|database]")
		os.Exit(1)
	}
	siteName := os.Args[1]
	envName := os.Args[2]
	backupType := os.Args[3]

	if backupType != "files" && backupType != "database" {
		fmt.Printf("You must use the literal string 'files' or 'database' for the backup type. %s used.\r\n", backupType)
		os.Exit(1)
	}

	// Generate a session object based on the TERMINUS_API_TOKEN environment var.
	apiToken := os.Getenv("TERMINUS_API_TOKEN")
	if apiToken == "" {
		fmt.Println("Environment variable TERMINUS_API_TOKEN is not set. Please set it to a valid Terminus API Token.")
		os.Exit(1)
	}
	session := pantheon.NewAuthSession(os.Getenv("TERMINUS_API_TOKEN"))

	// Get a list of all sites the current user has access to. Ensure we can find the site which was used in the CLI arguments in that list.
	SiteList := &pantheon.SiteList{}
	err := session.Request("GET", SiteList)
	if err != nil {
		log.Fatalf("err: %v\nCould not complete GET request to retrieve site list.", err)
	}
	site, err := getSite(siteName, SiteList)
	if err != nil {
		fmt.Printf("Could not find site named %s\r\n", siteName)
		os.Exit(1)
	}

	// Get a list of all active environments for the current site.
	environmentList := pantheon.NewEnvironmentList(site.ID)
	err = session.Request("GET", environmentList)
	if err != nil {
		log.Fatalf("err: %v\nCould not complete GET request to retrieve enivronment list.", err)
	}
	env, ok := environmentList.Environments[envName]
	if !ok {
		fmt.Printf("There was no environment named %s for site %s\r\n", envName, siteName)
		os.Exit(1)
	}

	// Find either a files or database backup, depending on what was asked for.
	bl := pantheon.NewBackupList(site.ID, env.Name)
	err = session.Request("GET", bl)
	if err != nil {
		log.Fatalf("err: %v\nCould not complete GET request to retrieve backup list.", err)
	}
	backup, err := getBackup(backupType, bl, session)
	if err != nil {
		fmt.Printf("Could not get backup of type %s: %v", backupType, err)
		os.Exit(1)
	}

	// Print the backup download link for the user.
	fmt.Printf("\t%s %s %s backup: %s\n", site.Site.Name, envName, backup.ArchiveType, backup.DownloadURL)
}

func getSite(name string, sl *pantheon.SiteList) (*pantheon.Site, error) {
	// Get a list of environments for a given site.
	for i, site := range sl.Sites {
		if site.Site.Name == name {
			return &sl.Sites[i], nil
		}
	}

	return &pantheon.Site{}, fmt.Errorf("could not find site")
}

func getBackup(archiveType string, bl *pantheon.BackupList, session *pantheon.AuthSession) (*pantheon.Backup, error) {
	for _, backup := range bl.Backups {
		if backup.ArchiveType == archiveType {
			// Get a time-limited backup URL from Pantheon. This requires a POST of the backup type to their API.
			err := session.Request("POST", &backup)
			if err != nil {
				return &pantheon.Backup{}, fmt.Errorf("could not get backup URL: %v", err)
			}

			return &backup, nil
		}
	}

	// If no matches were found, just return an empty backup along with an error.
	return &pantheon.Backup{}, fmt.Errorf("could not find a backup of type %s", archiveType)
}
