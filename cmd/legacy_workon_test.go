package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"os/user"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLegacyWorkonCancel runs `drud legacy workon` and selects 0.
func TestLegacyWorkonCancel(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(DrudBin, "dev", "workon")
	cmd.Stdin = strings.NewReader("0\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), "0: Cancel")
}

func getLegacyTestID() string {
	cmd := exec.Command(DrudBin, "dev", "workon")
	cmd.Stdin = strings.NewReader("0\n")

	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	testApp := LegacyTestApp + "-" + LegacyTestEnv
	testOption := "0"
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			r := strings.NewReplacer(" ", "")
			l := strings.Split(r.Replace(line), ":")
			if l[1] == testApp {
				testOption = l[0]
			}
		}
	}
	return testOption
}

// TestLegacyWorkon runs `drud legacy workon` and selects our legacy test app.
func TestLegacyWorkon(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(DrudBin, "dev", "workon")
	selection := getLegacyTestID()
	cmd.Stdin = strings.NewReader(selection + "\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), fmt.Sprintf("You are now working on %s-%s", LegacyTestApp, LegacyTestEnv))

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	fb, err := ioutil.ReadFile(usr.HomeDir + "/drud.yaml")
	assert.Contains(string(fb), "activeapp: "+LegacyTestApp)
	assert.Contains(string(fb), "activedeploy: "+LegacyTestEnv)
}
