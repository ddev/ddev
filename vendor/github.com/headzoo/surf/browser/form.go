package browser

import (
	"net/url"
	"strings"

	"io"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf/errors"
)

// Submittable represents an element that may be submitted, such as a form.
type Submittable interface {
	Method() string
	Action() string
	Input(name, value string) error
	Set(name, value string) error
	File(name string, fileName string, data io.Reader) error
	SetFile(name string, fileName string, data io.Reader)
	Click(button string) error
	ClickByValue(name, value string) error
	Submit() error
	Dom() *goquery.Selection
}

// Form is the default form element.
type Form struct {
	bow       Browsable
	selection *goquery.Selection
	method    string
	action    string
	fields    url.Values
	buttons   url.Values
	files     FileSet
}

// NewForm creates and returns a *Form type.
func NewForm(bow Browsable, s *goquery.Selection) *Form {
	fields, buttons, files := serializeForm(s)
	method, action := formAttributes(bow, s)

	return &Form{
		bow:       bow,
		selection: s,
		method:    method,
		action:    action,
		fields:    fields,
		buttons:   buttons,
		files:     files,
	}
}

// Method returns the form method, eg "GET" or "POST".
func (f *Form) Method() string {
	return f.method
}

// Action returns the form action URL.
// The URL will always be absolute.
func (f *Form) Action() string {
	return f.action
}

// Input sets the value of a form field.
// it returns an ElementNotFound error if the field does not exists
func (f *Form) Input(name, value string) error {
	if _, ok := f.fields[name]; ok {
		f.fields.Set(name, value)
		return nil
	}
	return errors.NewElementNotFound(
		"No input found with name '%s'.", name)
}

// File sets the value for an form input type file,
// it returns an ElementNotFound error if the field does not exists
func (f *Form) File(name string, fileName string, data io.Reader) error {

	if _, ok := f.files[name]; ok {
		f.files[name] = &File{fileName: fileName, data: data}
		return nil
	}
	return errors.NewElementNotFound(
		"No input type 'file' found with name '%s'.", name)
}

// SetFile sets the value for an form input type file,
// It adds the field to the form if necessary
func (f *Form) SetFile(name string, fileName string, data io.Reader) {
	f.files[name] = &File{fileName: fileName, data: data}
}

// Set will set the value of a form field if it exists,
// or create and set it if it does not.
func (f *Form) Set(name, value string) error {
	if _, ok := f.fields[name]; !ok {
		f.fields.Add(name, value)
		return nil
	}
	return f.Input(name, value)
}

// Submit submits the form.
// Clicks the first button in the form, or submits the form without using
// any button when the form does not contain any buttons.
func (f *Form) Submit() error {
	if len(f.buttons) > 0 {
		for name := range f.buttons {
			return f.Click(name)
		}
	}
	return f.send("", "")
}

// Click submits the form by clicking the button with the given name.
func (f *Form) Click(button string) error {
	if _, ok := f.buttons[button]; !ok {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s'.", button)
	}
	return f.send(button, f.buttons[button][0])
}

// Click submits the form by clicking the button with the given name and value.
func (f *Form) ClickByValue(name, value string) error {
	if _, ok := f.buttons[name]; !ok {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s'.", name)
	}
	valueNotFound := true
	for _, val := range f.buttons[name] {
		if val == value {
			valueNotFound = false
			break
		}
	}
	if valueNotFound {
		return errors.NewInvalidFormValue(
			"Form does not contain a button with the name '%s' and value '%s'.", name, value)
	}
	return f.send(name, value)
}

// Dom returns the inner *goquery.Selection.
func (f *Form) Dom() *goquery.Selection {
	return f.selection
}

// send submits the form.
func (f *Form) send(buttonName, buttonValue string) error {
	method, ok := f.selection.Attr("method")
	if !ok {
		method = "GET"
	}
	action, ok := f.selection.Attr("action")
	if !ok {
		action = f.bow.Url().String()
	}
	aurl, err := url.Parse(action)
	if err != nil {
		return err
	}
	aurl = f.bow.ResolveUrl(aurl)

	values := make(url.Values, len(f.fields)+1)
	for name, vals := range f.fields {
		values[name] = vals
	}
	if buttonName != "" {
		values.Set(buttonName, buttonValue)
	}

	if strings.ToUpper(method) == "GET" {
		return f.bow.OpenForm(aurl.String(), values)
	}
	enctype, _ := f.selection.Attr("enctype")
	if enctype == "multipart/form-data" {
		return f.bow.PostMultipart(aurl.String(), values, f.files)
	}
	return f.bow.PostForm(aurl.String(), values)
}

// serializeForm converts the form fields into a url.Values type.
// Returns two url.Value types. The first is the form field values, and the
// second is the form button values.
func serializeForm(sel *goquery.Selection) (url.Values, url.Values, FileSet) {
	input := sel.Find("input,button,textarea,select")
	if input.Length() == 0 {
		return url.Values{}, url.Values{}, FileSet{}
	}
	fields := make(url.Values)
	buttons := make(url.Values)
	files := make(FileSet)
	input.Each(func(_ int, s *goquery.Selection) {
		if name, ok := s.Attr("name"); ok {
			t, _ := s.Attr("type")
			if t == "submit" {
				val, _ := s.Attr("value")
				buttons.Add(name, val)
			} else if t == "file" {
				files[name] = &File{}
			} else {
				elementName := s.First().Nodes[0].Data
				switch elementName {
				case "input", "textarea":
					val, _ := s.Attr("value")
					fields.Add(name, val)
				case "select":
					val, _ := s.Find("option:first-child").Attr("value")
					fields.Add(name, val)
				}
			}
		}
	})
	return fields, buttons, files
}

func formAttributes(bow Browsable, s *goquery.Selection) (string, string) {
	method, ok := s.Attr("method")
	if !ok {
		method = "GET"
	}
	action, ok := s.Attr("action")
	if !ok {
		action = bow.Url().String()
	}
	aurl, err := url.Parse(action)
	if err != nil {
		return "", ""
	}
	aurl = bow.ResolveUrl(aurl)

	return strings.ToUpper(method), aurl.String()
}
