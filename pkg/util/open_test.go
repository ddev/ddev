package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEditorURL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	require.Equal(t, "vscode://file/home/user/my%20site?windowId=_blank", EditorURL("vscode", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=/home/user/my%20site", EditorURL("phpstorm", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=/home/user/a%26b%2Bc", EditorURL("phpstorm", "/home/user/a&b+c"))
	require.Equal(t, "vscode://file/C:/Users/me/site?windowId=_blank", EditorURL("vscode", "C:/Users/me/site"))
	require.Empty(t, EditorURL("unknown", "/home/user"))

	t.Setenv("WSL_DISTRO_NAME", "Ubuntu-26.04")
	require.Equal(t, "vscode://vscode-remote/wsl+Ubuntu-26.04/home/user/my%20site?windowId=_blank", EditorURL("vscode", "/home/user/my site"))
	require.Equal(t, "phpstorm://open?file=//wsl.localhost/Ubuntu-26.04/home/user/my%20site", EditorURL("phpstorm", "/home/user/my site"))
}
