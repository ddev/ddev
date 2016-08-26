package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	drud "github.com/drud/bootstrap/cli/cmd"
	"gopkg.in/yaml.v2"
)

var binpath string

func init() {
	var err error
	binpath, err = exec.LookPath("drud")
	if err != nil {
		log.Fatal("could not find drud")
	}
}

/**
 * TestSecretString writes a string as a secret then ensures the secret can be
 * read, listed, and deleted.
 */
func TestSecretString(t *testing.T) {
	testname := "testcase1"
	testval := "someval"

	// write test val to test/testname
	log.Printf("Writing string '%s' to 'secret/test/%s'\n", testval, testname)
	out, err := exec.Command(binpath, "secret", "write", "test/"+testname, testval).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), "Creating secret at"), true)

	// read test secret
	log.Printf("Reading secret from 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testval), true)

	// list test secret
	log.Println("Listing secrets in 'secret/test'")
	out, err = exec.Command(binpath, "secret", "list", "test").Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testname), true)

	// delete test secret
	log.Printf("Deleting secret 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "delete", "-y", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testname+" deleted"), true)

	// verify secret gone
	log.Printf("Rereading secret from 'secret/test/%s'\n", testname)
	_, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	refute(t, err, nil)

}

// creates a secret from key/value pairs and then readds, lists, deletes
func TestSecretKeyVal(t *testing.T) {
	testname := "testcase2"
	testval1 := "name=bob"
	testval2 := "job=builder"

	// write test val to test/testname
	log.Printf("Writing string '%s %s' to 'secret/test/%s'\n", testval1, testval2, testname)
	out, err := exec.Command(binpath, "secret", "write", "test/"+testname, testval1, testval2).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), "Creating secret at"), true)

	// read test secret
	log.Printf("Reading secret from 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}

	yaml1 := strings.Replace(string(testval1), "=", ": ", -1)
	yaml2 := strings.Replace(string(testval2), "=", ": ", -1)
	expect(t, strings.Contains(string(out), yaml1), true)
	expect(t, strings.Contains(string(out), yaml2), true)

	// list test secret
	log.Println("Listing secrets in 'secret/test'")
	out, err = exec.Command(binpath, "secret", "list", "test").Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testname), true)

	// delete test secret
	log.Printf("Deleting secret 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "delete", "-y", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}

	expect(t, strings.Contains(string(out), testname+" deleted"), true)

	// verify secret gone
	log.Printf("Rereading secret from 'secret/test/%s'\n", testname)
	_, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	refute(t, err, nil)

}

// creates a secret from file and then readds, lists, deletes
func TestSecretFile(t *testing.T) {
	testname := "testcase3"
	testContent := make(map[string]string)
	testContent["test"] = "case"

	testJSON, err := json.Marshal(testContent)
	if err != nil {
		log.Fatal(err)
	}

	testYAML, err := yaml.Marshal(testContent)
	if err != nil {
		log.Fatal(err)
	}

	file, err := ioutil.TempFile(os.TempDir(), "temp")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	file.Write(testJSON)

	// write test val to test/testname
	log.Printf("Writing file '%s' to 'secret/test/%s'\n", file.Name(), testname)
	out, err := exec.Command(binpath, "secret", "write", "test/"+testname, "@"+file.Name()).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), "Creating secret at"), true)

	// read test secret
	log.Printf("Reading secret from 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}

	expect(t, strings.Contains(string(out), string(testYAML)), true)

	// list test secret
	log.Println("Listing secrets in 'secret/test'")
	out, err = exec.Command(binpath, "secret", "list", "test").Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testname), true)

	// delete test secret
	log.Printf("Deleting secret 'secret/test/%s'\n", testname)
	out, err = exec.Command(binpath, "secret", "delete", "-y", "test/"+testname).Output()
	if err != nil {
		log.Fatal(err)
	}
	expect(t, strings.Contains(string(out), testname+" deleted"), true)

	// verify secret gone
	log.Printf("Rereading secret from 'secret/test/%s'\n", testname)
	_, err = exec.Command(binpath, "secret", "read", "test/"+testname).Output()
	refute(t, err, nil)

}

// TestNormalizePath ensures input paths are translated correctly to what vault expects
func TestNormalizePath(t *testing.T) {
	tests := [][]string{
		[]string{"secret", "secret/secret"},
		[]string{"secret/", "secret/secret"},
		[]string{"/", "secret"},
		[]string{"test", "secret/test"},
		[]string{"test/", "secret/test"},
		[]string{"test/tacos", "secret/test/tacos"},
	}
	for _, v := range tests {
		expect(t, v[1], drud.NormalizePath(v[0]))
	}
}

// TestConfigCreate makes sure that when a config does not exist one will be created
func TestConfigCreate(t *testing.T) {
	conf := "testconf-igf73ilelkjsd834.yaml"
	_, err := exec.Command(binpath, "--config="+conf, "version").Output()
	if err != nil {
		log.Fatalln("failure runnign command:", err)
	}
	defer os.Remove(conf)

}
