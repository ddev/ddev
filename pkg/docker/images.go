package docker

import (
	"fmt"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/globalconfig/types"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/versionconstants"
)

// GetWebImage returns the correctly formatted web image:tag reference
func GetWebImage() string {
	fullWebImg := versionconstants.WebImg
	if globalconfig.DdevGlobalConfig.UseHardenedImages {
		fullWebImg = fullWebImg + "-prod"
	}
	return fmt.Sprintf("%s:%s", fullWebImg, versionconstants.WebTag)
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
		return fmt.Sprintf("%s-%s-%s:%s", versionconstants.DBImg, dbType, v, versionconstants.BaseDBTag)
	}
}

// GetSSHAuthImage returns the correctly formatted sshauth image:tag reference
func GetSSHAuthImage() string {
	return fmt.Sprintf("%s:%s", versionconstants.SSHAuthImage, versionconstants.SSHAuthTag)
}

// GetRouterImage returns the router image:tag reference
func GetRouterImage() string {
	image := versionconstants.TraefikRouterImage
	if globalconfig.DdevGlobalConfig.Router == types.RouterTypeNginxProxy {
		image = versionconstants.TraditionalRouterImage
	}
	return image
}
