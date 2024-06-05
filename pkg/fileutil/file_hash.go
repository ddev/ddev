package fileutil

import (
	"crypto/sha1"
	"fmt"
	"github.com/ddev/ddev/pkg/util"
	"io"
	"os"
)

// FileHash returns string of hash of filePath passed in
// And optional string can be added to content that will be hashed
func FileHash(filePath string, optionalExtraString string) (string, error) {
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
	// Include file location in the hash, if in a different
	// place it should not hash the same
	// file.Name() is the full path of the file
	if _, err := hash.Write([]byte(file.Name())); err != nil {
		return "", err
	}

	// Add optional string to hash if provided
	if len(optionalExtraString) > 0 {
		if _, err := hash.Write([]byte(optionalExtraString)); err != nil {
			return "", err
		}
	}

	sum := hash.Sum(nil)

	return fmt.Sprintf("%x", sum), nil
}
