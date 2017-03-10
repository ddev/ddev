package platform

import (
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/lextoumbourou/goodhosts"

	"github.com/drud/ddev/pkg/cms/config"
	"github.com/drud/ddev/pkg/cms/model"
	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/network"
	"github.com/drud/drud-go/utils/stringutil"
	"github.com/drud/drud-go/utils/system"
)


// LocalApp implements the AppBase interface local development apps
type LocalApp struct {
	AppBase
	Options *AppOptions
}

// NewLocalApp returns an empty local app with options struct pre inserted
func NewLocalApp(name string, environment string) *LocalApp {
	app := &LocalApp{
		Options: &AppOptions{},
	}
	app.AppBase.Name = name
	app.AppBase.Environment = environment

	return app
}

func (l *LocalApp) SetOpts(opts AppOptions) {
	l.Options = &opts
	l.Name = opts.Name
	l.Environment = opts.Environment
	//l.AppType = opts.AppType
	l.Template = LegacyComposeTemplate
	if opts.Template != "" {
		l.Template = opts.Template
	}
	l.SkipYAML = opts.SkipYAML
}

func (l *LocalApp) GetOpts() AppOptions {
	return *l.Options
}

func (l *LocalApp) GetTemplate() string {
	return l.Template
}

func (l *LocalApp) GetType() string {
	if l.AppType == "" {
		l.SetType()
	}
	return l.AppType
}

// Init sets values from the AppInitOptions on the Drud app object
func (l *LocalApp) Init(opts AppOptions) {
	l.SetOpts(opts)

	basePath := l.AbsPath()
	err := PrepLocalSiteDirs(basePath)
	if err != nil {
		log.Fatalln(err)
	}

}

// RelPath returns the path from the '.drud' directory to this apps directory
func (l LocalApp) RelPath() string {
	return path.Join("local", fmt.Sprintf("%s-%s", l.Name, l.Environment))
}

// AbsPath return the full path from root to the app directory
func (l LocalApp) AbsPath() string {
	workspace := GetWorkspace()
	return path.Join(workspace, l.RelPath())
}

// GetName returns the  name for local app
func (l LocalApp) GetName() string {
	return l.Name
}

// ContainerPrefix returns the base name for local app containers
func (l LocalApp) ContainerPrefix() string {
	return "local-"
}

// ContainerName returns the base name for local app containers
func (l LocalApp) ContainerName() string {
	return fmt.Sprintf("%s%s-%s", l.ContainerPrefix(), l.Name, l.Environment)
}

// GetResources downloads external data for this app
func (l *LocalApp) GetResources() error {

	fmt.Println("Getting Resources.")
	err := l.GetArchive()
	if err != nil {
		log.Println(err)
		fmt.Println(fmt.Errorf("Error retrieving site resources: %s", err))
	}

	err = l.SetType()
	if err != nil {
		return err
	}

	if !l.SkipYAML {
		err = WriteLocalAppYAML(l)
		if err != nil {
			log.Println("Could not create docker-compose.yaml")
			return err
		}
	}

	return nil
}

// GetArchive downloads external data
func (l *LocalApp) GetArchive() error {
	name := fmt.Sprintf("%[2]s-%[1]s.tar.gz", l.Name, l.Environment)
	basePath := path.Dir(l.AbsPath())
	archive := path.Join(basePath, name)

	if system.FileExists(archive) {
		l.Archive = archive
	}
	return nil
}

// UnpackResources takes the archive from the GetResources method and
// unarchives it. Then the contents are moved to their proper locations.
func (l LocalApp) UnpackResources() error {
	basePath := l.AbsPath()
	fileDir := ""

	if l.AppType == "wp" {
		fileDir = "content/uploads"
	} else if l.AppType == "drupal" || l.AppType == "drupal8" {
		fileDir = "sites/default/files"
	}

	out, err := system.RunCommand(
		"tar",
		[]string{
			"-xzvf",
			l.Archive,
			"-C", path.Join(basePath, "files"),
			"--exclude=sites/default/settings.php",
			"--exclude=docroot/wp-config.php",
		},
	)
	if err != nil {
		fmt.Println(out)
		return err
	}

	err = os.Rename(
		path.Join(basePath, "files", l.Name+".sql"),
		path.Join(basePath, "data", "data.sql"),
	)
	if err != nil {
		return err
	}

	// Ensure sites/default is readable.
	if l.AppType == "drupal" || l.AppType == "drupal8" {
		os.Chmod(path.Join(basePath, "files", "docroot", "sites", "default"), 0755)
	}

	rsyncFrom := path.Join(basePath, "files", "docroot", fileDir)
	rsyncTo := path.Join(basePath, "src", "docroot", fileDir)
	out, err = system.RunCommand(
		"rsync",
		[]string{"-avz", "--recursive", rsyncFrom + "/", rsyncTo},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	// Ensure extracted files are writable so they can be removed when we're done.
	out, err = system.RunCommand(
		"chmod",
		[]string{"-R", "ugo+rw", path.Join(basePath, "files")},
	)
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}
	defer os.RemoveAll(path.Join(basePath, "files"))

	dcfgFile := path.Join(basePath, "src", "drud.yaml")
	if system.FileExists(dcfgFile) {
		log.Println("copying drud.yaml to data container")
		out, err := system.RunCommand("cp", []string{
			dcfgFile,
			path.Join(basePath, "data/drud.yaml"),
		})
		if err != nil {
			fmt.Println(out)
			return err
		}
	}

	return nil
}

