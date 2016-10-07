package files

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// GetFile downloads a file from the DRUD file service and writes it to the path defined by localFilePath
func (fs *FileService) GetFile(localFilePath, fileName string) (*os.File, int64, error) {
	file, err := os.Create(localFilePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(session.New(&aws.Config{Region: aws.String(fs.Region)}))
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(fs.Bucket),
			Key:    aws.String(fileName),
		})
	if err != nil {
		// Remove the local file if we had a failure, and return the error.
		os.Remove(localFilePath)
		return nil, 0, err
	}

	return file, numBytes, nil
}
