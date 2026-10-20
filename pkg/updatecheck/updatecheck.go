package updatecheck

import (
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ddev/ddev/pkg/github"
	"github.com/ddev/ddev/pkg/util"
)

// AvailableUpdates returns true (along with a release URL) if there is an update available in the specified repo which is newer than the currentVersion string.
// Dev builds never report an update, and skip the GitHub API call entirely.
func AvailableUpdates(repoOrg string, repoName string, currentVersion string) (avail bool, newVersion string, releaseURL string, err error) {
	if !isReleaseVersion(currentVersion) {
		return false, "", "", nil
	}
	latestTag, latestURL, err := github.GetLatestRelease(repoOrg, repoName)
	if err != nil {
		return false, "", "", err
	}
	newer, err := isNewerVersion(currentVersion, latestTag)
	if err != nil || !newer {
		return false, latestTag, "", err
	}
	return true, latestTag, latestURL, nil
}

// isNewerVersion reports whether latestVersion is a higher semver than currentVersion.
func isNewerVersion(currentVersion, latestVersion string) (bool, error) {
	cv, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false, err
	}
	lv, err := semver.NewVersion(latestVersion)
	if err != nil {
		return false, err
	}
	return cv.LessThan(lv), nil
}

// IsUpdateNeeded returns true if the modification date on filepath is older than the duration specified.
func IsUpdateNeeded(filepath string, updateInterval time.Duration) (bool, error) {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return true, ResetUpdateTime(filepath)
	} else if err != nil {
		return false, err
	}

	timeSinceMod := time.Since(info.ModTime())

	if timeSinceMod >= updateInterval {
		return true, nil
	}

	return false, nil
}

// ResetUpdateTime resets the file modification date on filepath by removing and re-creating the file.
func ResetUpdateTime(filepath string) error {
	err := os.Remove(filepath)
	_ = err // We don't actually care if remove failed. All we care about is that the create succeeds.
	file, err := os.Create(filepath)
	util.CheckClose(file)
	return err
}

// isReleaseVersion does a (very naive) check on whether a version string consistutes a release version or a dev build.
func isReleaseVersion(version string) bool {
	parts := strings.Split(version, "-")

	if len(parts) > 2 || strings.Contains(version, "dirty") || strings.Contains(version, "-g") {
		return false
	}

	return true
}
