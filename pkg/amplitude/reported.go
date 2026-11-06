package amplitude

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/third_party/ampli"
)

// reportedFile is the name of the file in ~/.ddev recording what was sent.
const reportedFile = ".amplitude.reported"

// reportedState holds the hashes of the events queued on Date, per key, and
// the last hash per key for events that are sent only when they change.
type reportedState struct {
	Date   string              `json:"date"`
	Hashes map[string][]string `json:"hashes"`
	Latest map[string]string   `json:"latest,omitempty"`
}

// AlreadyReportedToday returns true if an event with the same key and the
// same properties was already queued today. Otherwise it records the event
// and returns false, so the caller should send it.
func AlreadyReportedToday(key string, event ampli.Event) bool {
	return alreadyReported(getReportedFileName(), time.Now().Format(time.DateOnly), key, event, false)
}

// AlreadyReportedUnchanged returns true if the last event queued with the
// same key, on any day, had the same properties. Otherwise it records the
// event and returns false, so the caller should send it.
func AlreadyReportedUnchanged(key string, event ampli.Event) bool {
	return alreadyReported(getReportedFileName(), time.Now().Format(time.DateOnly), key, event, true)
}

func alreadyReported(file string, today string, key string, event ampli.Event, onlyOnChange bool) bool {
	hash := hashEvent(event)

	// Without the lock, sending the event risks only a duplicate.
	unlock, ok := lockFile(file + ".lock")
	if !ok {
		return false
	}
	defer unlock()

	state := reportedState{}
	if content, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(content, &state)
	}

	// A new day discards the daily hashes but keeps the latest ones.
	if state.Date != today || state.Hashes == nil {
		state.Date = today
		state.Hashes = map[string][]string{}
	}
	if state.Latest == nil {
		state.Latest = map[string]string{}
	}

	if onlyOnChange {
		if state.Latest[key] == hash {
			return true
		}
		state.Latest[key] = hash
	} else {
		if slices.Contains(state.Hashes[key], hash) {
			return true
		}
		state.Hashes[key] = append(state.Hashes[key], hash)
	}

	if content, err := json.Marshal(state); err == nil {
		tmpFile := file + ".tmp"
		if os.WriteFile(tmpFile, content, 0644) == nil {
			_ = os.Rename(tmpFile, file)
		}
	}

	return false
}

// hashEvent hashes the event's properties and the DDEV version, so that an
// upgrade or a configuration change is reported again.
func hashEvent(event ampli.Event) string {
	amplitudeEvent := event.ToAmplitudeEvent()
	content, _ := json.Marshal([]any{
		versionconstants.DdevVersion,
		amplitudeEvent.EventType,
		amplitudeEvent.EventProperties,
		amplitudeEvent.UserProperties,
	})
	sum := sha256.Sum256(content)

	return hex.EncodeToString(sum[:])
}

// getReportedFileName returns the path of the reported-events file.
func getReportedFileName() string {
	return filepath.Join(globalconfig.GetGlobalDdevDir(), reportedFile)
}
