package amplitude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountCommand(t *testing.T) {
	file := filepath.Join(t.TempDir(), commandsFile)
	day1 := "2026-11-05"
	day2 := "2026-11-06"

	require.Nil(t, countCommand(file, day1, "ddev exec"))
	require.Nil(t, countCommand(file, day1, "ddev exec"))
	require.Nil(t, countCommand(file, day1, "ddev utility instrumentation flush"))

	previous := countCommand(file, day2, "ddev start")
	require.NotNil(t, previous)
	require.Equal(t, day1, previous.Date)
	require.Equal(t, map[string]int{"ddev exec": 2, "ddev utility instrumentation flush": 1}, previous.Counts)

	require.Nil(t, countCommand(file, day2, "ddev start"))

	previous = countCommand(file, "2026-11-09", "ddev exec")
	require.NotNil(t, previous)
	require.Equal(t, map[string]int{"ddev start": 2}, previous.Counts)
}

func TestCountCommandCorruptFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), commandsFile)
	require.NoError(t, os.WriteFile(file, []byte("not json"), 0644))

	require.Nil(t, countCommand(file, "2026-11-06", "ddev exec"))
	previous := countCommand(file, "2026-11-07", "ddev exec")
	require.NotNil(t, previous)
	require.Equal(t, map[string]int{"ddev exec": 1}, previous.Counts)
}

func TestCommandSummaryEvents(t *testing.T) {
	events := commandSummaryEvents(commandCounts{
		Date:   "2026-11-05",
		Counts: map[string]int{"ddev utility instrumentation flush": 1, "ddev exec": 400},
	})
	require.Len(t, events, 2)

	first := events[0].ToAmplitudeEvent()
	require.Equal(t, CommandSummaryEventType, first.EventType)
	require.Equal(t, map[string]any{
		"Command Name": "exec",
		"Command Path": "ddev exec",
		"Count":        400,
		"Date":         "2026-11-05",
	}, first.EventProperties)

	second := events[1].ToAmplitudeEvent()
	require.Equal(t, "flush", second.EventProperties["Command Name"])
	require.Equal(t, 1, second.EventProperties["Count"])
}
