package tui

import (
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
		require.Contains(t, ansi.Strip(f.View().Content), "❯ php (suggested)")
		// Escape closes the list without leaving the form.
		f, quit := send(f, "home", "esc")
		require.False(t, quit)
		require.Equal(t, nodeps.AppTypePHP, f.choices.Type)

		f.picking, f.picked = true, indexOf(f.values(rowType), nodeps.AppTypeDrupal7)
		f, _ = send(f, "enter")
		require.Equal(t, nodeps.AppTypeDrupal7, f.choices.Type)
		require.Equal(t, nodeps.PHP82, f.choices.PHPVersion)

		// A PHP version the user chose outlasts another project type.
		f, _ = send(f, "down", "enter", "home", "enter")
		require.Equal(t, nodeps.GetValidPHPVersions()[0], f.choices.PHPVersion)
		f, _ = send(f, "up", "enter", "home", "enter")
		require.Equal(t, ddevapp.GetValidAppTypes()[0], f.choices.Type)
		require.Equal(t, nodeps.GetValidPHPVersions()[0], f.choices.PHPVersion)
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
