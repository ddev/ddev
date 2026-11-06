package amplitude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCountCommand(t *testing.T) {
	file := filepath.Join(t.TempDir(), commandsFile)
	week1 := "2026-11-02"
	week2 := "2026-11-09"

	require.Nil(t, countCommand(file, week1, "ddev exec"))
	require.Nil(t, countCommand(file, week1, "ddev exec"))
	require.Nil(t, countCommand(file, week1, "ddev utility instrumentation flush"))

	previous := countCommand(file, week2, "ddev start")
	require.NotNil(t, previous)
	require.Equal(t, week1, previous.Week)
	require.Equal(t, map[string]int{"ddev exec": 2, "ddev utility instrumentation flush": 1}, previous.Counts)

	require.Nil(t, countCommand(file, week2, "ddev start"))

	previous = countCommand(file, "2026-11-30", "ddev exec")
	require.NotNil(t, previous)
	require.Equal(t, week2, previous.Week)
	require.Equal(t, map[string]int{"ddev start": 2}, previous.Counts)
}

func TestCountCommandCorruptFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), commandsFile)
	require.NoError(t, os.WriteFile(file, []byte("not json"), 0644))

	require.Nil(t, countCommand(file, "2026-11-02", "ddev exec"))
	previous := countCommand(file, "2026-11-09", "ddev exec")
	require.NotNil(t, previous)
	require.Equal(t, map[string]int{"ddev exec": 1}, previous.Counts)
}

func TestWeekStart(t *testing.T) {
	loc := time.FixedZone("test", -7*3600)
	for _, tc := range []struct{ day, monday string }{
		{"2026-11-02", "2026-11-02"},
		{"2026-11-05", "2026-11-02"},
		{"2026-11-08", "2026-11-02"},
		{"2026-11-09", "2026-11-09"},
		{"2027-01-01", "2026-12-28"},
		{"2028-03-01", "2028-02-28"},
	} {
		day, err := time.ParseInLocation(time.DateOnly, tc.day, loc)
		require.NoError(t, err)
		require.Equal(t, tc.monday, weekStart(day.Add(23*time.Hour)), tc.day)
	}
}

func TestCommandSummaryEvents(t *testing.T) {
	events := commandSummaryEvents(commandCounts{
		Week:   "2026-11-02",
		Counts: map[string]int{"ddev utility instrumentation flush": 1, "ddev exec": 400},
	})
	require.Len(t, events, 2)

	first := events[0].ToAmplitudeEvent()
	require.Equal(t, CommandSummaryEventType, first.EventType)
	require.Equal(t, map[string]any{
		"Command Name": "exec",
		"Command Path": "ddev exec",
		"Count":        400,
		"Week":         "2026-11-02",
	}, first.EventProperties)

	second := events[1].ToAmplitudeEvent()
	require.Equal(t, "flush", second.EventProperties["Command Name"])
	require.Equal(t, 1, second.EventProperties["Count"])
}
