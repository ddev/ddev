package docker

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// DdevImageTagLabel is the image label recording the tag an image was built as.
// The tag is otherwise lost in a derived image, so this is what lets DDEV tell
// which of its image generations a pinned webimage/dbimage came from.
const DdevImageTagLabel = "com.ddev.image-tag"

// dockerOrgEnvVar overrides the organization the repositories in
// versionconstants.go live under. A ddev-test/ddev build publishes its images
// to `ddevhq` rather than `ddev`, so without this its own release cannot be
// pull-tested. Unset, which is every normal build, nothing changes.
// See ddev/ddev#8753.
const dockerOrgEnvVar = "DDEV_DOCKER_ORG"

// imageRepo returns an image reference with its organization replaced by
// dockerOrgEnvVar, when that is set. A reference with no organization (the
// upstream `postgres`) has nothing to replace and is returned unchanged.
func imageRepo(image string) string {
	org := strings.Trim(strings.TrimSpace(os.Getenv(dockerOrgEnvVar)), "/")
	i := strings.LastIndex(image, "/")
	if org == "" || i < 0 {
		return image
	}
	return org + image[i:]
}

// releaseTagPattern matches a vX.Y.Z release tag.
var releaseTagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// ResolveImageTag prefers branch over tag when branch is a vX.Y.Z release
// tag. release-prep.sh stamps <TagVar>Branch with the release tag while
// leaving <TagVar> as the content hash autotag.sh relies on, but both
// resolve to the identical manifest, so a released binary can pull by the
// readable tag.
func ResolveImageTag(tag, branch string) string {
	if releaseTagPattern.MatchString(branch) {
		return branch
	}
	return tag
}

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	fullWebImg := imageRepo(versionconstants.WebImg)
	if globalconfig.DdevGlobalConfig.UseHardenedImages {
		fullWebImg = fullWebImg + "-prod"
	}
	return fmt.Sprintf("%s:%s", fullWebImg, ResolveImageTag(versionconstants.WebTag, versionconstants.WebTagBranch))
}

// GetDBImage returns the correctly formatted db image:tag reference
func GetDBImage(dbType string, dbVersion string) string {
	v := nodeps.MariaDBDefaultVersion
	if dbVersion != "" {
		v = dbVersion
	}
	if dbType == "" {
		dbType = nodeps.MariaDB
	}
	switch dbType {
	case nodeps.Postgres:
		return fmt.Sprintf("%s:%s", dbType, v)
	case nodeps.MySQL:
		fallthrough
	case nodeps.MariaDB:
		fallthrough
	default:
		return fmt.Sprintf("%s-%s-%s:%s", imageRepo(versionconstants.DBImg), dbType, v, ResolveImageTag(versionconstants.BaseDBTag, versionconstants.BaseDBTagBranch))
	}
}

// GetSSHAuthImage returns the correctly formatted sshauth image:tag reference
func GetSSHAuthImage() string {
	return fmt.Sprintf("%s:%s", imageRepo(versionconstants.SSHAuthImage), ResolveImageTag(versionconstants.SSHAuthTag, versionconstants.SSHAuthTagBranch))
}

// GetRouterImage returns the router image:tag reference
func GetRouterImage() string {
	return fmt.Sprintf("%s:%s", imageRepo(versionconstants.TraefikRouterImage), ResolveImageTag(versionconstants.TraefikRouterTag, versionconstants.TraefikRouterTagBranch))
}

// GetXhguiImage returns the xhgui image:tag reference
func GetXhguiImage() string {
	return fmt.Sprintf("%s:%s", imageRepo(versionconstants.XhguiImage), ResolveImageTag(versionconstants.XhguiTag, versionconstants.XhguiTagBranch))
}
