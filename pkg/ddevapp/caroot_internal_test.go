package ddevapp

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCARootArchive(t *testing.T) {
	caRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(caRoot, "rootCA.pem"), []byte("cert"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(caRoot, "rootCA-key.pem"), []byte("key"), 0400))

	archive, err := caRootArchive(caRoot, 1234, 5678)
	require.NoError(t, err)

	type entry struct {
		name    string
		mode    int64
		content string
	}
	var got []entry
	tr := tar.NewReader(bytes.NewReader(archive))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Equal(t, 1234, h.Uid, h.Name)
		require.Equal(t, 5678, h.Gid, h.Name)
		content, err := io.ReadAll(tr)
		require.NoError(t, err)
		got = append(got, entry{h.Name, h.Mode, string(content)})
	}
	require.Equal(t, []entry{
		{"usr/local/lib/ddev/mkcert/", 0755, ""},
		{"usr/local/lib/ddev/mkcert/rootCA-key.pem", 0600, "key"},
		{"usr/local/lib/ddev/mkcert/rootCA.pem", 0644, "cert"},
	}, got)
}
