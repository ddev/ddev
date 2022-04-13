package ddevapp_test

import (
	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	asrt "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetInactiveProjects(t *testing.T) {
	assert := asrt.New(t)

	// Make sure we have no running sites.
	// Pause all sites.
	_, err := exec.RunCommand(DdevBin, []string{"pause", "--all"})
	require.NoError(t, err)

	apps, err := ddevapp.GetInactiveProjects()
	require.NoError(t, err)

	assert.Greater(len(apps), 0)
}

func TestExtractProjectNames(t *testing.T) {
	var apps []*ddevapp.DdevApp

	assert := asrt.New(t)

	apps = append(apps, &ddevapp.DdevApp{Name: "bar"})
	apps = append(apps, &ddevapp.DdevApp{Name: "foo"})
	apps = append(apps, &ddevapp.DdevApp{Name: "zoz"})

	names := ddevapp.ExtractProjectNames(apps)
	expectedNames := []string{"bar", "foo", "zoz"}

	assert.Equal(expectedNames, names)
}
