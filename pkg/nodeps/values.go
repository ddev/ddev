package nodeps

import (
	"sort"
	"strings"

	"github.com/maruel/natural"

	"github.com/ddev/ddev/pkg/config/types"
)

// Providers
// TODO: This should be removed as many providers will now be valid
const (
	// ProviderDefault contains the name of the default provider which will be used if one is not otherwise specified.
	ProviderDefault = "default"
)

// Database Types
const (
	MariaDB  = "mariadb"
	MySQL    = "mysql"
	Postgres = "postgres"
)

// MySQLRemoveDeprecatedMessage is used to remove the deprecation from stderr when using mysql tools with MariaDB 11.x and later.
// mysql: Deprecated program name. It will be removed in a future release, use '/usr/bin/mariadb' instead
// mysqldump: Deprecated program name. It will be removed in a future release, use '/usr/bin/mariadb-dump' instead
const MySQLRemoveDeprecatedMessage = " 2> >(grep -v 'Deprecated program name.' >&2) "

// Container types used with ddev
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBContainer           = "db"
	WebContainer          = "web"
	RouterContainer       = "ddev-router"
)

// Webserver types
const (
	WebserverNginxFPM    = "nginx-fpm"
	WebserverApacheFPM   = "apache-fpm"
	WebserverNginxNodeJS = "nginx-nodejs"
)

// ValidOmitContainers is the list of things that can be omitted
var ValidOmitContainers = map[string]bool{
	DBContainer:           true,
	DdevSSHAgentContainer: true,
}

// DdevFileSignature is the text we use to detect whether a settings file is managed by us.
// If this string is found, we assume we can replace/update the file.
const DdevFileSignature = "#ddev-generated"

// WebserverDefault is the default webserver type, overridden by $DDEV_WEBSERVER_TYPE
var WebserverDefault = WebserverNginxFPM

// PerformanceModeDefault is default value for app.PerformanceMode
var PerformanceModeDefault = types.PerformanceModeEmpty

const NodeJSDefault = "22"

// NoBindMountsDefault is default value for globalconfig.DDEVGlobalConfig.NoBindMounts
var NoBindMountsDefault = false

// UseNginxProxyRouter is used in testing to override the default
// setting for tests.
var UseNginxProxyRouter = false

// SimpleFormatting is turned on by DDEV_USE_SIMPLE_FORMATTING
// and makes ddev list and describe, etc. use simpler formatting
var SimpleFormatting = false

// FailOnHookFailDefault is the default value for app.FailOnHookFail
var FailOnHookFailDefault = false

// GoroutineLimit is the number of goroutines allowed at exit in parts of some tests
// Can be overridden by setting DDEV_TEST_GOROUTINE_LIMIT=<somenumber>
var GoroutineLimit = 10

// ValidWebserverTypes should be updated whenever supported webserver types are added or
// removed, and should be used to ensure user-supplied values are valid.
var ValidWebserverTypes = map[string]bool{
	WebserverNginxFPM:    true,
	WebserverApacheFPM:   true,
	WebserverNginxNodeJS: true,
}

const AppTypeDrupalLatestStable = AppTypeDrupal11

// App types
const (
	AppTypeNone     = ""
	AppTypeBackdrop = "backdrop"
	AppTypeCakePHP  = "cakephp"
	AppTypeCraftCms = "craftcms"
	AppTypeDrupal6  = "drupal6"
	AppTypeDrupal7  = "drupal7"
	AppTypeDrupal8  = "drupal8"
	AppTypeDrupal9  = "drupal9"
	AppTypeDrupal10 = "drupal10"
	AppTypeDrupal11 = "drupal11"
	// AppTypeDrupal is an alias for "most recent Drupal version"
	AppTypeDrupal       = "drupal"
	AppTypeLaravel      = "laravel"
	AppTypeNodeJS       = "nodejs"
	AppTypeSilverstripe = "silverstripe"
	AppTypeSymfony      = "symfony"
	AppTypeMagento      = "magento"
	AppTypeMagento2     = "magento2"
	AppTypePHP          = "php"
	AppTypeShopware6    = "shopware6"
	AppTypeTYPO3        = "typo3"
	AppTypeWordPress    = "wordpress"
)

// Ports and other defaults
const (
	// DdevDefaultRouterHTTPPort is the default router HTTP port
	DdevDefaultRouterHTTPPort = "80"

	// DdevDefaultRouterHTTPSPort is the default router HTTPS port
	DdevDefaultRouterHTTPSPort = "443"
	// DdevDefaultMailpitHTTPPort is the default router port for Mailpit
	DdevDefaultMailpitHTTPPort  = "8025"
	DdevDefaultMailpitHTTPSPort = "8026"
	// DdevDefaultTLD is the top-level-domain used by default, can be overridden
	DdevDefaultTLD                  = "ddev.site"
	DefaultDefaultContainerTimeout  = "120"
	InternetDetectionTimeoutDefault = 3000
	TraefikMonitorPortDefault       = "10999"
	MinimumDockerSpaceWarning       = 5000000 // 5GB in KB (to compare against df reporting in KB)
)

