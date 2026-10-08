package util_test

import (
	"testing"

	"github.com/ddev/ddev/pkg/util"
	"github.com/stretchr/testify/require"
)

// TestHostEnv tests that only variables set on the host are returned, empty
// ones included, in the order they were asked for.
func TestHostEnv(t *testing.T) {
	t.Setenv("DDEV_TEST_HOSTENV_SET", "one")
	t.Setenv("DDEV_TEST_HOSTENV_OTHER", "two=three")
	t.Setenv("DDEV_TEST_HOSTENV_EMPTY", "")

	require.Equal(t,
		[]string{"DDEV_TEST_HOSTENV_OTHER=two=three", "DDEV_TEST_HOSTENV_EMPTY=", "DDEV_TEST_HOSTENV_SET=one"},
		util.HostEnv("DDEV_TEST_HOSTENV_OTHER", "DDEV_TEST_HOSTENV_EMPTY", "DDEV_TEST_HOSTENV_UNSET", "DDEV_TEST_HOSTENV_SET"),
	)
	require.Nil(t, util.HostEnv("DDEV_TEST_HOSTENV_UNSET"))
	require.Nil(t, util.HostEnv())
}

// TestMergeEnv tests that the last entry for a name wins, in place, whether
// it is NAME=value or a bare NAME.
func TestMergeEnv(t *testing.T) {
	testCases := []struct {
		lists    [][]string
		expected []string
	}{
		{nil, nil},
		{[][]string{{"A=1"}, nil, {}}, []string{"A=1"}},
		{[][]string{{"A=1", "B=2"}, {"A=3"}}, []string{"B=2", "A=3"}},
		{[][]string{{"ONE=one", "ONE=two", "ONE=three", "TWO=two", "TWO=three", "TWO=four"}}, []string{"ONE=three", "TWO=four"}},
		{[][]string{{"BARE", "KEY=value"}}, []string{"BARE", "KEY=value"}},
		{[][]string{{"TERM=xterm-256color", "COLORTERM=truecolor"}, {"COLORTERM="}}, []string{"TERM=xterm-256color", "COLORTERM="}},
		{[][]string{{"BARE"}, {"BARE=explicit"}}, []string{"BARE=explicit"}},
		{[][]string{{"BARE=explicit"}, {"BARE"}}, []string{"BARE"}},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.expected, util.MergeEnv(tc.lists...), "lists=%v", tc.lists)
	}
}
