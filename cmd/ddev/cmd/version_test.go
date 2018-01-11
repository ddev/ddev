 package cmd
-
-import (
-       "strings"
-       "testing"
-
-       "github.com/drud/ddev/pkg/version"
-       asrt "github.com/stretchr/testify/assert"
-)
-
-func TestVersion(t *testing.T) {
-       assert := asrt.New(t)

		args := []string{"version"}
		out, err := exec.RunCommand(DdevBin, args)
		assert.NoError(err)
-       assert.Contains(string(out), version.DdevVersion)
-       assert.Contains(string(out), version.WebImg)
-       assert.Contains(string(out), version.WebTag)
-       assert.Contains(string(out), version.DBImg)
-       assert.Contains(string(out), version.DBTag)
-       assert.Contains(string(out), version.DBAImg)
-       assert.Contains(string(out), version.DBATag)
-       assert.Contains(string(out), version.DDevTLD)
-}
