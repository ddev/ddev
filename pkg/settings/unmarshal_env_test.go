package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestViperUnmarshalEnv(t *testing.T) {
	os.Setenv("DDEV_TEST_PORT", "8888")
	defer os.Unsetenv("DDEV_TEST_PORT")

	type Config struct {
		TestPort string `yaml:"test_port"`
	}

	p := NewConfigProvider()

	var cfg Config
	err := p.Unmarshal(&cfg)
	assert.NoError(t, err)

	assert.Equal(t, "", cfg.TestPort, "Unmarshal SHOULD NOT pick up unbound environment variables")
	assert.Equal(t, "8888", p.GetString("test_port"), "GetString SHOULD pick up environment variables via AutomaticEnv")
}

func TestViperUnmarshalEnvStandardDDEV(t *testing.T) {
	v := NewConfigProvider()

	type Config struct {
		RouterHTTPPort string `yaml:"router_http_port"`
	}

	_ = os.Setenv("DDEV_ROUTER_HTTP_PORT", "9999")
	defer os.Unsetenv("DDEV_ROUTER_HTTP_PORT")

	var cfg Config
	err := v.Unmarshal(&cfg)
	assert.NoError(t, err)

	// This should now be "9999" because it's bound in NewConfigProvider
	assert.Equal(t, "9999", cfg.RouterHTTPPort)
}
