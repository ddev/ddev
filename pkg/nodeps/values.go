package nodeps

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
	PHP74 = "7.4"
)

// MariaDB Versions
const (
	MariaDB55  = "5.5"
	MariaDB100 = "10.0"
	MariaDB101 = "10.1"
	MariaDB102 = "10.2"
	MariaDB103 = "10.3"
	MariaDB104 = "10.4"
)

// Oracle MySQL versions
const (
	MySQL55 = "5.5"
	MySQL56 = "5.6"
	MySQL57 = "5.7"
	MySQL80 = "8.0"
)

// Database Types
const (
	MariaDB = "mariadb"
	MySQL   = "mysql"
)

// Container types used with ddev
const (
	DdevSSHAgentContainer = "ddev-ssh-agent"
	DBAContainer          = "dba"
	DBContainer           = "db"
	WebContainer          = "web"
	RouterContainer       = "ddev-router"
)

// PHPDefault is the default PHP version, overridden by $DDEV_PHP_VERSION
const PHPDefault = PHP73

// ValidPHPVersions should be updated whenever PHP versions are added or removed, and should
// be used to ensure user-supplied values are valid.
var ValidPHPVersions = map[string]bool{
	PHP56: true,
	PHP70: true,
	PHP71: true,
	PHP72: true,
	PHP73: true,
	PHP74: true,
}

var ValidMariaDBVersions = map[string]bool{
	MariaDB55:  true,
	MariaDB100: true,
	MariaDB101: true,
	MariaDB102: true,
	MariaDB103: true,
	MariaDB104: true,
}

var ValidMySQLVersions = map[string]bool{
	MySQL55: true,
	MySQL56: true,
	MySQL57: true,
	MySQL80: true,
}

// Webserver types
const (
	WebserverNginxFPM  = "nginx-fpm"
	WebserverApacheFPM = "apache-fpm"
	WebserverApacheCGI = "apache-cgi"
)

var ValidOmitContainers = map[string]bool{
	DBContainer:           true,
	DdevSSHAgentContainer: true,
	DBAContainer:          true,
}

// WebserverDefault is the default webserver type, overridden by $DDEV_WEBSERVER_TYPE
var WebserverDefault = WebserverNginxFPM

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
	AppTypeDrupal9   = "drupal9"
	AppTypePHP       = "php"
	AppTypeTYPO3     = "typo3"
	AppTypeWordPress = "wordpress"
	AppTypeMagento   = "magento"
	AppTypeMagento2  = "magento2"
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

// IsValidMySQLVersion is a helper function to determine if a MySQL version is valid, returning
// true if the supplied version is valid and false otherwise.
func IsValidMySQLVersion(v string) bool {
	if _, ok := ValidMySQLVersions[v]; !ok {
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

// GetValidMySQLVersions is a helper function that returns a list of valid MySQL versions.
func GetValidMySQLVersions() []string {
	s := make([]string, 0, len(ValidMySQLVersions))

	for p := range ValidMySQLVersions {
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
