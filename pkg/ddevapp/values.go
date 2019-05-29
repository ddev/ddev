package ddevapp

// Providers
const (
	ProviderDrudS3   = "drud-s3"
	ProviderPantheon = "pantheon"

	// ProviderDefault contains the name of the default provider which will be used if one is not otherwise specified.
	ProviderDefault = "default"
)

// ValidProviders should be updated whenever provider plugins are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidProviders = map[string]bool{
	ProviderDefault:  true,
	ProviderDrudS3:   true,
	ProviderPantheon: true,
}

// PHP Versions
const (
	PHP56 = "5.6"
	PHP70 = "7.0"
	PHP71 = "7.1"
	PHP72 = "7.2"
	PHP73 = "7.3"
)

// MariaDB Versions
const (
	MariaDB101 = "10.1"
	MariaDB102 = "10.2"
)

// Container types used with ddev
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBAContainer          = "dba"
	DBContainer           = "db"
	WebContainer          = "web"
	RouterContainer       = "ddev-router"
	BGSYNCContainer       = "bgsync"
)

// PHPDefault is the default PHP version, overridden by $DDEV_PHP_VERSION
const PHPDefault = PHP72

// ValidPHPVersions should be updated whenever PHP versions are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidPHPVersions = map[string]bool{
	PHP56: true,
	PHP70: true,
	PHP71: true,
	PHP72: true,
	PHP73: true,
}

var ValidMariaDBVersions = map[string]bool{
	MariaDB101: true,
	MariaDB102: true,
}

// Webserver types
const (
	WebserverNginxFPM  = "nginx-fpm"
	WebserverApacheFPM = "apache-fpm"
	WebserverApacheCGI = "apache-cgi"
)

var ValidOmitContainers = map[string]bool{
	DdevSSHAgentContainer: true,
	DBAContainer:          true,
}

// WebserverDefault is the default webserver type, overridden by $DDEV_WEBSERVER_TYPE
var WebserverDefault = WebserverNginxFPM

// WebcacheEnabledDefault is the default value for app.WebCacheEnabled
var WebcacheEnabledDefault = false

// NFSMountEnabledDefault is default value for app.NFSMountEnabled
var NFSMountEnabledDefault = false

// ValidWebserverTypes should be updated whenever supported webserver types are added or
// removed, and should be used to ensure user-supplied values are valid.
var ValidWebserverTypes = map[string]bool{
	WebserverNginxFPM:  true,
	WebserverApacheFPM: true,
	WebserverApacheCGI: true,
}

// App types
const (
	AppTypeBackdrop  = "backdrop"
	AppTypeDrupal6   = "drupal6"
	AppTypeDrupal7   = "drupal7"
	AppTypeDrupal8   = "drupal8"
	AppTypePHP       = "php"
	AppTypeTYPO3     = "typo3"
	AppTypeWordPress = "wordpress"
)

// Ports and other defaults
const (
	// DdevDefaultRouterHTTPPort is the default router HTTP port
	DdevDefaultRouterHTTPPort = "80"

	// DdevDefaultRouterHTTPSPort is the default router HTTPS port
	DdevDefaultRouterHTTPSPort = "443"
	// DdevDefaultPHPMyAdminPort is the default router port for dba/PHPMyadmin
	DdevDefaultPHPMyAdminPort = "8036"
	// DdevDefaultMailhogPort is the default router port for Mailhog
	DdevDefaultMailhogPort = "8025"
	// DdevDefaultTLD is the top-level-domain used by default, can be overridden
	DdevDefaultTLD = "ddev.site"
)

// IsValidProvider is a helper function to determine if a provider value is valid, returning
// true if the supplied provider is valid and false otherwise.
func IsValidProvider(provider string) bool {
	if _, ok := ValidProviders[provider]; !ok {
		return false
	}

	return true
}

// GetValidProviders is a helper function that returns a list of valid providers.
func GetValidProviders() []string {
	s := make([]string, 0, len(ValidProviders))

	for p := range ValidProviders {
		s = append(s, p)
	}

	return s
}

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

	return s
}

// IsValidMariaDBVersion is a helper function to determine if a MariaDB version is valid, returning
// true if the supplied MariaDB version is valid and false otherwise.
func IsValidMariaDBVersion(MariaDBVersion string) bool {
	if _, ok := ValidMariaDBVersions[MariaDBVersion]; !ok {
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

// IsValidAppType is a helper function to determine if an app type is valid, returning
// true if the given app type is valid and configured and false otherwise.
func IsValidAppType(apptype string) bool {
	if _, ok := appTypeMatrix[apptype]; !ok {
		return false
	}

	return true
}

// GetValidAppTypes returns the valid apptype keys from the appTypeMatrix
func GetValidAppTypes() []string {
	keys := make([]string, 0, len(appTypeMatrix))
	for k := range appTypeMatrix {
		keys = append(keys, k)
	}
	return keys
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
