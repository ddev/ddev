package fileutil

import (
	"crypto/sha1"
	"fmt"
	"github.com/ddev/ddev/pkg/util"
	"io"
	"os"
)

// FileHash returns string of hash of filePath passed in
func FileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		err = file.Close()
		if err != nil {
			util.Warning("unable to close file: %v", err)
		}
	}()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	sum := hash.Sum(nil)

	return fmt.Sprintf("%x", sum), nil
}
