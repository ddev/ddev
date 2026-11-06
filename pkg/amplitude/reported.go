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

// reportedFile is the name of the file in ~/.ddev recording what was sent today.
const reportedFile = ".amplitude.reported"

// reportedState holds hashes of the events queued on Date, per key.
type reportedState struct {
	Date   string              `json:"date"`
	Hashes map[string][]string `json:"hashes"`
}

// AlreadyReportedToday returns true if an event with the same key and the
// same properties was already queued today. Otherwise it records the event
// and returns false, so the caller should send it.
func AlreadyReportedToday(key string, event ampli.Event) bool {
	return alreadyReported(getReportedFileName(), time.Now().Format(time.DateOnly), key, event)
}

func alreadyReported(file string, today string, key string, event ampli.Event) bool {
	hash := hashEvent(event)

	state := reportedState{}
	if content, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(content, &state)
	}

	// A new day discards everything recorded on earlier days.
	if state.Date != today || state.Hashes == nil {
		state = reportedState{Date: today, Hashes: map[string][]string{}}
	}

	if slices.Contains(state.Hashes[key], hash) {
		return true
	}

	state.Hashes[key] = append(state.Hashes[key], hash)

	// Concurrent ddev processes may overwrite each other's entries, which
	// only costs a duplicate event later in the day.
	if content, err := json.Marshal(state); err == nil {
		tmpFile := file + ".tmp"
		if os.WriteFile(tmpFile, content, 0644) == nil {
			_ = os.Rename(tmpFile, file)
		}
	}

	return false
}

// hashEvent hashes the event's properties and the DDEV version, so that an
// upgrade or a configuration change is reported again the same day.
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
