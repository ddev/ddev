package testinteraction

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/drud/ddev/pkg/exec"
	"github.com/headzoo/surf/browser"
)

// commonInteraction holds values that all interaction structs will need, allowing for common
// methods to be defined.
type commonInteraction struct {
	adminUsername string
	adminPassword string
	authedPath    string
	baseURL       string
	browser       *browser.Browser
	loginPath     string
	rootDir       string
}

// configure receives specific configuration information from downstream implementations of Interactor,
// running the ddev config command in the application's root directory.
func (c *commonInteraction) configure(docroot, projectName, projectType string) error {
	args := []string{"config", "--docroot", docroot, "--projectname", projectName, "--projecttype", projectType}
	if _, err := exec.RunCommandInDir("ddev", args, c.rootDir); err != nil {
		return nil
	}

	return nil
}

// LoadURL accepts a URL and an HTTP status, performing a GET request on the URL and
// ensuring the response's status is the expected HTTP status.
func (c *commonInteraction) LoadURL(url string, expectedStatus int) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("expected status code %d, got %d", expectedStatus, resp.StatusCode)
	}

	return nil
}

// FindContentAtPath will open the supplied path and search for the provided expression,
// returning an error if no matches are found on the page.
func (c *commonInteraction) FindContentAtPath(contentPath string, contentExpr string) error {
	var err error

	// TODO: Find a better way to build URLs.
	urlString := fmt.Sprintf("%s/%s", c.baseURL, contentPath)
	if err = c.browser.Open(urlString); err != nil {
		return err
	}

	match, err := regexp.Match(contentExpr, []byte(c.browser.Body()))
	if err != nil {
		return err
	}

	if !match {
		return fmt.Errorf("page at url: %s does not contain expression: %s", urlString, contentExpr)
	}

	return nil
}
