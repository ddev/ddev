package nodeps

import "sort"

// Providers
//TODO: This should be removed as many providers will now be valid
const (
	ProviderPantheon = "pantheon"
	ProviderDdevLive = "ddev-live"
	// ProviderDefault contains the name of the default provider which will be used if one is not otherwise specified.
	ProviderDefault = "default"
)

// ValidProviders should be updated whenever provider plugins are added or removed, and should
// be used to ensure user-supplied values are valid.
//TODO: This should be removed as many providers will now be valid
var ValidProviders = map[string]bool{
	ProviderDefault:  true,
	ProviderPantheon: true,
	ProviderDdevLive: true,
}

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

// Webserver types
const (
	WebserverNginxFPM  = "nginx-fpm"
	WebserverApacheFPM = "apache-fpm"
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

// FailOnHookFailDefault is the default value for app.FailOnHookFail
var FailOnHookFailDefault = false

// ValidWebserverTypes should be updated whenever supported webserver types are added or
// removed, and should be used to ensure user-supplied values are valid.
var ValidWebserverTypes = map[string]bool{
	WebserverNginxFPM:  true,
	WebserverApacheFPM: true,
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
	AppTypeLaravel   = "laravel"
	AppTypeShopware6 = "shopware6"
)

// Ports and other defaults
const (
	// DdevDefaultRouterHTTPPort is the default router HTTP port
	DdevDefaultRouterHTTPPort = "80"

	// DdevDefaultRouterHTTPSPort is the default router HTTPS port
	DdevDefaultRouterHTTPSPort = "443"
	// DdevDefaultPHPMyAdminPort is the default router port for dba/PHPMyadmin
	DdevDefaultPHPMyAdminPort      = "8036"
	DdevDefaultPHPMyAdminHTTPSPort = "8037"
	// DdevDefaultMailhogPort is the default router port for Mailhog
	DdevDefaultMailhogPort      = "8025"
	DdevDefaultMailhogHTTPSPort = "8026"
	// DdevDefaultTLD is the top-level-domain used by default, can be overridden
	DdevDefaultTLD                  = "ddev.site"
	InternetDetectionTimeoutDefault = 750
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
	sort.Strings(s)
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
	sort.Strings(s)
	return s
}

// GetValidMySQLVersions is a helper function that returns a list of valid MySQL versions.
func GetValidMySQLVersions() []string {
	s := make([]string, 0, len(ValidMySQLVersions))

	for p := range ValidMySQLVersions {
		s = append(s, p)
	}
	sort.Strings(s)
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
