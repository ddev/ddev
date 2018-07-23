package testinteraction

import "github.com/drud/ddev/pkg/ddevapp"

// Interactor defines an interface for interacting with projects of any type.
type Interactor interface {
	Configure() error
	FindContentAtPath(path string, expression string) error
	Login() error
	Install() error
}

// NewInteractor will inspect the provided app and return an instantiated Interactor
// if one is available. If no Interactor is defined for the app type, NewInteractor
// will return nil.
func NewInteractor(app *ddevapp.DdevApp) Interactor {
	switch app.Type {
	case "wordpress":
		return NewWordpressInteractor(app)
	}

	return nil
}
