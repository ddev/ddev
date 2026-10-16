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

func TestEnvFlagEntries(t *testing.T) {
	testCases := []struct {
		flagValues []string
		expected   []string
	}{
		{nil, []string{}},
		{[]string{"", `""`, `''`, "  "}, []string{}},
		{[]string{"A=1"}, []string{"A=1"}},
		{[]string{"A=1,B=2"}, []string{"A=1", "B=2"}},
		{[]string{"A=1, B=2"}, []string{"A=1", "B=2"}},
		{[]string{"  A=1 "}, []string{"A=1"}},
		{[]string{"A=1", "B=2,C=3"}, []string{"A=1", "B=2", "C=3"}},
		{[]string{"TOKEN=a,b,ENV=main"}, []string{"TOKEN=a,b", "ENV=main"}},
		{[]string{"LIST=1,2,3"}, []string{"LIST=1,2,3"}},
		{[]string{"MSG=red, green"}, []string{"MSG=red, green"}},
		{[]string{"DSN=a=b,c=d"}, []string{"DSN=a=b", "c=d"}},
		{[]string{"QUERY=x?a=1&b=2,ENV=main"}, []string{"QUERY=x?a=1&b=2", "ENV=main"}},
		{[]string{"EMPTY=,NEXT=1"}, []string{"EMPTY=", "NEXT=1"}},
		{[]string{"TRAILING=a,"}, []string{"TRAILING=a,"}},
		{[]string{"FOO,BAR"}, []string{"FOO", "BAR"}},
		{[]string{"foo,bar"}, []string{"foo", "bar"}},
		{[]string{"FOO,bar=1"}, []string{"FOO", "bar=1"}},
		{[]string{"BAR,BAZ,FOO=a"}, []string{"BAR", "BAZ", "FOO=a"}},
		{[]string{"METHODS=GET,POST"}, []string{"METHODS=GET,POST"}},
		{[]string{"FOO=a", "BAR"}, []string{"FOO=a", "BAR"}},
		{[]string{"FOO=a,b"}, []string{"FOO=a,b"}},
		{[]string{"MODULES=Foo,Bar"}, []string{"MODULES=Foo,Bar"}},
		{[]string{"FOO=a, BAR"}, []string{"FOO=a, BAR"}},
		{[]string{"FOO=a,1b"}, []string{"FOO=a,1b"}},
		{[]string{"FOO=a,,b"}, []string{"FOO=a,,b"}},
		{[]string{"FOO=a", ",b"}, []string{"FOO=a", "b"}},
		{[]string{"FOO,,BAR,"}, []string{"FOO", "BAR"}},
		{[]string{"bad-name=x"}, []string{"bad-name=x"}},
	}
	for _, tc := range testCases {
		require.Equal(t, tc.expected, util.EnvFlagEntries(tc.flagValues), "flagValues=%q", tc.flagValues)
	}
}
