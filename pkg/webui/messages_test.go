package webui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRemoteMessagesReload(t *testing.T) {
	loads := 0
	orig := newRemote
	newRemote = func() remote {
		loads++
		return remote{}
	}
	t.Cleanup(func() {
		newRemote = orig
		remoteCache.r, remoteCache.at = remote{}, time.Time{}
	})
	remoteCache.at = time.Time{}

	remoteMessages()
	require.Equal(t, 1, loads)
	remoteMessages()
	require.Equal(t, 1, loads, "within the hour the config is reused")
	remoteCache.at = time.Now().Add(-61 * time.Minute)
	remoteMessages()
	require.Equal(t, 2, loads, "after an hour it is read again")
}
