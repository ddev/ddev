package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/drud/bootstrap/cli/cms/config"
	"github.com/drud/bootstrap/cli/cms/model"
	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/drudapi"
	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

const drupalFilesPath = "src/docroot/sites/default/files/"
const drupalFilesDir = "files"
const wordpressFilesPath = "src/docroot/content/uploads/"
const wordpressFilesdir = "uploads"

// CloneSource clones or pulls a repo
func CloneSource(branch string, repoURL string, basePath string) error {
	out, err := exec.Command(
		"git", "clone", "-b", branch, repoURL, path.Join(basePath, "src"),
	).CombinedOutput()

	if err != nil {
		if strings.Contains(string(out), "already exists") {
			log.Println("Local copy of site exists...updating")
			cwd, _ := os.Getwd()
			defer os.Chdir(cwd)
			os.Chdir(path.Join(basePath, "src"))

			out, err = exec.Command(
				"git", "pull", "origin", branch,
			).CombinedOutput()

			if err != nil {
				return fmt.Errorf("%s - %s", err.Error(), string(out))
			}
		} else {
			return fmt.Errorf("%s - %s", err.Error(), string(out))
		}
	}

	if len(out) > 0 {
		fmt.Println(string(out))
	}

	return nil
}

// GetBackup gets a signed url and downloads a backup of the given type
// for the given app and deploys
func GetBackup(aid string, did string, kind string, basePath string) error {
	link := &drudapi.BackUpLink{
		AppID:    aid,
		DeployID: did,
		Type:     kind,
	}

	err := drudclient.Get(link)
	if err != nil {
		return err
	}

	// download backup, extract, remove
	// dgfsdgsdfg
	fmt.Println("downloading/unpacking", kind)
	fpath := path.Join(basePath, fmt.Sprintf("%s.tar.gz", kind))
	defer os.Remove(fpath)
	downloadFile(fpath, link.URL)

	destdir := "files"
	if kind == "mysql" {
		destdir = "data"
	}

	out, err := exec.Command(
		"tar", "-xzvf", fpath, "-C", path.Join(basePath, destdir),
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s - %s", err.Error(), string(out))
	}

	return nil
}

