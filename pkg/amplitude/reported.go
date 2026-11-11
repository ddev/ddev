package amplitude

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ddev/ddev/pkg/amplitude/storages"
	"github.com/ddev/ddev/pkg/globalconfig"
	"github.com/ddev/ddev/pkg/versionconstants"
	"github.com/ddev/ddev/third_party/ampli"
)

// reportedFile is the name of the file in ~/.ddev recording what was sent.
const reportedFile = ".amplitude.reported"

// unchangedResendDays is how many days an unchanged event waits before it is
// sent again. A queued event can still be dropped, for example when a send
// fails with an error that is not retried, so it can't wait forever.
const unchangedResendDays = 7

// reportedState holds the hashes of the events queued on Date, per key, and
// the last hash per key, with the date it was queued, for events that are
// sent only when they change.
type reportedState struct {
	Date       string              `json:"date"`
	Hashes     map[string][]string `json:"hashes"`
	Latest     map[string]string   `json:"latest,omitempty"`
	LatestDate map[string]string   `json:"latest_date,omitempty"`
}

// AlreadyReportedToday returns true if an event with the same key and the
// same properties was already queued today, or if DDEV is offline. Otherwise
// it records the event and returns false, so the caller should send it.
func AlreadyReportedToday(key string, event ampli.Event) bool {
	return alreadyReported(getReportedFileName(), time.Now().Format(time.DateOnly), key, event, false, globalconfig.IsInternetActive)
}

// AlreadyReportedUnchanged returns true if the last event queued with the
// same key had the same properties and was queued less than
// unchangedResendDays ago, or if DDEV is offline. Otherwise it records the
// event and returns false, so the caller should send it.
func AlreadyReportedUnchanged(key string, event ampli.Event) bool {
	return alreadyReported(getReportedFileName(), time.Now().Format(time.DateOnly), key, event, true, globalconfig.IsInternetActive)
}

func alreadyReported(file string, today string, key string, event ampli.Event, onlyOnChange bool, canSend func() bool) bool {
	hash := hashEvent(event)

	// Without the lock, sending the event risks only a duplicate.
	unlock, ok := storages.LockFile(file + ".lock")
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
	if state.LatestDate == nil {
		state.LatestDate = map[string]string{}
	}

	if onlyOnChange {
		if state.Latest[key] == hash && daysBetween(state.LatestDate[key], today) < unchangedResendDays {
			return true
		}
	} else if slices.Contains(state.Hashes[key], hash) {
		return true
	}

	// Offline, main() exits without waiting for the queue, so the event would
	// be lost but recorded as sent.
	if !canSend() {
		return true
	}

	if onlyOnChange {
		state.Latest[key] = hash
		state.LatestDate[key] = today
	} else {
		state.Hashes[key] = append(state.Hashes[key], hash)
	}

	if content, err := json.Marshal(state); err == nil {
		_ = storages.WriteFileAtomic(file, content)
	}

	return false
}

// daysBetween returns the number of days from one date to another, or a
// large number if either date can't be parsed.
func daysBetween(from string, to string) int {
	fromDate, errFrom := time.Parse(time.DateOnly, from)
	toDate, errTo := time.Parse(time.DateOnly, to)
	if errFrom != nil || errTo != nil {
		return math.MaxInt
	}
	return int(toDate.Sub(fromDate).Hours() / 24)
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
