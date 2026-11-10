package tui

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/interrupt"
	"github.com/ddev/ddev/pkg/nodeps"
)

// ErrConfigCancelled is returned by ConfigForm when the user leaves it
// without saving.
var ErrConfigCancelled = errors.New("configuration cancelled")

// ConfigForm shows the settings DDEV detects for app in a form where each can
// be changed, and configures app with them when the user saves.
func ConfigForm(app *ddevapp.DdevApp) error {
	// Bubble Tea restores the terminal itself on SIGINT and SIGTERM.
	releaseInterrupt := interrupt.Delegate(os.Interrupt, syscall.SIGTERM)
	final, err := tea.NewProgram(newConfigForm(app)).Run()
	releaseInterrupt()
	if err != nil {
		return fmt.Errorf("configuration form error: %v", err)
	}
	form := final.(configForm)
	if !form.saved {
		return ErrConfigCancelled
	}
	return app.ApplyConfigChoices(form.choices)
}

// The rows of the form, in order. The save row comes first in focus, so
// that enter alone accepts the detected configuration.
const (
	rowName = iota
	rowDocroot
	rowType
	rowPHP
	rowDBType
	rowDBVersion
	rowWebserver
	rowSave
	rowCount
)

// pickerHeight is how many of a setting's values are listed at a time.
const pickerHeight = 8

var configRowLabels = [rowCount]string{"Project name", "Docroot", "Project type", "PHP version", "Database type", "Database version", "Webserver type"}

type configFormKeys struct {
	Up, Down, Edit, Choose, Save, Close, Cancel key.Binding
}

type configForm struct {
	app       *ddevapp.DdevApp
	choices   ddevapp.ConfigChoices
	suggested ddevapp.ConfigChoices
	// changed holds the rows the user set, which a new project type's
	// defaults don't replace.
	changed map[int]bool
	focus   int
	// input edits the name or docroot while its row has the focus.
	input textinput.Model
	// picking is set while the values of the focused row are listed, with
	// picked the index of the highlighted one.
	picking bool
	picked  int
	problem string
	saved   bool
	quit    bool
	styles  Styles
	help    help.Model
	keys    configFormKeys
}

func newConfigForm(app *ddevapp.DdevApp) configForm {
	styles := NewStyles(nil, nil)
	input := textinput.New()
	input.Prompt = ""
	input.SetStyles(styles.filter())
	h := help.New()
	h.Styles = styles.help()
	suggested := app.SuggestedConfigChoices()
	return configForm{
		app:       app,
		choices:   suggested,
		suggested: suggested,
		changed:   map[int]bool{},
		focus:     rowSave,
		input:     input,
		styles:    styles,
		help:      h,
		keys: configFormKeys{
			Up:     key.NewBinding(key.WithKeys("up", "shift+tab"), key.WithHelp("↑/↓", "move")),
			Down:   key.NewBinding(key.WithKeys("down", "tab")),
			Edit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("type", "edit")),
			Choose: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")),
			Save:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			Close:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			Cancel: key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel")),
		},
	}
}

func (f configForm) Init() tea.Cmd {
	return nil
}

func (f configForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.help.SetWidth(msg.Width)
		f.input.SetWidth(max(10, msg.Width-configLabelWidth-4))
		return f, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			f.quit = true
			return f, tea.Quit
		}
		if f.picking {
			return f.updatePicker(msg), nil
		}
		return f.updateRows(msg)
	}
	if f.isText(f.focus) {
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return f, cmd
	}
	return f, nil
}

func (f configForm) updateRows(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, f.keys.Cancel):
		f.quit = true
		return f, tea.Quit
	case key.Matches(msg, f.keys.Up):
		return f.moveTo((f.focus + rowCount - 1) % rowCount)
	case key.Matches(msg, f.keys.Down):
		return f.moveTo((f.focus + 1) % rowCount)
	case msg.String() == "enter":
		switch {
		case f.focus == rowSave:
			f.problem = ""
			if err := f.app.ValidateConfigChoices(f.choices); err != nil {
				f.problem = err.Error()
				return f, nil
			}
			f.saved = true
			return f, tea.Quit
		case f.isText(f.focus):
			return f.moveTo(f.focus + 1)
		}
		f.picking = true
		f.picked = max(0, indexOf(f.values(f.focus), f.value(f.focus)))
		return f, nil
	}
	if f.isText(f.focus) {
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		f.setValue(f.focus, strings.TrimSpace(f.input.Value()))
		return f, cmd
	}
	return f, nil
}

