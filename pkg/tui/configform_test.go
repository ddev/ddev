package tui

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ddev/ddev/pkg/ddevapp"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/stretchr/testify/require"
)

func TestConfigForm(t *testing.T) {
	newForm := func(t *testing.T) configForm {
		dir := filepath.Join(t.TempDir(), "formproject")
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "web"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "web", "index.php"), nil, 0644))
		app, err := ddevapp.NewApp(dir, false)
		require.NoError(t, err)
		return newConfigForm(app)
	}
	send := func(f configForm, keys ...string) (configForm, bool) {
		quit := false
		for _, k := range keys {
			m, cmd := f.Update(press(k))
			f = m.(configForm)
			// Only these keys can quit; the text input's commands wait to blink.
			if cmd != nil && (k == "enter" || k == "esc") {
				_, isQuit := cmd().(tea.QuitMsg)
				quit = quit || isQuit
			}
		}
		return f, quit
	}

	t.Run("enter saves the detected configuration", func(t *testing.T) {
		f := newForm(t)
		require.Contains(t, ansi.Strip(f.View().Content), "❯ Save this configuration")
		f, quit := send(f, "enter")
		require.True(t, quit)
		require.True(t, f.saved)
		require.Equal(t, ddevapp.ConfigChoices{
			Name: "formproject", Docroot: "web", Type: nodeps.AppTypePHP, PHPVersion: nodeps.PHPDefault,
			Database: ddevapp.DatabaseDefault, WebserverType: nodeps.WebserverDefault,
		}, f.choices)
	})

	t.Run("escape cancels", func(t *testing.T) {
		f, quit := send(newForm(t), "esc")
		require.True(t, quit)
		require.False(t, f.saved)
	})

	t.Run("project type sets what follows from it", func(t *testing.T) {
		f, _ := send(newForm(t), "up", "up", "up", "up", "up", "enter")
		require.True(t, f.picking)
		view := ansi.Strip(f.View().Content)
		require.Contains(t, view, "❯ php (suggested)")
		require.NotContains(t, view, "drupal7")
		require.NotContains(t, view, "Drupal version")
		// Escape closes the list without leaving the form.
		f, quit := send(f, "home", "esc")
		require.False(t, quit)
		require.Equal(t, nodeps.AppTypePHP, f.choices.Type)

		f.picking, f.picked = true, indexOf(f.values(rowType), nodeps.AppTypeDrupal)
		f, _ = send(f, "enter")
		require.Equal(t, nodeps.AppTypeDrupalLatestStable, f.choices.Type)
		require.Contains(t, ansi.Strip(f.View().Content), "Drupal version    11")

		// The Drupal version brings its own PHP version.
		f, _ = send(f, "down", "enter", "7", "enter")
		require.Equal(t, nodeps.AppTypeDrupal7, f.choices.Type)
		require.Equal(t, nodeps.PHP82, f.choices.PHPVersion)

		// A PHP version the user chose outlasts another Drupal version.
		f, _ = send(f, "down", "enter", "home", "enter")
		require.Equal(t, nodeps.GetValidPHPVersions()[0], f.choices.PHPVersion)
		f, _ = send(f, "up", "enter", "end", "enter")
		require.Equal(t, nodeps.AppTypeDrupal12, f.choices.Type)
		require.Equal(t, nodeps.GetValidPHPVersions()[0], f.choices.PHPVersion)

		// Without Drupal the version row is gone again.
		f, _ = send(f, "up", "enter", "home", "enter")
		require.Equal(t, ddevapp.ProjectTypeFamilies()[0], f.choices.Type)
		require.NotContains(t, ansi.Strip(f.View().Content), "Drupal version")
		f, _ = send(f, "down")
		require.Equal(t, rowPHP, f.focus)
	})

	t.Run("bedrock brings its docroot", func(t *testing.T) {
		f := newForm(t)
		f.suggested.Docroot, f.choices.Docroot = "", ""
		f.focus, f.picking = rowType, true
		f.picked = indexOf(f.values(rowType), nodeps.AppTypeWordPress)
		f, _ = send(f, "enter")
		require.Equal(t, nodeps.AppTypeWordPress, f.choices.Type)
		require.Contains(t, ansi.Strip(f.View().Content), "WordPress variant standard")
		require.Empty(t, f.choices.Docroot)

		f, _ = send(f, "down", "enter", "b", "enter")
		require.Equal(t, nodeps.AppTypeWPBedrock, f.choices.Type)
		require.Equal(t, "web", f.choices.Docroot)

		// Back to standard WordPress, the docroot is the detected one again.
		f, _ = send(f, "enter", "s", "enter")
		require.Equal(t, nodeps.AppTypeWordPress, f.choices.Type)
		require.Empty(t, f.choices.Docroot)
	})

	t.Run("magento 1 is shown with OpenMage", func(t *testing.T) {
		f := newForm(t)
		f.focus, f.picking = rowType, true
		f.picked = indexOf(f.values(rowType), nodeps.AppTypeMagento)
		f, _ = send(f, "enter")
		require.Equal(t, nodeps.AppTypeMagento2, f.choices.Type)
		f, _ = send(f, "down", "enter")
		view := ansi.Strip(f.View().Content)
		require.Contains(t, view, "  1 / OpenMage")
		require.Contains(t, view, "❯ 2 (suggested)")
	})

	t.Run("terminal colors restyle the form", func(t *testing.T) {
		f := newForm(t)
		m, _ := f.Update(tea.BackgroundColorMsg{Color: color.Black})
		m, _ = m.(configForm).Update(tea.ForegroundColorMsg{Color: color.White})
		f = m.(configForm)
		require.Equal(t, NewStyles(color.White, color.Black), f.styles)
	})

	t.Run("database type sets its default version", func(t *testing.T) {
		f, _ := send(newForm(t), "up", "up", "up", "enter", "p", "enter")
		require.Equal(t, ddevapp.DatabaseDesc{Type: nodeps.Postgres, Version: nodeps.PostgresDefaultVersion}, f.choices.Database)
		f, _ = send(f, "down", "enter", "end", "enter")
		versions := nodeps.GetValidPostgresVersions()
		require.Equal(t, versions[len(versions)-1], f.choices.Database.Version)
	})

	t.Run("invalid name is reported instead of saved", func(t *testing.T) {
		f, quit := send(newForm(t), "down", "_", "x", "up", "enter")
		require.Equal(t, "formproject_x", f.choices.Name)
		require.False(t, quit)
		require.Contains(t, ansi.Strip(f.View().Content), "'formproject_x' is not a valid project name")

		f, quit = send(f, "down", "backspace", "backspace", "down", "/", "s", "up", "up", "enter")
		require.True(t, quit)
		require.Equal(t, "formproject", f.choices.Name)
		require.Equal(t, "web/s", f.choices.Docroot)
	})
}
