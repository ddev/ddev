package local

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"

	"github.com/drud/bootstrap/cli/cms/config"
	"github.com/drud/bootstrap/cli/cms/model"
	"github.com/drud/drud-go/drudapi"
	"github.com/drud/drud-go/utils"
	"github.com/lextoumbourou/goodhosts"
)

// DrudApp implements the LocalApp interface for Legacy Newmedia apps
type DrudApp struct {
	AppBase
	Client     string
	APIApp     *drudapi.Application
	APIDeploy  *drudapi.Deploy
	DrudClient *drudapi.Request
	Options    *AppOptions
}

func (l *DrudApp) SetOpts(opts AppOptions) {
	l.Options = &opts
	l.Name = opts.Name
	l.Environment = opts.Environment
	l.AppType = opts.AppType
	l.Client = opts.Client
	l.DrudClient = opts.DrudClient
	l.Template = LegacyComposeTemplate
	l.SkipYAML = opts.SkipYAML
}

// Init sets values from the AppInitOptions on the Drud app object
func (l *DrudApp) Init(opts AppOptions) {
	l.SetOpts(opts)

	al := &drudapi.ApplicationList{}
	l.DrudClient.Query = fmt.Sprintf(`where={"name":"%s","client":"%s"}`, l.Name, l.Client)

	err := l.DrudClient.Get(al)
	if err != nil {
		log.Fatal(err)
	}

	if len(al.Items) == 0 {
		log.Fatalln("No deploys found for app", l.Name)
	}
	// GET app again in order to get injected repo data
	app := &al.Items[0]
	err = l.DrudClient.Get(app)
	if err != nil {
		log.Fatal(err)
	}
	l.APIApp = app

	// get deploy that has the passed in name
	deploy := app.GetDeploy(l.Environment)
	if deploy == nil {
		log.Fatalln("Deploy", l.Environment, "does not exist.")
	}
	l.APIDeploy = deploy
	basePath := l.AbsPath()
	err = PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

	if !l.SkipYAML {
		err = WriteLocalAppYAML(l)
		if err != nil {
			log.Println(err)
			log.Fatal("Could not create docker-compose.yaml")
		}
	}

}

func (l *DrudApp) GetOpts() AppOptions {
	return *l.Options
}

func (l *DrudApp) GetTemplate() string {
	return l.Template
}

func (l *DrudApp) GetType() string {
	if l.AppType == "" {
		l.SetType()
	}
	return l.AppType
}

// GetName returns the  name for legacy app
func (l DrudApp) GetName() string {
	return l.Name
}

func (l DrudApp) ContainerPrefix() string {
	return "drud-"
}

// ContainerName returns the base name for drud app containers
func (l DrudApp) ContainerName() string {
	return fmt.Sprintf("%s%s-%s", l.ContainerPrefix(), l.Name, l.Environment)
}

// RelPath returns the path from the '.drud' directory to this apps directory
func (l DrudApp) RelPath() string {
	return path.Join("drud", fmt.Sprintf("%s-%s", l.Name, l.Environment))
}

// AbsPath returnt he full path from root to the app directory
func (l DrudApp) AbsPath() string {
	homedir, err := utils.GetHomeDir()
	if err != nil {
		log.Fatalln(err)
	}
	return path.Join(homedir, ".drud", l.RelPath())
}

// GetRepoDetails uses the Environment field to get the relevant repo details about an app
func (l DrudApp) GetRepoDetails() (RepoDetails, error) {

	details := RepoDetails{
		Org:    l.APIApp.RepoDetails.Org,
		Name:   l.APIApp.RepoDetails.Name,
		Host:   l.APIApp.RepoDetails.Host,
		Branch: l.APIApp.RepoDetails.Branch,
	}

	return details, nil
}

// URL returns the URL for a given application.
func (l *DrudApp) URL() string {
	return "http://" + l.HostName()
}

// HostName returns the hostname of a given application.
func (l *DrudApp) HostName() string {
	return l.ContainerName()
}

func (l *DrudApp) FindPorts() error {
	var err error
	l.WebPublicPort, err = GetPodPort(l.ContainerName() + "-web")
	if err != nil {
		return err
	}

	l.DbPublicPort, err = GetPodPort(l.ContainerName() + "-db")
	return err
}

