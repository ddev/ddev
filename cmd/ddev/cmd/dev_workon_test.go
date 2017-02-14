package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"testing"
	"github.com/drud/drud-go/utils"

	"github.com/stretchr/testify/assert"
)

// TestDevWorkonCancel runs `drud legacy workon` and selects 0.
func TestDevWorkonCancel(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(DdevBin, "workon")
	cmd.Stdin = strings.NewReader("0\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), "0: Cancel")
}

func getDevTestID() string {
	cmd := exec.Command(DdevBin, "workon")
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
	cmd := exec.Command(DdevBin, "workon")
	selection := getDevTestID()
	cmd.Stdin = strings.NewReader(selection + "\n")

	out, err := cmd.Output()
	assert.NoError(err)
	assert.Contains(string(out), fmt.Sprintf("You are now working on %s-%s", DevTestApp, DevTestEnv))

	home, _ := utils.GetHomeDir()
	fb, err := ioutil.ReadFile(home + "/drud.yaml")
	assert.Contains(string(fb), "activeapp: "+DevTestApp)
	assert.Contains(string(fb), "activedeploy: "+DevTestEnv)
}
