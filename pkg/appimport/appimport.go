package appimport

import (
	"errors"
	"fmt"
	"os"

	"strings"

	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
)

// ValidateAsset determines if a given asset matches the required criteria for a given asset type
// and returns the absolute path to the asset, whether or not the asset is an archive type, and an error.
func ValidateAsset(unexpandedAssetPath string, assetType string) (string, bool, error) {
	var invalidAssetError = "invalid asset: %v"
	extensions := []string{"tar", "gz", "tgz", "zip", "bz2", "xz"}

	// Input provided via prompt or "--flag=value" is not expanded by shell. This will help ensure ~ is expanded to the user home directory.
	assetPath, err := homedir.Expand(unexpandedAssetPath)
	if err != nil {
		return "", false, fmt.Errorf(invalidAssetError, err)
	}

	// ensure we are working w/ an absolute path
	assetPath, err = filepath.Abs(assetPath)
	if err != nil {
		return "", false, fmt.Errorf(invalidAssetError, err)
	}

	info, err := os.Stat(assetPath)
	if os.IsNotExist(err) {
		return "", false, fmt.Errorf(invalidAssetError, errors.New("file not found"))
	}
	if err != nil {
		return "", false, fmt.Errorf(invalidAssetError, err)
	}

	for _, ext := range extensions {
		if strings.HasSuffix(assetPath, ext) {
			return assetPath, true, nil
		}
	}

	if assetType == "files" && !info.IsDir() {
		return "", false, fmt.Errorf(invalidAssetError, errors.New("provided path is not a directory or archive"))
	}

	if assetType == "db" && assetPath != "" && !strings.HasSuffix(assetPath, "sql") {
		return "", false, fmt.Errorf(invalidAssetError, errors.New("provided path is not a .sql file or archive"))
	}

	return assetPath, false, nil
}
