package ddevapp

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/AlecAivazis/survey"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/output"
	"github.com/drud/ddev/pkg/util"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// DrudS3BucketName is the name of hte bucket where we can expect to find backups.
// TODO: Move it into configuration
var DrudS3BucketName = "ddev-local-tests"

// DrudS3Provider provides DrudS3-specific import functionality.
type DrudS3Provider struct {
	ProviderType string   `yaml:"provider"`
	app          *DdevApp `yaml:"-"`
	//projectEnvironments []string `yaml:"-"`
	EnvironmentName string `yaml:"environment"`
}

// Init handles loading data from saved config.
func (p *DrudS3Provider) Init(app *DdevApp) error {
	var err error

	p.app = app
	configPath := app.GetConfigPath("import.yaml")
	if fileutil.FileExists(configPath) {
		err = p.Read(configPath)
	}

	p.ProviderType = "drud-s3"
	return err
}

// ValidateField provides field level validation for config settings. This is
// used any time a field is set via `ddev config` on the primary app config, and
// allows provider plugins to have additional validation for top level config
// settings.
func (p *DrudS3Provider) ValidateField(field, value string) error {
	switch field {
	case "Name":
		_, err := findDrudS3Project(value)
		if err != nil {
			return nil
		}
		return err
		// TODO: Validate environment as well, but that has to be done in the context of the project
	}

	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config DrudS3`
func (p *DrudS3Provider) PromptForConfig() error {
	for {
		err := p.environmentPrompt()

		if err == nil {
			return nil
		}

		output.UserOut.Errorf("%v\n", err)
	}
}

// GetBackup will download the most recent backup specified by backupType. Valid values for backupType are "database" or "files".
func (p *DrudS3Provider) GetBackup(backupType string) (fileLocation string, importPath string, err error) {
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// Set the import path (within the archive) blank, to use the root of the archive by default.
	importPath = ""
	err = p.environmentExists()
	if err != nil {
		return "", "", err
	}

	sess, client, err := getDrudS3Session()
	if err != nil {
		return "", "", err
	}

	prefix := "db"
	if backupType == "files" {
		prefix = "files"
	}
	object, err := getLatestS3Object(client, DrudS3BucketName, p.app.Name+"/"+p.EnvironmentName+"/"+prefix)
	if err != nil {
		return "", "", fmt.Errorf("unable to getLatestS3Object for bucket %s project %s environment %s prefix %s, %v", DrudS3BucketName, p.app.Name, p.EnvironmentName, prefix, err)
	}

	// Check to see if this file has been downloaded previously.
	// Attempt a new download If we can't stat the file or we get a mismatch on the filesize.
	destFile := filepath.Join(p.getDownloadDir(), path.Base(*object.Key))

	stat, err := os.Stat(destFile)
	if err != nil || stat.Size() != int64(*object.Size) {
		p.prepDownloadDir()

		err = downloadS3Object(sess, DrudS3BucketName, object, p.getDownloadDir())
		if err != nil {
			return "", "", err
		}
	}

	return destFile, importPath, nil
}

// prepDownloadDir ensures the download cache directories are created and writeable.
func (p *DrudS3Provider) prepDownloadDir() {
	destDir := p.getDownloadDir()
	err := os.MkdirAll(destDir, 0755)
	util.CheckErr(err)
}

func (p *DrudS3Provider) getDownloadDir() string {
	ddevDir := util.GetGlobalDdevDir()
	destDir := filepath.Join(ddevDir, "drud-s3", p.app.Name)

	return destDir
}

// environmentPrompt does the interactive for configuration of the DrudS3 environment.
func (p *DrudS3Provider) environmentPrompt() error {

	environments, err := p.GetEnvironments()
	envAry := util.MapKeysToArray(environments)
	if len(envAry) == 1 {
		p.EnvironmentName = envAry[0]
		util.Success("Only one environment is available, environment is set to %s", p.EnvironmentName)
		return nil
	}
	fmt.Printf("Available environments: %v", envAry)
	var prompt = []*survey.Question{
		{
			Name: "EnvironmentName",
			Prompt: &survey.Select{
				Message: "Choose an environment to pull from:",
				Options: envAry,
				Default: p.EnvironmentName,
			},

			Validate: func(val interface{}) error {

				if str, ok := environments[val.(string)]; !ok {
					return fmt.Errorf("%s is not a valid environment for site %s; available environments are %v", str, p.app.Name, environments)
				}
				return nil
			},
		},
	}
	answer := struct {
		EnvironmentName string
	}{}

	err = survey.Ask(prompt, &answer)
	if err != nil {
		return fmt.Errorf("survey.Ask failed: %v", err)
	}

	p.EnvironmentName = answer.EnvironmentName
	return nil
}

// Write the DrudS3 provider configuration to a spcified location on disk.
func (p *DrudS3Provider) Write(configPath string) error {
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

// Read DrudS3 provider configuration from a specified location on disk.
func (p *DrudS3Provider) Read(configPath string) error {
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

// GetEnvironments will return a list of environments for the currently configured upstream DrudS3 site.
func (p *DrudS3Provider) GetEnvironments() (map[string]interface{}, error) {
	environments, err := findDrudS3Project(p.app.Name)
	if err != nil {
		return nil, err
	}
	return environments, nil
}

// Validate ensures that the current configuration is valid (i.e. the configured DrudS3 site/environment exists)
func (p *DrudS3Provider) Validate() error {
	return p.environmentExists()
}

// environmentExists ensures the currently configured DrudS3 site & environment exists.
func (p *DrudS3Provider) environmentExists() error {
	environments, err := p.GetEnvironments()
	if err != nil {
		return err
	}
	if _, ok := environments[p.EnvironmentName]; !ok {
		return fmt.Errorf("could not find an environment with backups named '%s'", p.EnvironmentName)
	}

	return nil
}

// findDrudS3Project ensures the DrudS3 site specified by name exists, and the current user has access to it.
// It returns an array of environment names and err
func findDrudS3Project(project string) (map[string]interface{}, error) {
	_, client, err := getDrudS3Session()
	if err != nil {
		return nil, err
	}
	// Get a list of all projects the current user has access to.
	projectMap, err := getDrudS3Projects(client, DrudS3BucketName)
	if err != nil {
		return nil, err
	}

	projectWithEnvs := projectMap[project]

	// Get a list of environments for a given project.
	return projectWithEnvs, nil
}

// getDrudS3Session loads the DrudS3 API config from disk and returns a DrudS3 session struct.
func getDrudS3Session() (*session.Session, *s3.S3, error) {
	//ddevDir := util.GetGlobalDdevDir()

	// TODO: Move S3 auth into global config?
	//sessionLocation := filepath.Join(ddevDir, "DrudS3config.json")

	// This is currently depending on auth in ~/.aws/credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to NewSession: %v", err)
	}

	client := s3.New(sess)

	return sess, client, nil
}

// Functions to allow golang sort-by-modified-date (descending) for s3 objects found.
type byModified []*s3.Object

func (objs byModified) Len() int {
	return len(objs)
}
func (objs byModified) Swap(i, j int) {
	objs[i], objs[j] = objs[j], objs[i]
}
func (objs byModified) Less(i, j int) bool {
	return objs[j].LastModified.Before(*objs[i].LastModified)
}

// getDrudS3Projects returns a map of project[environment] so it's easy to
// find what projects are available in the bucket and then to get the
// environments for that project.
func getDrudS3Projects(client *s3.S3, bucket string) (map[string]map[string]interface{}, error) {
	objects, err := getS3ObjectsWithPrefix(client, bucket, "")
	if err != nil {
		return nil, err
	}
	projectMap := make(map[string]map[string]interface{})

	// This sadly is processing all of the items we receive, all the files in all the directories
	for _, obj := range objects {
		// TODO: It might be possible but unlikely for the object key separator not to be a "/"
		components := strings.Split(*obj.Key, "/")
		if (len(components)) >= 2 {
			tmp := make(map[string]interface{})
			tmp[components[1]] = true
			projectMap[components[0]] = tmp
		}
	}
	return projectMap, nil
}

func getS3ObjectsWithPrefix(client *s3.S3, bucket string, prefix string) ([]*s3.Object, error) {
	// TODO: This may be fragile because it could return a lot of items.
	maxKeys := aws.Int64(1000000000)

	// TODO: WARNING: ListObjects only returns first 1000 objects
	resp, err := client.ListObjects(&s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: maxKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list items in bucket %s with prefix %s: %v", bucket, prefix, err)
	}

	if len(resp.Contents) == 0 {
		return nil, fmt.Errorf("there are no objects matching %s in bucket %s", prefix, bucket)
	}
	return resp.Contents, nil
}

func getLatestS3Object(client *s3.S3, bucket string, prefix string) (*s3.Object, error) {
	// TODO: Manage maxKeys better; it would be nice if we could just get recent, but
	// AWS doesn't support that.
	maxKeys := aws.Int64(1000000000)

	// TODO: WARNING: ListObjects only returns first 1000 objects
	resp, err := client.ListObjects(&s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: maxKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list items in bucket %s with prefix %s: %v", bucket, prefix, err)
	}

	if len(resp.Contents) == 0 {
		return nil, fmt.Errorf("there are no objects matching %s in bucket %s", prefix, bucket)
	}

	sort.Sort(byModified(resp.Contents))

	return resp.Contents[0], nil
}

func downloadS3Object(sess *session.Session, bucket string, object *s3.Object, localDir string) error {

	localPath := filepath.Join(localDir, path.Base(*object.Key))
	_, err := os.Stat(localPath)
	var file *os.File
	if os.IsNotExist(err) {
		err = os.MkdirAll(localDir, 0755)
		if err != nil {
			return fmt.Errorf("Failed to mkdir %s: %v", localDir, err)
		}

		file, err = os.Create(localPath)
		if err != nil {
			return fmt.Errorf("Unable to create file %v, %v", localPath, err)
		}
	} else {
		fmt.Printf("File %s already downloaded, skipping\n", localPath)
		return nil
	}

	// nolint: errcheck
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    object.Key,
		})
	if err != nil {
		return fmt.Errorf("unable to download item %v, %v", object, err)
	}

	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
	return nil
}
