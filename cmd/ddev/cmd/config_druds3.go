package cmd

import (
	"fmt"
	"github.com/drud/ddev/pkg/nodeps"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/spf13/cobra"
)

// drudS3EnvironmentName is the environment for non-default providers, dev/test/prod
var drudS3EnvironmentName string

// Flag variables for drud-s3 provider
var drudS3awsAccessKeyID string
var drudS3awsSecretAccessKey string
var drudS3Bucket string

// drudS3ConfigCommand is the the `ddev config drud-s3` command
var drudS3ConfigCommand *cobra.Command = &cobra.Command{
	Use:     "drud-s3",
	Short:   "Create or modify a ddev project drud-s3 configuration in the current directory",
	Example: `"ddev config drud-s3" or "ddev config drud-s3 --access-key-id=AKIAISOMETHINGMAGIC --secret-access-key=rweeMAGICSECRET --docroot=. --project-name=d7-kickstart --project-type=drupal7 --bucket-name=my_aws_s3_bucket --environment=production"`,
	PreRun: func(cmd *cobra.Command, args []string) {
		providerName = nodeps.ProviderDrudS3
		extraFlagsHandlingFunc = handleDrudS3Flags
	},
	Run: handleConfigRun,
}

func init() {
	drudS3ConfigCommand.Flags().AddFlagSet(ConfigCommand.Flags())
	drudS3ConfigCommand.Flags().StringVarP(&drudS3awsAccessKeyID, "access-key-id", "", "", "drud-s3 only: AWS S3 access key ID")
	drudS3ConfigCommand.Flags().StringVarP(&drudS3awsSecretAccessKey, "secret-access-key", "", "", "drud-s3 only: AWS S3 secret access key")
	drudS3ConfigCommand.Flags().StringVarP(&drudS3Bucket, "bucket-name", "", "", "drud-s3 only: AWS S3 bucket")
	drudS3ConfigCommand.Flags().StringVarP(&drudS3EnvironmentName, "environment", "", "", "Choose the environment for a  project (production/staging/etc)")

	ConfigCommand.AddCommand(drudS3ConfigCommand)
}

// handleDrudS3Flags is the drud-s3 specific flag handler
func handleDrudS3Flags(cmd *cobra.Command, args []string, app *ddevapp.DdevApp) error {
	provider, err := app.GetProvider()
	if err != nil {
		return fmt.Errorf("failed to GetProvider: %v", err)
	}
	drudS3Provider := provider.(*ddevapp.DrudS3Provider)
	drudS3Provider.AWSAccessKey = drudS3awsAccessKeyID
	drudS3Provider.AWSSecretKey = drudS3awsSecretAccessKey
	drudS3Provider.S3Bucket = drudS3Bucket
	drudS3Provider.EnvironmentName = drudS3EnvironmentName
	return nil
}
