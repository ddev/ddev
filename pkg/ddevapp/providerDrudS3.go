package ddevapp

import (
	"fmt"
	"github.com/drud/ddev/pkg/globalconfig"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/drud/ddev/pkg/fileutil"
	"github.com/drud/ddev/pkg/util"
	"gopkg.in/yaml.v2"
)

// DrudS3Provider provides DrudS3-specific import functionality.
type DrudS3Provider struct {
	ProviderType    string   `yaml:"provider"`
	app             *DdevApp `yaml:"-"`
	EnvironmentName string   `yaml:"environment"`
	AWSAccessKey    string   `yaml:"aws_access_key_id"`
	AWSSecretKey    string   `yaml:"aws_secret_access_key"`
	S3Bucket        string   `yaml:"s3_bucket"`
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
	// No validation is done here because so many things depend on each other.
	// Instead we use p.Validate()
	return nil
}

// PromptForConfig provides interactive configuration prompts when running `ddev config DrudS3`
// 0. Get AWS keys if they don't already exist
// 1. Get bucketnames accessible
// 2. If only one bucket, choose it
// 3. Get projects, this project (same exact name) must exist in the bucket, choose it
// 4. Get environments, if only one, choose it
func (p *DrudS3Provider) PromptForConfig() error {

	var err error
	if p.AWSAccessKey != "" && p.AWSSecretKey != "" {
		util.Success("AWS Access Key ID and AWS Secret Access Key already configured in .ddev/import.yaml")
	} else {
		p.AWSAccessKey = util.Prompt("AWS access key id", p.AWSAccessKey)
		p.AWSSecretKey = util.Prompt("AWS secret access key", p.AWSSecretKey)
	}
	_, client, err := p.getDrudS3Session()
	if err != nil {
		return fmt.Errorf("could not  get s3 session: %v", err)
	}

	// Get buckets
	bucketsAvailable, err := p.getBucketList()
	if err != nil {
		return fmt.Errorf("unable to list buckets: %v", err)
	}
	if len(bucketsAvailable) == 0 {
		return fmt.Errorf("no buckets are accessible with the provided AWS credentials")
	}
	if len(bucketsAvailable) == 1 {
		p.S3Bucket = bucketsAvailable[0]
		util.Success("Only one accessible bucket (%s), so using it.", p.S3Bucket)
	} else {
		bucketNameString := strings.Join(bucketsAvailable, ", ")
		promptWithBuckets := fmt.Sprintf("AWS S3 Bucket Name [%s]", bucketNameString)
		p.S3Bucket = util.Prompt(promptWithBuckets, p.S3Bucket)
	}

	projects, err := getDrudS3Projects(client, p.S3Bucket)
	if err != nil {
		return fmt.Errorf("could not getDrudS3Projects: %v", err)
	}
	if _, ok := projects[p.app.Name]; !ok {
		return fmt.Errorf("project name %s has no backups in S3 bucket %s", p.app.Name, p.S3Bucket)
	}

	environments, err := p.GetEnvironments()
	if err != nil {
		return fmt.Errorf("unable to GetEnvironments: %v", err)
	}
	envAry := util.MapKeysToArray(environments)
	if len(envAry) == 1 {
		p.EnvironmentName = envAry[0]
		util.Success("Only one environment is available for project %s, environment is set to '%s'", p.app.Name, p.EnvironmentName)
		return nil
	}

	envNames := strings.Join(envAry, ", ")
	fullPrompt := fmt.Sprintf("pantheonEnvironment Name [%s]", envNames)
	p.EnvironmentName = util.Prompt(fullPrompt, p.EnvironmentName)

	return nil
}

