package model

// DrupalConfig encapsulates all the configurations for a Drupal site.
type DrupalConfig struct {
	DeployName       string
	DeployURL        string
	DatabaseName     string
	DatabaseUsername string
	DatabasePassword string
	DatabaseHost     string
	DatabaseDriver   string
	DatabasePort     int
	DatabasePrefix   string
}

// NewDrupalConfig produces a DrupalConfig object with default.
func NewDrupalConfig() *DrupalConfig {
	return &DrupalConfig{
		DatabaseName:     "data",
		DatabaseUsername: "root",
		DatabasePassword: "root",
		DatabaseHost:     "127.0.0.1",
		DatabaseDriver:   "mysql",
		DatabasePort:     3306,
		DatabasePrefix:   "",
	}
}
