package platform

import "os"

// Config is just a placeholder for when we get a little farther and have config in each dir
// TODO: Replace all this. It is only intended for a very short lifespan in March, 2017
type Config struct {
	ActiveApp    string
	ActiveDeploy string
	Workspace    string
}

// GetConfig Loads a config structure from yaml and environment.
func GetConfig() (cfg *Config, err error) {
	workspace := os.Getenv("HOME") + "/.drud/local/"
	c := &Config{ActiveApp: "", ActiveDeploy: "", Workspace: workspace}
	return c, nil
}