// GetBackup will download the most recent backup specified by backupType in the given environment. If no environment
// is supplied, the configured environment will be used. Valid values for backupType are "database" or "files".
func (p *DrudS3Provider) GetBackup(backupType, environment string) (fileLocation string, importPath string, err error) {
	if backupType != "database" && backupType != "files" {
		return "", "", fmt.Errorf("could not get backup: %s is not a valid backup type", backupType)
	}

	// If the user hasn't defined an environment override, use the configured value.
	if environment == "" {
		environment = p.EnvironmentName
	}

	// Set the import path (within the archive) blank, to use the root of the archive by default.
	importPath = ""
	err = p.environmentExists(environment)
	if err != nil {
		return "", "", err
	}

	sess, client, err := p.getDrudS3Session()
	if err != nil {
		return "", "", err
	}

	prefix := "db"
	if backupType == "files" {
		prefix = "files"
	}
	object, err := getLatestS3Object(client, p.S3Bucket, p.app.Name+"/"+environment+"/"+prefix)
	if err != nil {
		return "", "", fmt.Errorf("unable to getLatestS3Object for bucket %s project %s environment %s prefix %s, %v", p.S3Bucket, p.app.Name, environment, prefix, err)
	}

	// Check to see if this file has been downloaded previously.
	// Attempt a new download If we can't stat the file or we get a mismatch on the filesize.
	destFile := filepath.Join(p.getDownloadDir(), path.Base(*object.Key))

	stat, err := os.Stat(destFile)
	if err != nil || stat.Size() != int64(*object.Size) {
		p.prepDownloadDir()

		err = downloadS3Object(sess, p.S3Bucket, object, p.getDownloadDir())
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
	globalDir := globalconfig.GetGlobalDdevDir()
	destDir := filepath.Join(globalDir, "drud-s3", p.app.Name)

	return destDir
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
	environments, err := p.findDrudS3Project()
	if err != nil {
		return nil, err
	}
	return environments, nil
}

// Validate ensures that the current configuration is valid (i.e. the configured DrudS3 site/environment exists)
// If the environment exists, the project exists, and the AWS keys are working right.
func (p *DrudS3Provider) Validate() error {
	return p.environmentExists(p.EnvironmentName)
}

// environmentExists ensures the currently configured DrudS3 site & environment exists.
func (p *DrudS3Provider) environmentExists(environment string) error {
	environments, err := p.GetEnvironments()
	if err != nil {
		return err
	}
	if _, ok := environments[environment]; !ok {
		return fmt.Errorf("could not find an environment with backups named '%s'", environment)
	}

	return nil
}

// getBucketList returns an array of buckets available with current S3 public/private keys
func (p *DrudS3Provider) getBucketList() ([]string, error) {
	_, client, err := p.getDrudS3Session()
	if err != nil {
		return nil, err
	}
	result, err := client.ListBuckets(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to list S3 buckets: %v", err)
	}

	buckets := []string{}
	for _, b := range result.Buckets {
		buckets = append(buckets, aws.StringValue(b.Name))
	}
	return buckets, nil
}

// findDrudS3Project ensures the DrudS3 site specified by name exists, and the current user has access to it.
// It returns an array of environment names and err
func (p *DrudS3Provider) findDrudS3Project() (map[string]interface{}, error) {
	_, client, err := p.getDrudS3Session()
	if err != nil {
		return nil, err
	}
	// Get a list of all projects the current user has access to.
	projectMap, err := getDrudS3Projects(client, p.S3Bucket)
	if err != nil {
		return nil, err
	}

	projectWithEnvs := projectMap[p.app.Name]

	// Get a list of environments for a given project.
	return projectWithEnvs, nil
}

// getDrudS3Session loads the DrudS3 API config from disk and returns a DrudS3 session struct.
func (p *DrudS3Provider) getDrudS3Session() (*session.Session, *s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(p.AWSAccessKey, p.AWSSecretKey, ""),
	},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to NewSession: %v", err)
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
		components := strings.Split(strings.TrimRight(*obj.Key, "/"), "/")
		// We're only interested in items that have an actual dump in them.
		if (len(components)) == 3 {
			if _, ok := projectMap[components[0]]; !ok {
				tmp := make(map[string]interface{})
				projectMap[components[0]] = tmp
			}
			projectMap[components[0]][components[1]] = true
		}
	}
	return projectMap, nil
}

// getS3ObjectsWithPrefix gets all S3 objects in the named bucket with the prefix provided.
// Examples at https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/s3-example-basic-bucket-operations.html
// and at https://github.com/awsdocs/aws-doc-sdk-examples/tree/master/go/example_code/s3
// S3 api description at https://docs.aws.amazon.com/AmazonS3/latest/API/v2-RESTBucketGET.html
// ListObjectsPages example https://gist.github.com/eferro/651fbb72851fa7987fc642c8f39638eb
func getS3ObjectsWithPrefix(client *s3.S3, bucket string, prefix string) ([]*s3.Object, error) {
	maxKeys := aws.Int64(100)

	query := &s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: maxKeys,
	}
	var allObjs []*s3.Object
	err := client.ListObjectsPages(query, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, value := range page.Contents {
			allObjs = append(allObjs, value)
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ListObjectsPages: %v", err)
	}

	if len(allObjs) == 0 {
		return nil, fmt.Errorf("there are no objects matching %s in bucket %s", prefix, bucket)
	}
	return allObjs, nil
}

// getLatestS3Object gets the most recently modified key in the named bucket
// with the provided prefix.
func getLatestS3Object(client *s3.S3, bucket string, prefix string) (*s3.Object, error) {
	allObjs, err := getS3ObjectsWithPrefix(client, bucket, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to getS3ObjectsWithPrefix: %v", err)
	}
	sort.Sort(byModified(allObjs))
	return allObjs[0], nil
}

// downloadS3Object grabs the object named and brings it down to the directory named
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
		util.Success("File %s was already downloaded, skipping", localPath)
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

	util.Success("Downloaded file %s (%d bytes)", file.Name(), numBytes)
	return nil
}