// IsValidPHPVersion is a helper function to determine if a PHP version is valid, returning
// true if the supplied PHP version is valid and false otherwise.
func IsValidPHPVersion(phpVersion string) bool {
	if _, ok := ValidPHPVersions[phpVersion]; !ok {
		return false
	}

	return true
}

// GetValidPHPVersions is a helper function that returns a list of valid PHP versions.
func GetValidPHPVersions() []string {
	s := make([]string, 0, len(ValidPHPVersions))

	for p := range ValidPHPVersions {
		s = append(s, p)
	}
	sort.Sort(natural.StringSlice(s))
	return s
}

// IsValidDatabaseVersion checks if the version is valid for the provided database type
func IsValidDatabaseVersion(dbType string, dbVersion string) bool {
	switch dbType {
	case MariaDB:
		return IsValidMariaDBVersion(dbVersion)
	case MySQL:
		return IsValidMySQLVersion(dbVersion)
	case Postgres:
		return IsValidPostgresVersion(dbVersion)
	}
	return false
}

// GetValidDatabaseVersions returns a slice of valid versions with the format
// mariadb:10.5/mysql:5.7/postgres:14
func GetValidDatabaseVersions() []string {
	combos := []string{}
	for _, v := range GetValidMariaDBVersions() {
		combos = append(combos, MariaDB+":"+v)
	}
	for _, v := range GetValidMySQLVersions() {
		combos = append(combos, MySQL+":"+v)
	}
	for _, v := range GetValidPostgresVersions() {
		combos = append(combos, Postgres+":"+v)
	}

	return combos
}

// IsValidMariaDBVersion is a helper function to determine if a MariaDB version is valid, returning
// true if the supplied MariaDB version is valid and false otherwise.
func IsValidMariaDBVersion(v string) bool {
	if _, ok := ValidMariaDBVersions[strings.TrimPrefix(v, MariaDB+":")]; !ok {
		return false
	}

	return true
}

// IsValidMySQLVersion is a helper function to determine if a MySQL version is valid, returning
// true if the supplied version is valid and false otherwise.
func IsValidMySQLVersion(v string) bool {
	if _, ok := ValidMySQLVersions[strings.TrimPrefix(v, MySQL+":")]; !ok {
		return false
	}

	return true
}

// GetValidMariaDBVersions is a helper function that returns a list of valid MariaDB versions.
func GetValidMariaDBVersions() []string {
	s := make([]string, 0, len(ValidMariaDBVersions))

	for p := range ValidMariaDBVersions {
		s = append(s, p)
	}
	sort.Sort(natural.StringSlice(s))
	return s
}

// IsValidPostgresVersion is a helper function to determine if a PostgreSQL version is valid, returning
// true if the supplied version is valid and false otherwise.
func IsValidPostgresVersion(v string) bool {
	if _, ok := ValidPostgresVersions[strings.TrimPrefix(v, Postgres+":")]; !ok {
		return false
	}

	return true
}

// GetValidMySQLVersions is a helper function that returns a list of valid MySQL versions.
func GetValidMySQLVersions() []string {
	s := make([]string, 0, len(ValidMySQLVersions))

	for p := range ValidMySQLVersions {
		s = append(s, p)
	}
	sort.Sort(natural.StringSlice(s))
	return s
}

// GetValidPostgresVersions is a helper function that returns a list of valid PostgreSQL versions.
func GetValidPostgresVersions() []string {
	s := make([]string, 0, len(ValidPostgresVersions))

	for p := range ValidPostgresVersions {
		s = append(s, p)
	}
	sort.Sort(natural.StringSlice(s))
	return s
}

// IsValidWebserverType is a helper function to determine if a webserver type is valid, returning
// true if the supplied webserver type is valid and false otherwise.
func IsValidWebserverType(webserverType string) bool {
	if _, ok := ValidWebserverTypes[webserverType]; !ok {
		return false
	}

	return true
}

// GetValidWebserverTypes is a helper function that returns a list of valid webserver types.
func GetValidWebserverTypes() []string {
	s := make([]string, 0, len(ValidWebserverTypes))

	for p := range ValidWebserverTypes {
		s = append(s, p)
	}

	return s
}

// IsValidOmitContainers is a helper function to determine if a the OmitContainers array is valid
func IsValidOmitContainers(containerList []string) bool {
	for _, containerName := range containerList {
		if _, ok := ValidOmitContainers[containerName]; !ok {
			return false
		}
	}
	return true
}

// GetValidOmitContainers is a helper function that returns a list of valid containers for OmitContainers.
func GetValidOmitContainers() []string {
	s := make([]string, 0, len(ValidOmitContainers))

	for p := range ValidOmitContainers {
		s = append(s, p)
	}

	return s
}
