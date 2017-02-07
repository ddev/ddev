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

// TestDevWorkonCancel runs `drud legacy workon` and selects 0.
func TestDevWorkonCancel(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(DrudBin, "dev", "workon")
	cmd.Stdin = strings.NewReader("0\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), "0: Cancel")
}

func getDevTestID() string {
	cmd := exec.Command(DrudBin, "dev", "workon")
	cmd.Stdin = strings.NewReader("0\n")

	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	testApp := DevTestApp + "-" + DevTestEnv
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

// TestDevWorkon runs `drud legacy workon` and selects our legacy test app.
func TestDevWorkon(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(DrudBin, "dev", "workon")
	selection := getDevTestID()
	cmd.Stdin = strings.NewReader(selection + "\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), fmt.Sprintf("You are now working on %s-%s", DevTestApp, DevTestEnv))

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	fb, err := ioutil.ReadFile(usr.HomeDir + "/drud.yaml")
	assert.Contains(string(fb), "activeapp: "+DevTestApp)
	assert.Contains(string(fb), "activedeploy: "+DevTestEnv)
}