// WriteComposeFile creates docker-compose.yaml
func WriteComposeFile(dest string, app *drudapi.Application, deploy *drudapi.Deploy) {
	f, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	nameContainer := fmt.Sprintf("%s-%s", app.AppID, deploy.Name)

	var webImage string
	if deploy.Template == "wordpress" {
		webImage = "drud/nginx-php-fpm-wp"
	} else {
		webImage = "drud/nginx-php-fpm-drupal"
	}

	template := fmt.Sprintf(local.DrudComposeTemplate, nameContainer, webImage)
	f.WriteString(template)
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [app_name] [deploy_name]",
	Short: "Add an existing application to your local development environment",
	Long: `Add an existing application to your local dev environment.  This involves
	downloading of containers, media, and databases.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			log.Fatalln("app_name and deploy_name are expected as arguments.")
		}

		if appClient == "" {
			appClient = cfg.Client
		}

		al := &drudapi.ApplicationList{}
		drudclient.Query = fmt.Sprintf(`where={"name":"%s","client":"%s"}`, args[0], appClient)
		err := drudclient.Get(al)
		if err != nil {
			log.Fatal(err)
		}

		if len(al.Items) == 0 {
			log.Fatalln("No deploys found for app", args[0])
		}
		// GET app again in order to get injected repo data
		app := &al.Items[0]
		err = drudclient.Get(app)
		if err != nil {
			log.Fatal(err)
		}

		// get deploy that hasd the passed in name
		deploy := app.GetDeploy(args[1])
		if deploy == nil {
			log.Fatalln("Deploy", args[1], "does not exist.")
		}

		// set a basepath at $HOME/.drud/client_name/app_name/deploy_name
		basePath := path.Join(homedir, ".drud", appClient, args[0], args[1])
		err = PrepLocalSiteDirs(basePath)
		if err != nil {
			log.Fatalln(err)
		}

		// create a docker-compose file for this deploy
		WriteComposeFile(
			path.Join(basePath, "docker-compose.yaml"),
			app,
			deploy,
		)

		// Gather data, files, src resoruces in parallel

		// limit logical processors to 3
		runtime.GOMAXPROCS(3)
		// set up wait group
		var wg sync.WaitGroup
		wg.Add(3)

		// clone git repo
		go func() {
			defer wg.Done()

			log.Println("Cloning source into", basePath)
			cloneURL := fmt.Sprintf("git@github.com:%s/%s.git", app.RepoDetails.Org, app.RepoDetails.Name)
			err = CloneSource(branch, cloneURL, basePath)
			if err != nil {
				log.Fatalln(err)
			}
		}()

		// reset host to not include api version for these requests
		drudclient.Host = fmt.Sprintf("%s://%s", cfg.Protocol, cfg.DrudHost)

		// get signed link to mysql backup and download
		go func() {
			defer wg.Done()

			err = GetBackup(app.AppID, deploy.Name, "mysql", basePath)
			if err != nil {
				log.Fatalln(err)
			}
		}()

		// get link to files backup and download
		go func() {
			defer wg.Done()

			err = GetBackup(app.AppID, deploy.Name, "files", basePath)
			if err != nil {
				log.Fatalln(err)
			}
		}()

		// wait for them all to finish
		wg.Wait()

		// add config/settings file
		// if no template is set then default to drupal
		if deploy.Template == "" {
			deploy.Template = "drupal"
		}

		// drupal has files in 'files' and wp has them in 'uploads'
		var dirPath string
		var filesdir string
		if deploy.Template == "drupal" {
			dirPath = path.Join(basePath, drupalFilesPath)
			filesdir = drupalFilesDir
		} else if deploy.Template == "wordpress" {
			dirPath = path.Join(basePath, wordpressFilesPath)
			filesdir = wordpressFilesdir
		}

		// Make the files dir inside the src dir if it does not exist...we will rsync to there later
		if _, err = os.Stat(dirPath); os.IsNotExist(err) {
			err = os.Mkdir(dirPath, os.FileMode(int(0774)))
			if err != nil {
				log.Fatal(err)
			}
		}

		// rsync backup files into src directory so they are placed correctly on container startup
		rsyncFrom := path.Join(basePath, fmt.Sprintf("files/%s", filesdir))
		out, err := exec.Command("rsync", "-avz", "--recursive", rsyncFrom+"/", dirPath).CombinedOutput()
		if err != nil {
			fmt.Println(fmt.Errorf("%s - %s", err.Error(), string(out)))
		}

		// run docker-compose up -d in the newly created directory
		dcErr := drudapi.DockerCompose(
			"-f", path.Join(basePath, "docker-compose.yaml"),
			"up",
			"-d",
		)
		if dcErr != nil {
			log.Fatalln(dcErr)
		}

		// use the docker client to wait for the containers to spin up then print a linik to the app
		client, _ := GetDockerClient()
		var publicPort int64
	Loop:
		for {
			fmt.Println("checking for containers")
			containers, _ := client.ListContainers(docker.ListContainersOptions{All: false})
			for _, ctr := range containers {
				if strings.Contains(ctr.Names[0][1:], fmt.Sprintf("%s-%s-web", app.AppID, deploy.Name)) {
					for _, port := range ctr.Ports {
						if port.PublicPort != 0 {
							publicPort = port.PublicPort
							fmt.Printf("http://localhost:%d\n", port.PublicPort)
							break Loop
						}
					}
				}
			}

			time.Sleep(2 * time.Second)
		}

		// create a settings file with db info and place it in the src dir
		settingsFilePath := ""
		if deploy.Template == "drupal" {
			log.Printf("Drupal site. Creating settings.php file.")
			settingsFilePath = path.Join(basePath, "src", "docroot/sites/default/settings.php")
			drupalConfig := model.NewDrupalConfig()
			drupalConfig.DatabaseHost = "db"
			err = config.WriteDrupalConfig(drupalConfig, settingsFilePath)
			if err != nil {
				log.Fatalln(err)
			}
		} else if deploy.Template == "wordpress" {
			log.Printf("WordPress site. Creating wp-config.php file.")
			settingsFilePath = path.Join(basePath, "src", "docroot/wp-config.php")
			wpConfig := model.NewWordpressConfig()
			wpConfig.DatabaseHost = "db"
			wpConfig.DeployURL = fmt.Sprintf("http://localhost:%d", publicPort)
			wpConfig.AuthKey = app.AuthKey
			wpConfig.AuthSalt = app.AuthSalt
			wpConfig.LoggedInKey = app.LoggedInKey
			wpConfig.LoggedInSalt = app.LoggedInSalt
			wpConfig.NonceKey = app.NonceKey
			wpConfig.NonceSalt = app.NonceSalt
			wpConfig.SecureAuthKey = app.SecureAuthKey
			wpConfig.SecureAuthSalt = app.SecureAuthSalt
			err = config.WriteWordpressConfig(wpConfig, settingsFilePath)
			if err != nil {
				log.Fatalln(err)
			}
		}

	},
}

func init() {
	addCmd.Flags().StringVarP(&appClient, "client", "c", "", "Client name")
	LocalCmd.AddCommand(addCmd)
}
