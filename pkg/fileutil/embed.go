package fileutil

import (
	"embed"
	"github.com/ddev/ddev/pkg/nodeps"
	"os"
	"path"
	"path/filepath"
)

// CopyEmbedAssets copies files in the named embed.FS sourceDir to the local targetDir (full path)
func CopyEmbedAssets(fsys embed.FS, sourceDir string, targetDir string) error {
	subdirs, err := fsys.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, d := range subdirs {
		sourcePath := path.Join(sourceDir, d.Name())
		if d.IsDir() {
			err = CopyEmbedAssets(fsys, path.Join(sourceDir, d.Name()), path.Join(targetDir, d.Name()))
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
				err = os.WriteFile(localPath, content, 0755)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
