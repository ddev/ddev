package files

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// FileService allows you to interact with DRUD files.
type FileService struct {
	Connection *s3.S3
	Bucket     string
	Region     string
}

// NewFileService returns a fileservice struct
func NewFileService(awsID, awsSecret, region, bucket string) (*FileService, error) {
	os.Setenv("AWS_ACCESS_KEY_ID", awsID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsSecret)
	return &FileService{
		Connection: s3.New(session.New(&aws.Config{Region: aws.String(region)})),
		Bucket:     bucket,
		Region:     region,
	}, nil
}
