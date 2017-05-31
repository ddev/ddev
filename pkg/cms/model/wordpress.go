package model

import "github.com/drud/drud-go/utils/stringutil"

// WordpressConfig encapsulates all the configurations for a WordPress site.
type WordpressConfig struct {
	WPGeneric        bool
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	AuthKey          string
	SecureAuthKey    string
	LoggedInKey      string
	NonceKey         string
	AuthSalt         string
	SecureAuthSalt   string
	LoggedInSalt     string
	NonceSalt        string
	Docroot          string
	TablePrefix      string
}

// NewWordpressConfig produces a WordpressConfig object with defaults.
func NewWordpressConfig() *WordpressConfig {
	return &WordpressConfig{
		WPGeneric:        false,
		DatabaseName:     "db",
		DatabaseUsername: "db",
		DatabasePassword: "db",
		DatabaseHost:     "db",
		Docroot:          "/var/www/html/docroot",
		TablePrefix:      "wp_",
		AuthKey:          stringutil.RandomString(64),
		AuthSalt:         stringutil.RandomString(64),
		LoggedInKey:      stringutil.RandomString(64),
		LoggedInSalt:     stringutil.RandomString(64),
		NonceKey:         stringutil.RandomString(64),
		NonceSalt:        stringutil.RandomString(64),
		SecureAuthKey:    stringutil.RandomString(64),
		SecureAuthSalt:   stringutil.RandomString(64),
	}
}
