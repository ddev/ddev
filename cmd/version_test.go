package cmd

import (
	"log"
	"os/exec"
	"strings"
	"testing"
)

func init() {
	setup()
}

func TestVersion(t *testing.T) {
	log.Printf("%s version should be '%s'\n", binary, cliVersion)
	v, err := exec.Command(binary, "version").Output()
	if err != nil {
		t.Errorf("Error executing %s version", binary)
	}
	output := strings.TrimSpace(string(v))
	expect(t, output, cliVersion)
}
