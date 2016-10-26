package files

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// DeleteFile will remove a file from an s3 bucket.
func (fs *FileService) DeleteFile(key string) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(fs.Bucket), // Required
		Key:    aws.String(key),       // Required
	}

	_, err := fs.Connection.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}