// moveTo gives row the focus, and the text input if it is typed into.
func (f configForm) moveTo(row int) (tea.Model, tea.Cmd) {
	f.focus = row
	f.input.Blur()
	if !f.isText(row) {
		return f, nil
	}
	f.input.SetValue(f.value(row))
	f.input.CursorEnd()
	return f, f.input.Focus()
}

func (f configForm) updatePicker(msg tea.KeyPressMsg) configForm {
	values := f.values(f.focus)
	switch msg.String() {
	case "esc":
		f.picking = false
	case "enter":
		f.picking = false
		f.setValue(f.focus, values[f.picked])
	case "up":
		f.picked = max(0, f.picked-1)
	case "down":
		f.picked = min(len(values)-1, f.picked+1)
	case "pgup":
		f.picked = max(0, f.picked-pickerHeight)
	case "pgdown":
		f.picked = min(len(values)-1, f.picked+pickerHeight)
	case "home":
		f.picked = 0
	case "end":
		f.picked = len(values) - 1
	default:
		// A letter or digit moves to the next value that starts with it.
		if msg.Text == "" {
			break
		}
		for i := 1; i <= len(values); i++ {
			next := (f.picked + i) % len(values)
			if strings.HasPrefix(values[next], strings.ToLower(msg.Text)) {
				f.picked = next
				break
			}
		}
	}
	return f
}

func (f configForm) isText(row int) bool {
	return row == rowName || row == rowDocroot
}

func (f configForm) value(row int) string {
	switch row {
	case rowName:
		return f.choices.Name
	case rowDocroot:
		return f.choices.Docroot
	case rowType:
		return f.choices.Type
	case rowPHP:
		return f.choices.PHPVersion
	case rowDBType:
		return f.choices.Database.Type
	case rowDBVersion:
		return f.choices.Database.Version
	case rowWebserver:
		return f.choices.WebserverType
	}
	return ""
}

// values lists what the setting in row can be, if it is chosen from a list.
func (f configForm) values(row int) []string {
	switch row {
	case rowType:
		return ddevapp.GetValidAppTypes()
	case rowPHP:
		return nodeps.GetValidPHPVersions()
	case rowDBType:
		return ddevapp.DatabaseTypes()
	case rowDBVersion:
		versions, _ := ddevapp.DatabaseVersions(f.choices.Database.Type)
		return versions
	case rowWebserver:
		types := nodeps.GetValidWebserverTypes()
		sort.Strings(types)
		return types
	}
	return nil
}

// setValue sets the setting in row, and the settings that follow from it:
// a project type brings its own PHP version, database, and webserver type
// unless the user has set them, and a database type its default version.
func (f *configForm) setValue(row int, value string) {
	f.problem = ""
	if row != rowDBType || value != f.choices.Database.Type {
		f.changed[row] = true
	}
	switch row {
	case rowName:
		f.choices.Name = value
	case rowDocroot:
		f.choices.Docroot = strings.Trim(value, `"'`)
	case rowType:
		f.choices.Type = value
		defaults := f.app.ConfigTypeDefaults(f.choices)
		if !f.changed[rowPHP] {
			f.choices.PHPVersion = defaults.PHPVersion
		}
		if !f.changed[rowDBType] && !f.changed[rowDBVersion] {
			f.choices.Database = defaults.Database
		}
		if !f.changed[rowWebserver] {
			f.choices.WebserverType = defaults.WebserverType
		}
	case rowPHP:
		f.choices.PHPVersion = value
	case rowDBType:
		if value != f.choices.Database.Type {
			_, version := ddevapp.DatabaseVersions(value)
			f.choices.Database = ddevapp.DatabaseDesc{Type: value, Version: version}
		}
	case rowDBVersion:
		f.choices.Database.Version = value
	case rowWebserver:
		f.choices.WebserverType = value
	}
}

