package amplitude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/amplitude/analytics-go/amplitude"
	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/third_party/ampli"
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
// In a later week it returns the earlier week's counts for the caller to send,
// or, if canSend() is false, keeps them and skips this run.
func countCommand(file string, week string, commandPath string, canSend func() bool) *commandCounts {
	unlock, ok := storages.LockFile(file + ".lock")
	if !ok {
		return nil
	}
	defer unlock()

	state := commandCounts{}
	if content, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(content, &state)
	}

	var previous *commandCounts
	// An earlier week, from a clock set back or another time zone, counts
	// into the stored one.
	if _, err := time.Parse(time.DateOnly, state.Week); err != nil || state.Counts == nil {
		state = commandCounts{Week: week, Counts: map[string]int{}}
	} else if week > state.Week {
		if len(state.Counts) > 0 {
			if !canSend() {
				return nil
			}
			previous = &commandCounts{Week: state.Week, Counts: state.Counts}
		}
		state = commandCounts{Week: week, Counts: map[string]int{}}
	}

	state.Counts[commandPath]++

	content, err := json.Marshal(state)
	if err != nil || storages.WriteFileAtomic(file, content) != nil {
		return nil
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

// getCommandsFileName returns the path of the command counts file.
func getCommandsFileName() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), commandsFile)
}