// GetResources ...
func (l DrudApp) GetResources() error {
	basePath := l.AbsPath()
	cfg, err := GetConfig()
	if err != nil {
		return err
	}

	// save errors for when the waitgroup has finished executing
	errChannel := make(chan error, 1)
	// apparently this is necessary
	finished := make(chan bool, 1)
	// Gather data, files, src resources in parallel
	// limit logical processors to 3
	runtime.GOMAXPROCS(3)
	// set up wait group
	var wg sync.WaitGroup
	wg.Add(3)

	// clone git repo
	go func() {
		defer wg.Done()

		log.Println("Cloning source into", basePath)
		err := CloneSource(&l)
		if err != nil {
			log.Println(err)
			errChannel <- fmt.Errorf("Error cloning source: %s", err)
		}
	}()

	// reset host to not include api version for these requests
	l.DrudClient.Host = fmt.Sprintf("%s://%s", cfg.Protocol, cfg.DrudHost)

	// get signed link to mysql backup and download
	go func() {
		defer wg.Done()

		err := l.GetBackup("mysql")
		if err != nil {
			log.Println(err)
			errChannel <- fmt.Errorf("Error retrieving database backup: %s", err)
		}
	}()

	// get link to files backup and download
	go func() {
		defer wg.Done()

		err := l.GetBackup("files")
		if err != nil {
			log.Println(err)
			errChannel <- fmt.Errorf("Error retrieving file backup: %s", err)
		}
	}()

	// wait for them all to finish
	wg.Wait()
	close(finished)

	select {
	case <-finished:
	case err := <-errChannel:
		if err != nil {
			return fmt.Errorf("Unable to retrieve one or more resources.")
		}
	}

	return nil
}

// UnpackResources ...
func (l DrudApp) UnpackResources() error {
	basePath := l.AbsPath()

	fpath := path.Join(basePath, "mysql.tar.gz")
	defer os.Remove(fpath)
	out, err := exec.Command(
		"tar", "-xzvf", fpath, "-C", path.Join(basePath, "data"),
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	err = os.Rename(
		path.Join(basePath, "data", l.Name+".sql"),
		path.Join(basePath, "data", "data.sql"),
	)
	if err != nil {
		return err
	}

	fpath = path.Join(basePath, "files.tar.gz")
	defer os.Remove(fpath)
	out, err = exec.Command(
		"tar", "-xzvf", fpath, "-C", path.Join(basePath, "files"),
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	t := l.GetType()
	var rsyncFrom string
	var rsyncTo string
	if t == "wp" {
		rsyncFrom = path.Join(basePath, "files", "uploads")
		rsyncTo = path.Join(basePath, "src", "docroot/content/uploads")
	} else {
		rsyncFrom = path.Join(basePath, "files", "files")
		rsyncTo = path.Join(basePath, "src", "docroot/sites/default/files")
	}

	outs, err := utils.RunCommand(
		"rsync",
		[]string{"-avz", "--recursive", "--exclude=profiles", rsyncFrom + "/", rsyncTo},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), outs)
	}

	return nil
}

// Config creates the apps config file adding thigns like database host, name, and password
// as well as other sensitive data like salts.
func (l *DrudApp) Config() error {
	basePath := l.AbsPath()

	err := l.SetType()
	if err != nil {
		return err
	}

	// add config/settings file
	// if no template is set then default to drupal
	if l.APIDeploy.Template == "" {
		l.APIDeploy.Template = "drupal"
	}

	settingsFilePath := ""
	if l.AppType == "drupal" {
		log.Printf("Drupal site. Creating settings.php file.")
		settingsFilePath = path.Join(basePath, "src", "docroot/sites/default/settings.php")
		drupalConfig := model.NewDrupalConfig()
		drupalConfig.DatabaseHost = "db"
		drupalConfig.HashSalt = ""
		if drupalConfig.HashSalt == "" {
			drupalConfig.HashSalt = utils.RandomString(64)
		}

		drupalConfig.DeployURL = l.URL()
		err = config.WriteDrupalConfig(drupalConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}

		// Setup a custom settings file for use with drush.
		dbPort, err := GetPodPort(l.ContainerName() + "-db")
		if err != nil {
			return err
		}

		drushSettingsPath := path.Join(basePath, "src", "drush.settings.php")
		drushConfig := model.NewDrushConfig()
		drushConfig.DatabasePort = dbPort
		err = config.WriteDrushConfig(drushConfig, drushSettingsPath)

		if err != nil {
			log.Fatalln(err)
		}
	} else if l.AppType == "wp" {
		log.Printf("WordPress site. Creating wp-config.php file.")
		settingsFilePath = path.Join(basePath, "src", "docroot/wp-config.php")
		wpConfig := model.NewWordpressConfig()
		wpConfig.DatabaseHost = "db"
		wpConfig.DeployURL = l.URL()
		wpConfig.AuthKey = l.APIApp.AuthKey
		wpConfig.AuthSalt = l.APIApp.AuthSalt
		wpConfig.LoggedInKey = l.APIApp.LoggedInKey
		wpConfig.LoggedInSalt = l.APIApp.LoggedInSalt
		wpConfig.NonceKey = l.APIApp.NonceKey
		wpConfig.NonceSalt = l.APIApp.NonceSalt
		wpConfig.SecureAuthKey = l.APIApp.SecureAuthKey
		wpConfig.SecureAuthSalt = l.APIApp.SecureAuthSalt
		err = config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}
	}

	return nil
}

