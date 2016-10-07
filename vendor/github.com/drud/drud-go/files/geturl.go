package files

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// GetURL will get a temporary download link for a given file.
func (fs *FileService) GetURL(filename string, expires int) (string, error) {
	req, _ := fs.Connection.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(fs.Bucket),
		Key:    aws.String(filename),
	})

	urlStr, err := req.Presign(time.Duration(expires) * time.Hour)

	return urlStr, err
}
