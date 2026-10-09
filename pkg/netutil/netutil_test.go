package netutil_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/ddev/ddev/pkg/netutil"
	"github.com/stretchr/testify/require"
)

// listenFreePorts finds n consecutive free ports on 127.0.0.1 and returns the first.
func listenFreePorts(t *testing.T, n int) int {
	for start := 23000; start < 29000; start += n {
		ok := true
		for p := start; p < start+n; p++ {
			if !netutil.IsHostPortFree("127.0.0.1", p) {
				ok = false
				break
			}
		}
		if ok {
			return start
		}
	}
	t.Fatalf("no %d consecutive free ports found", n)
	return 0
}

func TestIsHostPortFree(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	port := l.Addr().(*net.TCPAddr).Port

	require.False(t, netutil.IsHostPortFree("127.0.0.1", port))
	require.NoError(t, l.Close())
	require.True(t, netutil.IsHostPortFree("127.0.0.1", port))
}

func TestAllocateHostPort(t *testing.T) {
	start := listenFreePorts(t, 3)

	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(start))
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	skipLast := func(port string) bool { return port == strconv.Itoa(start+2) }
	for range 20 {
		require.Equal(t, strconv.Itoa(start+1), netutil.AllocateHostPort("127.0.0.1", start, start+2, skipLast))
	}

	skipAll := func(port string) bool { return port != strconv.Itoa(start) }
	require.Equal(t, "", netutil.AllocateHostPort("127.0.0.1", start, start+2, skipAll), "the only unskipped port is in use")

	// 192.0.2.1 is reserved for documentation, so it is never a local address.
	require.Equal(t, "", netutil.AllocateHostPort("192.0.2.1", start, start+2, nil))
}
