package files

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// ListFiles will list files of a given prefix, until listLimit is reached.
func (fs *FileService) ListFiles(prefix string, listLimit int) ([]string, error) {
	params := &s3.ListObjectsInput{
		Bucket: aws.String(fs.Bucket),
	}
	args := []string{prefix}
	if prefix != "" {
		params.Prefix = &args[0]
	}

	resp, err := fs.Connection.ListObjects(params)
	if err != nil {
		return []string{}, err
	}
	var output []string
	for i, key := range resp.Contents {
		if i < listLimit {
			output = append(output, *key.Key)
		}
	}

	return output, nil
}
