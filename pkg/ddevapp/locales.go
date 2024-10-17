package ddevapp

import (
	"bufio"
	"fmt"
	"github.com/ddev/ddev/pkg/archive"
	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/util"
	"os"
	"path/filepath"
	"strings"
)

// GetGlobalLocalesFilepath returns the expected location of the locales.txt
func GetGlobalLocalesFilepath() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), "bin", "locales.txt")
}

// DownloadLocalesIfNeeded downloads the locales master list if it doesn't already exist.
func DownloadLocalesIfNeeded() {
	globalLocalesFile := GetGlobalLocalesFilepath()
	if fileutil.FileExists(globalLocalesFile) {
		return
	}
	// TODO: Provide the actual URL
	localesURL := "https://github.com/ddev/ddev/scripts/locales.tar.gz"
	d, cleanup, err := archive.DownloadAndExtractTarball(localesURL, false)
	if err == nil {
		// use it
		err = fileutil.CopyFile(filepath.Join(d, "locales.txt"), globalLocalesFile)
		if err != nil {
			util.Warning("Unable to copy master locales file. This doesn't matter for most people: %v", err)
		}
		cleanup()
	} else {
		util.Warning("Unable to download master locales file. This doesn't matter for most people: %v", err)
	}
}

type LocaleDesc struct {
	shortName string
	encoding  string
}

// LoadLocales loads locales from a text file into a map for use in
// creating /etc/locale.gen
func LoadLocales() (map[string]LocaleDesc, error) {
	globalLocalesFile := GetGlobalLocalesFilepath()

	localeMap := make(map[string]LocaleDesc)

	file, err := os.Open(globalLocalesFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			fmt.Println("Skipping malformed line:", scanner.Text())
			continue
		}

		key := fields[0]
		localeMap[key] = LocaleDesc{
			shortName: fields[1],
			encoding:  fields[2],
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return localeMap, nil
}
