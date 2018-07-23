package testinteraction

import (
	"fmt"

	"github.com/drud/ddev/pkg/ddevapp"
	"github.com/drud/ddev/pkg/exec"
	"gopkg.in/headzoo/surf.v1"
)

// wordpressInteraction defines information specific to Wordpress interactions.
type wordpressInteraction struct {
	*commonInteraction

	adminEmail string
	title      string
}

// NewWordpressInteractor returns a struct pointer that satisfies the Interactor interface
// specific to a Wordpress project.
func NewWordpressInteractor(app *ddevapp.DdevApp) Interactor {
	c := &commonInteraction{
		adminUsername: "ddevusername",
		adminPassword: "ddevpassword",
		rootDir:       app.AppRoot,
		browser:       surf.NewBrowser(),
		baseURL:       app.GetHTTPURL(),
		loginPath:     fmt.Sprintf("%s/wp-login.php", app.GetHTTPURL()),
		authedPath:    "/wp-admin/",
	}

	return &wordpressInteraction{
		adminEmail:        "admin@test.site",
		title:             app.GetName(),
		commonInteraction: c,
	}
}

// Login will attempt a Wordpress admin login.
func (w *wordpressInteraction) Login() error {
	var err error
	if err = w.browser.Open(w.loginPath); err != nil {
		return err
	}

	fm, err := w.browser.Form("form#loginform")
	if err != nil {
		return err
	}

	if err = fm.Input("log", w.adminUsername); err != nil {
		return err
	}

	if err = fm.Input("pwd", w.adminPassword); err != nil {
		return err
	}

	if err = fm.Submit(); err != nil {
		return err
	}

	if w.browser.Url().Path != w.authedPath {
		return fmt.Errorf("login did not redirect to expected location, expected: %s, got: %s", w.authedPath, w.browser.Url().Path)
	}

	return nil
}

// Install will attempt a non-interactive installation of a Wordpress project.
func (w *wordpressInteraction) Install() error {
	args := []string{
		"exec",
		"wp",
		"core",
		"install",
		fmt.Sprintf("--admin_user=%s", w.adminUsername),
		fmt.Sprintf("--admin_password=%s", w.adminPassword),
		fmt.Sprintf("--url=%s", w.baseURL),
		fmt.Sprintf("--title=%s", w.title),
		fmt.Sprintf("--admin_email=%s", w.adminEmail),
	}

	if _, err := exec.RunCommandInDir("ddev", args, w.rootDir); err != nil {
		return err
	}

	return nil
}

// Configure will attempt to configure a Wordpress project, a prerequisite for interaction.
func (w *wordpressInteraction) Configure() error {
	if err := w.configure("htdocs", w.title, "wordpress"); err != nil {
		return err
	}

	return nil
}
