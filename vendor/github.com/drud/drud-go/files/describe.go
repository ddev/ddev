package files

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// DescribeFile will describe a file defined by key, within a given s3 bucket.
func (fs *FileService) DescribeFile(key string) (*s3.GetObjectOutput, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(fs.Bucket), // Required
		Key:    aws.String(key),       // Required
	}
	resp, err := fs.Connection.GetObject(params)

	return resp, err
}
