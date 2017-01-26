package cmd

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/drud/bootstrap/cli/local"
	"github.com/drud/drud-go/utils"
	"github.com/stretchr/testify/assert"
)

var configTest = []struct {
	config   string
	app      string
	env      string
	client   string
	host     string
	protocol string
}{
	{utils.RandomString(10) + ".yaml", "drudio", "production", "snuffleupagus", "drudapi.turtleduck.drud.io", "http"},
	{utils.RandomString(10) + ".yaml", "nonsense", "staging", "barney", "drudapi.bowbell.drud.io", "https"},
	{utils.RandomString(10) + ".yaml", "bologna", "default", "bigbird", "drudapi.genesis.drud.io", "https"},
}

// TestIntegrationConfigSet uses the compiled binary to test config set operation
func TestIntegrationConfigSet(t *testing.T) {
	for _, c := range configTest {
		args := []string{
			"config", "set",
			"--config", c.config,
			"--activeapp", c.app,
			"--activedeploy", c.env,
			"--client", c.client,
			"--drudhost", c.host,
			"--protocol", c.protocol,
		}
		out, err := utils.RunCommand(DrudBin, args)
		assert.NoError(t, err)
		assert.Contains(t, string(out), "Config items set.")

		fbytes, err := ioutil.ReadFile(c.config)
		assert.NoError(t, err)
		assert.Contains(t, string(fbytes), "client: "+c.client)
		assert.Contains(t, string(fbytes), "activeapp: "+c.app)
		assert.Contains(t, string(fbytes), "activedeploy: "+c.env)
		assert.Contains(t, string(fbytes), "drudhost: "+c.host)
		assert.Contains(t, string(fbytes), "protocol: "+c.protocol)

		err = os.Remove(c.config)
		assert.NoError(t, err)
	}
}

// TestUnitConfigSet uses internal functions to test the config set operation
func TestUnitConfigSet(t *testing.T) {
	var err error

	for _, c := range configTest {
		cfg, err = local.GetConfig()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// make sure the config file has been created
		cfgFile = c.config
		PrepConf()

		// set these globals vars to they will be used in setConfigItems
		client = c.client
		activeApp = c.app
		activeDeploy = c.env
		drudHost = c.host
		protocol = c.protocol

		setCmd.Run(RootCmd, []string{})

		fbytes, err := ioutil.ReadFile(c.config)
		assert.NoError(t, err)
		assert.Contains(t, string(fbytes), "client: "+c.client)
		assert.Contains(t, string(fbytes), "activeapp: "+c.app)
		assert.Contains(t, string(fbytes), "activedeploy: "+c.env)
		assert.Contains(t, string(fbytes), "drudhost: "+c.host)
		assert.Contains(t, string(fbytes), "protocol: "+c.protocol)

		err = os.Remove(c.config)
		assert.NoError(t, err)
	}
}
