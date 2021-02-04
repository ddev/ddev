package ddevapp

import (
	"github.com/drud/ddev/pkg/dockerutil"
	"github.com/drud/ddev/pkg/nodeps"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/version"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"

	"gopkg.in/yaml.v2"
)

// DdevLiveProvider provides ddevLive-specific import functionality.
type DdevLiveProvider struct {
	ProviderType string   `yaml:"provider"`
	app          *DdevApp `yaml:"-"`
	SiteName     string   `yaml:"ddevlive_site_name"`
	OrgName      string   `yaml:"ddevlive_org_name"`
}

// Init handles loading data from saved config.
func (p *DdevLiveProvider) Init(app *DdevApp) error {
	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err := p.Read(configPath)
		if err != nil {
			return err
		}
	}

	p.ProviderType = nodeps.ProviderDdevLive
	return nil
}

// ValidateField provides field level validation for config settings. This is
// used any time a field is set via `ddev config` on the primary app config, and
// allows provider plugins to have additional validation for top level config
// settings.
func (p *DdevLiveProvider) ValidateField(field, value string) error {
	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config ddev-live`
func (p *DdevLiveProvider) PromptForConfig() error {
	for {
		err := p.OrgNamePrompt()
		if err != nil {
			output.UserOut.Errorf("%v\n", err)
			return err
		}

		err = p.SiteNamePrompt()
		if err != nil {
			output.UserOut.Errorf("%v\n", err)
			continue
		}
		break
	}
	return nil
}

// SiteNamePrompt prompts for the ddev-live site name.
func (p *DdevLiveProvider) SiteNamePrompt() error {
	sites, err := p.GetSites()
	if err != nil {
		return err
	}

	if len(sites) < 1 {
		return fmt.Errorf("No DDEV-Live sites were found configured for org %v", p.OrgName)
	}

	prompt := "Site name to use (" + strings.Join(sites, " ") + ")"
	defSitename := sites[0]
	if nodeps.ArrayContainsString(sites, p.app.Name) {
		defSitename = p.app.Name
	}
	siteName := util.Prompt(prompt, defSitename)

	p.SiteName = siteName
	return nil
}

func (p *DdevLiveProvider) GetSites() ([]string, error) {
	// Get a list of all active environments for the current site.
	cmd := fmt.Sprintf(`set -eo pipefail; ddev-live list sites --org="%s" -o json | jq -r ".sites[] | .name"`, p.OrgName)
	uid, _, _ := util.GetContainerUIDGid()
	_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)
	if err != nil {
		return []string{}, fmt.Errorf(`unable to get DDEV-Live sites for org %s - please try ddev exec ddev-live list sites --org="%s -o json" (error=%v, output=%v)`, p.OrgName, p.OrgName, err, out)
	}
	siteAry := strings.Split(strings.Trim(out, "\n"), "\n")
	return siteAry, nil
}

// OrgNamePrompt prompts for the ddev-live org.
func (p *DdevLiveProvider) OrgNamePrompt() error {
	var out string
	var err error
	if p.OrgName == "" {
		uid, _, _ := util.GetContainerUIDGid()
		cmd := `set -eo pipefail; ddev-live config default-org get -o json | jq -r .defaultOrg`
		_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)
		if err != nil {
			util.Failed("Failed to get default org: %v (%v) command=%s", err, out, cmd)
		}
	}
	prompt := "DDEV-Live org name"
	orgName := util.Prompt(prompt, strings.Trim(out, "\n"))

	p.OrgName = orgName
	return nil
}

// GetBackup will create and download a backup
// Valid values for backupType are "database" or "files".
// returns fileURL, importPath, error
func (p *DdevLiveProvider) GetBackup(backupType, environment string) (string, string, error) {
	var err error
	var filePath string
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// Set the import path blank to use the root of the archive by default.
	importPath := ""

	p.prepDownloadDir()

	switch backupType {
	case "database":
		filePath, err = p.getDatabaseBackup()
	case "files":
		filePath, err = p.getFilesBackup()
	default:
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}
	if err != nil {
		return "", "", err
	}

	return filePath, importPath, nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *DdevLiveProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	filesDir := filepath.Join(destDir, "files")
	_ = os.RemoveAll(destDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *DdevLiveProvider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".downloads")
	return destDir
}

func (p *DdevLiveProvider) getFilesBackup() (filename string, error error) {

	uid, _, _ := util.GetContainerUIDGid()

	destDir := filepath.Join(p.getDownloadDir(), "files")
	_ = os.RemoveAll(destDir)
	_ = os.MkdirAll(destDir, 0755)

	// Create a files backup first so we can pull
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("ddev-live backup files %s/%s", p.OrgName, p.SiteName)
	}
	cmd := fmt.Sprintf(`ddev-live backup files %s/%s --output=json | jq -r .filesBackup`, p.OrgName, p.SiteName)
	_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)
	if err != nil {
		return "", fmt.Errorf("unable to ddev-live backup files: %v, cmd=%v output=%v ", err, cmd, out)
	}
	backupDescriptor := fmt.Sprintf("%s/%s", p.OrgName, strings.TrimRight(out, "\n"))

	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("ddev-live describe backup files %s", backupDescriptor)
	}
	// Wait for the files backup to complete
	cmd = fmt.Sprintf(`until [ "$(ddev-live describe backup files %s --output=json 2>/tmp/getbackup.out | jq -r .complete)" = "Completed" ]; do sleep 1; ((count++)); if [ "$count" -ge 360 ]; then echo "failed waiting for ddev-live describe backup files %s: $(cat /tmp/getbackup.out); onemoretry=$(ddev-live describe backup files %s --output=json)"; exit 104; fi; done`, backupDescriptor, backupDescriptor, backupDescriptor)
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)
	if err != nil {
		return "", fmt.Errorf("unable to ddev-live describe backup files: %v, output=%v ", err, out)
	}

	// Retrieve files with ddev-live pull files
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("ddev-live pull files %s/%s", p.OrgName, p.SiteName)
	}
	// In WSL2 there is a docker bug where the /mnt/ddevlive-downloads gets mounted as root:root if
	// you don't wait here.
	wslDistro := nodeps.GetWSLDistro()
	if wslDistro != "" {
		time.Sleep(10 * time.Second)
	}
	cmd = fmt.Sprintf(`until ddev-live pull files --dest /mnt/ddevlive-downloads/files %s/%s 2>/tmp/filespull.out; do sleep 1; ((count++)); if [ "$count" -ge 30 ]; then echo "failed waiting for ddev-live pull files: $(cat /tmp/filespull.out)"; exit 105; fi; done`, p.OrgName, p.SiteName)
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)

	if err != nil {
		return "", fmt.Errorf("unable to pull ddev-live files backup: %v, output=%v ", err, out)
	}

	// Now delete the files backup since we don't need it any more, and to stay under quota
	cmd = fmt.Sprintf(`ddev-live delete backup files -y %s`, backupDescriptor)
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Print(cmd)
	}
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)
	if err != nil {
		util.Warning("unable to delete backup files (output=`%s`): err=%v, command=%s", out, err, cmd)
	}
	return filepath.Join(p.getDownloadDir(), "files"), nil
}

// getDatabaseBackup retrieves database using `ddev-live backup database`, then
// describe until it appears, then download it.
func (p *DdevLiveProvider) getDatabaseBackup() (filename string, error error) {
	_ = os.RemoveAll(p.getDownloadDir())
	_ = os.Mkdir(p.getDownloadDir(), 0755)

	// First, kick off the database backup
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Print("ddev-live backup database")
	}
	uid, _, _ := util.GetContainerUIDGid()
	cmd := fmt.Sprintf(`set -eo pipefail; ddev-live backup database -y -o json %s/%s 2>/dev/null | jq -r .databaseBackup`, p.OrgName, p.SiteName)
	_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp", "DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)

	backupName := strings.Trim(out, "\n")
	if err != nil {
		return "", fmt.Errorf("unable to run `ddev-live backup database %s/%s -o json`: output=%v, err=%v", p.OrgName, p.SiteName, out, err)
	}
	if backupName == "" {
		return "", fmt.Errorf("Received empty backupName from ddev-live backup database")
	}

	// Run ddev-live describe while waiting for database backup to complete
	// ddev-live describe has a habit of failing, especially early, so we keep trying.
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("ddev-live describe backup db %s/%s", p.OrgName, backupName)
	}
	cmd = fmt.Sprintf(`count=0; until [ "$(set -eo pipefail; ddev-live describe backup db %s/%s -y -o json | tee /tmp/ddevlivedescribe.out | jq -r .complete)" = "true" ]; do ((count++)); if [ "$count" -ge 360 ]; then echo "Timed out waiting for ddev-live describe backup db onemoretry=$(ddev-live describe backup db %s/%s -o json)" && cat /tmp/ddevlivedescribe.out; exit 101; fi; sleep 1; done `, p.OrgName, backupName, p.OrgName, backupName)
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true, false)

	if err != nil {
		return "", fmt.Errorf("failure waiting for ddev-live backup database completion: %v; cmd=%s, output=%s", err, cmd, out)
	}

	// Retrieve db backup by using ddev-live pull. Unfortunately, we often get
	// failed to download asset: The access key ID you provided does not exist in our records
	// https://github.com/drud/ddev-live/issues/348, also https://github.com/drud/ddev-live-client/issues/402
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Printf("ddev-live pull db %s/%s", p.OrgName, backupName)
	}

	// In WSL2 there is a docker bug where the /mnt/ddevlive-downloads gets mounted as root:root if
	// you don't wait here.
	wslDistro := nodeps.GetWSLDistro()
	if wslDistro != "" {
		time.Sleep(10 * time.Second)
	}

	cmd = fmt.Sprintf(`cd /mnt/ddevlive-downloads && count=0; until ddev-live pull db %s/%s 2>/tmp/pull.out; do sleep 1; ((count++)); if [ "$count" -ge 5 ]; then echo "failed waiting for ddev-live pull db: $(cat /tmp/pull.out)"; exit 103; fi; done`, p.OrgName, backupName)
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)
	w := strings.Split(out, " ")
	if err != nil || len(w) != 2 {
		return "", fmt.Errorf("unable to pull ddev-live database backup (output=`%s`): err=%v, command=%s", out, err, cmd)
	}

	// Now delete the db backup since we don't need it any more, and to stay under quota
	cmd = fmt.Sprintf(`ddev-live delete backup db -y %s/%s`, p.OrgName, backupName)
	if os.Getenv("DDEV_DEBUG") != "" {
		output.UserOut.Print(cmd)
	}
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"DDEV_LIVE_NO_ANALYTICS=" + os.Getenv("DDEV_LIVE_NO_ANALYTICS")}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true, false)
	if err != nil {
		util.Warning("unable to delete backup (output=`%s`): err=%v, command=%s", out, err, cmd)
	}

	f := strings.Trim(w[1], "\n")
	// Rename the on-host filename to a usable extension
	newFilename := filepath.Join(p.getDownloadDir(), "ddevlivedb.sql.gz")
	err = os.Rename(filepath.Join(p.getDownloadDir(), f), newFilename)
	if err != nil {
		return "", err
	}
	return newFilename, nil
}

// Write the ddevLive provider configuration to a specified location on disk.
func (p *DdevLiveProvider) Write(configPath string) error {
	err := PrepDdevDirectory(filepath.Dir(configPath))
	if err != nil {
		return err
	}

	cfgbytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configPath, cfgbytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// Read ddevLive provider configuration from a specified location on disk.
func (p *DdevLiveProvider) Read(configPath string) error {
	source, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Read config values from file.
	err = yaml.Unmarshal(source, p)
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *DdevLiveProvider) Validate() error {
	return nil
}