// configLabelWidth is the width of the column of setting names.
const configLabelWidth = 18

func (f configForm) View() tea.View {
	if f.quit {
		return tea.NewView("")
	}
	var b strings.Builder
	title := "DDEV suggests this configuration. Change any setting, then save."
	if f.app.ConfigExists() {
		title = "This is the project's configuration. Change any setting, then save."
	}
	if f.saved {
		title = "Configuration:"
	}
	b.WriteString(f.styles.Title.Render(title) + "\n\n")
	for row := range rowSave {
		b.WriteString(f.rowView(row) + "\n")
		if f.picking && f.focus == row {
			b.WriteString(f.pickerView())
		}
	}
	if f.saved {
		return tea.NewView(b.String())
	}

	save := "  Save this configuration"
	if f.focus == rowSave {
		save = f.styles.Selected.Render("❯ Save this configuration")
	}
	b.WriteString("\n" + save + "\n\n")
	if f.problem != "" {
		b.WriteString(f.styles.Error.Render(f.problem) + "\n\n")
	}
	b.WriteString(f.help.ShortHelpView(f.helpKeys()) + "\n")
	return tea.NewView(b.String())
}

func (f configForm) rowView(row int) string {
	focused := f.focus == row && !f.saved
	label := fmt.Sprintf("%-*s", configLabelWidth, configRowLabels[row])
	value := f.value(row)
	switch {
	case focused && f.isText(row):
		value = f.input.View()
	case row == rowDocroot && value == "":
		value = f.styles.Muted.Render("(project root)")
	case row == rowDBType:
		value = ddevapp.DatabaseDisplayName(value)
	}
	if !focused {
		return "  " + f.styles.Label.Render(label) + value
	}
	return f.styles.Selected.Render("❯ "+label) + value
}

// pickerView lists the values of the focused row around the highlighted one.
func (f configForm) pickerView() string {
	values := f.values(f.focus)
	start := min(max(0, f.picked-pickerHeight/2), max(0, len(values)-pickerHeight))
	end := min(len(values), start+pickerHeight)
	indent := strings.Repeat(" ", configLabelWidth)
	var b strings.Builder
	if start > 0 {
		b.WriteString(indent + f.styles.Muted.Render("  ↑ more") + "\n")
	}
	for i := start; i < end; i++ {
		text := values[i]
		if f.focus == rowDBType {
			text = ddevapp.DatabaseDisplayName(text)
		}
		note := ""
		if values[i] == f.suggestedValue(f.focus) {
			note = f.styles.Muted.Render(" (suggested)")
		}
		if i == f.picked {
			b.WriteString(indent + f.styles.Selected.Render("❯ "+text) + note + "\n")
		} else {
			b.WriteString(indent + "  " + text + note + "\n")
		}
	}
	if end < len(values) {
		b.WriteString(indent + f.styles.Muted.Render("  ↓ more") + "\n")
	}
	return b.String()
}

// suggestedValue is the value DDEV would choose for row, given the settings
// above it: the detected project type, or what the chosen one calls for.
func (f configForm) suggestedValue(row int) string {
	if row == rowType {
		return f.suggested.Type
	}
	defaults := f.app.ConfigTypeDefaults(f.choices)
	if row == rowDBVersion && f.choices.Database.Type != defaults.Database.Type {
		_, version := ddevapp.DatabaseVersions(f.choices.Database.Type)
		return version
	}
	suggested := configForm{choices: defaults}
	return suggested.value(row)
}

func (f configForm) helpKeys() []key.Binding {
	switch {
	case f.picking:
		return []key.Binding{f.keys.Up, f.keys.Choose, f.keys.Close}
	case f.focus == rowSave:
		return []key.Binding{f.keys.Up, f.keys.Save, f.keys.Cancel}
	case f.isText(f.focus):
		return []key.Binding{f.keys.Up, f.keys.Edit, f.keys.Cancel}
	}
	return []key.Binding{f.keys.Up, f.keys.Choose, f.keys.Cancel}
}

func indexOf(values []string, value string) int {
	for i, v := range values {
		if v == value {
			return i
		}
	}
	return -1
}
