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
	cmd := fmt.Sprintf(`ddev-live list sites --org="%s" -o json | jq -r ".sites[] | .name"`, p.OrgName)
	uid, _, _ := util.GetContainerUIDGid()
	_, sites, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)
	if err != nil {
		return []string{}, fmt.Errorf("unable to get DDEV-Live sites for org %s - please try `ddev exec ddev-live list sites` (error=%v)", p.OrgName, err)
	}
	siteAry := strings.Split(sites, "\n")
	return siteAry, nil
}

// OrgNamePrompt prompts for the ddev-live org.
func (p *DdevLiveProvider) OrgNamePrompt() error {
	prompt := "DDEV-Live org name"
	orgName := util.Prompt(prompt, "")

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

	if backupType == "files" {
		importPath = fmt.Sprintf("files_%s", environment)
	}

	return filePath, importPath, nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *DdevLiveProvider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	filesDir := filepath.Join(destDir, "files")
	_ = os.RemoveAll(filesDir)
	err := os.MkdirAll(filesDir, 0755)
	util.CheckErr(err)
}

func (p *DdevLiveProvider) getDownloadDir() string {
	destDir := p.app.GetConfigPath(".ddevlive-downloads")
	return destDir
}

func (p *DdevLiveProvider) getFilesBackup() (filename string, error error) {

	uid, _, _ := util.GetContainerUIDGid()

	// Retrieve db backup by using ddev-live pull
	cmd := fmt.Sprintf(`ddev-live pull files --dest /mnt/ddevlive-downloads/files %s/%s`, p.OrgName, p.SiteName)
	_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true)

	if err != nil {
		return "", fmt.Errorf("unable to pull ddev-live files backup: %v, output=%v ", err, out)
	}
	return filepath.Join(p.getDownloadDir(), "files"), nil
}

func (p *DdevLiveProvider) getDatabaseBackup() (filename string, error error) {
	// First, kick off the database backup
	uid, _, _ := util.GetContainerUIDGid()
	cmd := fmt.Sprintf(`ddev-live backup database -y -o json %s/%s | jq -r .databaseBackup`, p.OrgName, p.SiteName)
	_, backupName, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)

	backupName = strings.Trim(backupName, "\n")
	if err != nil {
		return "", fmt.Errorf("unable to start ddev-live backup: output=%v, err=%v", backupName, err)
	}

	// Run ddev-live describe while waiting for database backup to complete
	// ddev-live describe has a habit of failing, especially early, so we keep trying.
	cmd = fmt.Sprintf(`count=0; until [ "$(ddev-live describe backup db %s/%s -y -o json >/tmp/ddevlivedescribe.out && jq -r .complete </tmp/ddevlivedescribe.out)" = "true" ]; do ((count++)); if [ ${count} -ge 120 ]; then cat /tmp/ddevlivedescribe.out; exit 101; fi; sleep 1; done `, p.OrgName, backupName)
	_, out, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, nil, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)

	if err != nil {
		return "", fmt.Errorf("unable to wait for ddev-live backup completion: %v; output=%s", err, out)
	}

	// Retrieve db backup by using ddev-live pull
	cmd = fmt.Sprintf(`cd /mnt/ddevlive-downloads && ddev-live pull db %s/%s`, p.OrgName, backupName)
	_, out, err = dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", cmd}, nil, nil, []string{"ddev-global-cache:/mnt/ddev-global-cache", fmt.Sprintf("%s:/mnt/ddevlive-downloads", p.getDownloadDir())}, uid, true)
	w := strings.Split(out, " ")
	if err != nil || len(w) != 2 {
		return "", fmt.Errorf("unable to pull ddev-live database backup (output=`%s`): err=%v", out, err)
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

// findddevLiveSite ensures the ddevLive site specified by name exists, and the current user has access to it.
func (p *DdevLiveProvider) findddevLiveSite(name string) (id string, e error) {
	uid, _, _ := util.GetContainerUIDGid()
	_, id, err := dockerutil.RunSimpleContainer(version.GetWebImage(), "", []string{"bash", "-c", "ddev-live list sites -o json " + name + " | jq -r .id"}, nil, []string{"HOME=/tmp"}, []string{"ddev-global-cache:/mnt/ddev-global-cache"}, uid, true)

	if err != nil {
		return "", fmt.Errorf("could not find a ddevLive site named %s: %v (%v)", name, err, id)
	}

	id = strings.Trim(id, "\n\r ")
	return id, nil
}

// Validate ensures that the current configuration is valid (i.e. the configured pantheon site/environment exists)
func (p *DdevLiveProvider) Validate() error {
	return nil
}
