package fileutil

import (
	"embed"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/ddev/ddev/pkg/nodeps"
)

// CopyEmbedAssets copies files in the named embed.FS sourceDir to the local targetDir (full path)
// Some files may be excluded if they are in the excludedFiles list and contain #ddev-generated.
func CopyEmbedAssets(fsys embed.FS, sourceDir string, targetDir string, excludedFiles []string) error {
	subdirs, err := fsys.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, d := range subdirs {
		sourcePath := path.Join(sourceDir, d.Name())
		if d.IsDir() {
			err = CopyEmbedAssets(fsys, path.Join(sourceDir, d.Name()), path.Join(targetDir, d.Name()), excludedFiles)
			if err != nil {
				return err
			}
		} else {
			localPath := filepath.Join(targetDir, d.Name())

			// We can overwrite the file if it has the #ddev-generated
			// or if it is an empty file.
			sigFound, err := FgrepStringInFile(localPath, nodeps.DdevFileSignature)
			s, _ := os.Stat(localPath)
			if sigFound || (s != nil && s.Size() == 0) || err != nil {
				content, err := fsys.ReadFile(sourcePath)
				if err != nil {
					return err
				}
				err = os.MkdirAll(filepath.Dir(localPath), 0755)
				if err != nil {
					return err
				}
				if sigFound {
					// If the file already exists and has the same content, don't overwrite it.
					if existingContent, err := os.ReadFile(localPath); err == nil && string(existingContent) == string(content) {
						continue
					}
					// If the file already exists and is excluded, don't overwrite it.
					if excludedFiles != nil && slices.Contains(excludedFiles, localPath) {
						continue
					}
				}
				err = os.WriteFile(localPath, content, 0755)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
