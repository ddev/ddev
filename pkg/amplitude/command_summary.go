package amplitude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/third_party/ampli"
	"github.com/gofrs/flock"
)

// CommandSummaryEventType is the event that reports one command's runs in one week.
const CommandSummaryEventType = "Command Summary"

// commandsFile is the name of the file in ~/.ddev that counts the week's commands.
const commandsFile = ".amplitude.commands"

// commandCounts holds the number of runs of each command path in the week
// that starts on Week, a Monday.
type commandCounts struct {
	Week   string         `json:"week"`
	Counts map[string]int `json:"counts"`
}

// commandSummaryEvent implements ampli.Event. It is not part of the
// generated tracking plan code, which ampli pull would overwrite.
type commandSummaryEvent struct {
	properties map[string]any
}

func (e commandSummaryEvent) ToAmplitudeEvent() amplitude.Event {
	return amplitude.Event{
		EventType:       CommandSummaryEventType,
		EventProperties: e.properties,
	}
}

// countCommand adds one run of commandPath to this week's counts in file.
// When the file holds counts from an earlier week, it starts a new week and
// returns those counts, so the caller can send them.
func countCommand(file string, week string, commandPath string) *commandCounts {
	unlock, ok := lockFile(file + ".lock")
	if !ok {
		return nil
	}
	defer unlock()

	state := commandCounts{}
	if content, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(content, &state)
	}

	var previous *commandCounts
	if state.Week != week || state.Counts == nil {
		if state.Week != "" && len(state.Counts) > 0 {
			previous = &commandCounts{Week: state.Week, Counts: state.Counts}
		}
		state = commandCounts{Week: week, Counts: map[string]int{}}
	}

	state.Counts[commandPath]++

	if content, err := json.Marshal(state); err == nil {
		tmpFile := file + ".tmp"
		if os.WriteFile(tmpFile, content, 0644) == nil {
			_ = os.Rename(tmpFile, file)
		}
	}

	return previous
}

// commandSummaryEvents returns one event per command path in counts, in a
// stable order.
func commandSummaryEvents(counts commandCounts) []ampli.Event {
	paths := make([]string, 0, len(counts.Counts))
	for path := range counts.Counts {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	events := make([]ampli.Event, 0, len(paths))
	for _, path := range paths {
		name := path
		if fields := strings.Fields(path); len(fields) > 0 {
			name = fields[len(fields)-1]
		}
		events = append(events, commandSummaryEvent{properties: map[string]any{
			`Command Name`: name,
			`Command Path`: path,
			`Count`:        counts.Counts[path],
			`Week`:         counts.Week,
		}})
	}

	return events
}

// sendCommandSummaries queues the summary events for an earlier week.
func sendCommandSummaries(counts commandCounts) {
	// Thursday noon falls in the same week whether weeks start on Sunday or
	// Monday, and stays on that day in UTC for most time zones.
	eventTime := time.Now()
	if monday, err := time.ParseInLocation(time.DateOnly, counts.Week, time.Local); err == nil {
		eventTime = monday.AddDate(0, 0, 3).Add(12 * time.Hour)
	}

	for _, event := range commandSummaryEvents(counts) {
		options := GetEventOptions()
		options.SetTime(eventTime)
		ampli.Instance.Track("", event, options)
	}
}

// weekStart returns the date of the Monday that starts t's week, in t's
// location.
func weekStart(t time.Time) string {
	daysSinceMonday := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location()).Format(time.DateOnly)
}

// lockFile holds an exclusive lock on path. It gives up after a second,
// because losing one count is better than delaying a command.
func lockFile(path string) (unlock func(), ok bool) {
	// A lock file created by root would lock out the user's own ddev.
	if os.Geteuid() == 0 {
		return func() {}, true
	}

	fl := flock.New(path)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if locked, err := fl.TryLockContext(ctx, 20*time.Millisecond); err != nil || !locked {
		_ = fl.Close()
		return nil, false
	}

	return func() {
		_ = fl.Unlock()
		_ = fl.Close()
	}, true
}

// getCommandsFileName returns the path of the command counts file.
func getCommandsFileName() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), commandsFile)
}