// Start initiates docker-compose up
func (l LocalApp) Start() error {

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
	_, err = system.RunCommand("docker-compose", cmdArgs)
	if err != nil {
		return err
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"up",
		"-d",
	)
}

// Stop initiates docker-compose stop
func (l LocalApp) Stop() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	if !dockerutil.IsRunning(l.ContainerName()+"-db") && !dockerutil.IsRunning(l.ContainerName()+"-web") && !ComposeFileExists(&l) {
		return fmt.Errorf("site does not exist or is malformed")
	}

	return dockerutil.DockerCompose(
		"-f", composePath,
		"stop",
	)
}

// SetType determines the app type and sets it
func (l *LocalApp) SetType() error {
	appType, err := DetermineAppType(l.AbsPath())
	if err != nil {
		return err
	}
	l.AppType = appType
	return nil
}

// Wait ensures that the app appears to be read before returning
func (l *LocalApp) Wait() (string, error) {
	o := network.NewHTTPOptions("http://127.0.0.1")
	o.Timeout = 90
	o.Headers["Host"] = l.HostName()
	err := network.EnsureHTTPStatus(o)
	if err != nil {
		return "", fmt.Errorf("200 Was not returned from the web container")
	}

	return l.URL(), nil
}

func (l *LocalApp) FindPorts() error {
	var err error
	l.WebPublicPort, err = GetPodPort(l.ContainerName() + "-web")
	if err != nil {
		return err
	}

	l.DbPublicPort, err = GetPodPort(l.ContainerName() + "-db")
	return err
}

// Config creates the apps config file adding things like database host, name, and password
// as well as other sensitive data like salts.
func (l *LocalApp) Config() error {
	basePath := l.AbsPath()

	if l.AppType == "" {
		err := l.SetType()
		if err != nil {
			return err
		}
	}

	err := l.FindPorts()
	if err != nil {
		return err
	}

	settingsFilePath := ""
	if l.AppType == "drupal" || l.AppType == "drupal8" {
		log.Printf("Drupal site. Creating settings.php file.")
		settingsFilePath = path.Join(basePath, "src", "docroot/sites/default/settings.php")
		drupalConfig := model.NewDrupalConfig()
		drupalConfig.DatabaseHost = "db"
		if drupalConfig.HashSalt == "" {
			drupalConfig.HashSalt = stringutil.RandomString(64)
		}
		if l.AppType == "drupal8" {
			drupalConfig.IsDrupal8 = true
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
		if l.AppType == "drupal8" {
			drushConfig.IsDrupal8 = true
		}
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
		wpConfig.AuthKey = stringutil.RandomString(64)
		wpConfig.AuthSalt = stringutil.RandomString(64)
		wpConfig.LoggedInKey = stringutil.RandomString(64)
		wpConfig.LoggedInSalt = stringutil.RandomString(64)
		wpConfig.NonceKey = stringutil.RandomString(64)
		wpConfig.NonceSalt = stringutil.RandomString(64)
		wpConfig.SecureAuthKey = stringutil.RandomString(64)
		wpConfig.SecureAuthSalt = stringutil.RandomString(64)
		err = config.WriteWordpressConfig(wpConfig, settingsFilePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}

// Down stops the docker containers for the local project.
func (l *LocalApp) Down() error {
	composePath := path.Join(l.AbsPath(), "docker-compose.yaml")

	err := dockerutil.DockerCompose(
		"-f", composePath,
		"down",
	)
	if err != nil {
		return Cleanup(l)
	}

	return nil

}

// URL returns the URL for a given application.
func (l *LocalApp) URL() string {
	return "http://" + l.HostName()
}

// HostName returns the hostname of a given application.
func (l *LocalApp) HostName() string {
	return l.ContainerName()
}

// AddHostsEntry will add the local site URL to the local hostfile.
func (l *LocalApp) AddHostsEntry() error {
	if os.Getenv("DRUD_NONINTERACTIVE") != "" {
		fmt.Printf("DRUD_NONINTERACTIVE is set. If this message is not in a test you may want to add the following entry to your host file:\n127.0.0.1 %s\n", l.HostName())
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
	hostnameArgs := []string{"ddev", "hostname", l.HostName(), "127.0.0.1"}
	err = system.RunCommandPipe("sudo", hostnameArgs)
	return err
}
