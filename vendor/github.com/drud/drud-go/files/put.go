package files

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Put will upload a file from localFilePath to the remote destination.
func (fs *FileService) Put(localFilePath, destination string) (*s3manager.UploadOutput, error) {
	file, err := os.Open(localFilePath)
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(session.New(&aws.Config{Region: aws.String(fs.Region)}))
	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   file,
		Bucket: aws.String(fs.Bucket),
		Key:    aws.String(destination),
	})

	return result, err

}