func (l DrudApp) GetBackup(kind string) error {
	link := &drudapi.BackUpLink{
		AppID:    l.APIApp.AppID,
		DeployID: l.APIDeploy.Name,
		Type:     kind,
	}

	err := l.DrudClient.Get(link)
	if err != nil {
		return err
	}

	// download backup, extract, remove
	fmt.Println("downloading", kind)
	fpath := path.Join(l.AbsPath(), fmt.Sprintf("%s.tar.gz", kind))
	utils.DownloadFile(fpath, link.URL)

	return nil
}

// Start initiates docker-compose up
func (l DrudApp) Start() error {

	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	err := l.SetType()
	if err != nil {
		return err
	}

	if !l.SkipYAML {
		fmt.Println("Creating docker-compose config.")
		err = WriteLocalAppYAML(&l)
		if err != nil {
			return err
		}
	}

	EnsureDockerRouter()

	err = l.AddHostsEntry()
	if err != nil {
		log.Fatal(err)
	}

	cmdArgs := []string{"-f", composePath, "pull"}
	_, err = utils.RunCommand("docker-compose", cmdArgs)

	if err != nil {
		return err
	}

	return utils.DockerCompose(
		"-f", composePath,
		"up",
		"-d",
	)
}

// Stop initiates docker-compose stop
func (l DrudApp) Stop() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	if !utils.IsRunning(l.ContainerName()+"-db") && !utils.IsRunning(l.ContainerName()+"-web") && !ComposeFileExists(&l) {
		return fmt.Errorf("Site does not exist or is malformed.")
	}

	return utils.DockerCompose(
		"-f", composePath,
		"stop",
	)
}

// Down stops the docker containers for the legacy project.
func (l *DrudApp) Down() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	if !ComposeFileExists(l) {
		return fmt.Errorf("Site does not exist or is malformed.")
	}

	err := utils.DockerCompose(
		"-f", composePath,
		"down",
	)
	if err != nil {
		return Cleanup(l)
	}

	return nil

}

// SetType determines the app type and sets it
func (l *DrudApp) SetType() error {
	appType, err := DetermineAppType(l.AbsPath())
	if err != nil {
		return err
	}
	l.AppType = appType
	return nil
}

// Wait ensures that the app appears to be read before returning
func (l *DrudApp) Wait() (string, error) {
	o := utils.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 420
	o.Headers["Host"] = l.HostName()
	err := utils.EnsureHTTPStatus(o)
	if err != nil {
		return "", fmt.Errorf("200 Was not returned from the web container")
	}

	return l.URL(), nil
}

// AddHostsEntry will add the legacy site URL to the local hostfile.
func (l *DrudApp) AddHostsEntry() error {
	if os.Getenv("DRUD_NONINTERACTIVE") != "" {
		fmt.Printf("Please add the following entry to your host file:\n127.0.0.1 %s\n", l.HostName())
		return nil
	}

	hosts, err := goodhosts.NewHosts()
	if err != nil {
		log.Fatalf("could not open hostfile. %s", err)
	}
	if hosts.Has("127.0.0.1", l.HostName()) {
		return nil
	}

	fmt.Println("\n\n\nAdding hostfile entry. You will be prompted for your password.")
	hostnameArgs := []string{"drud", "dev", "hostname", l.HostName(), "127.0.0.1"}
	err = utils.RunCommandPipe("sudo", hostnameArgs)
	return err
}
