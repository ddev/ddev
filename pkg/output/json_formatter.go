package output

import (
	"encoding/json"
	"time"
)

// JSONFormatter renders an Entry as a single line of JSON, matching the
// shape ddev has always emitted in --json-output mode: "time", "level" and
// "msg" as top-level keys, with Data fields merged in as siblings rather than
// nested. Consumers such as `ddev describe -j` read a WithField("raw", …)
// value as a top-level "raw" key on that same line.
type JSONFormatter struct{}

func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		if err, ok := v.(error); ok {
			data[k] = err.Error()
		} else {
			data[k] = v
		}
	}
	prefixFieldClashes(data)
	data["time"] = entry.Time.Format(time.RFC3339)
	data["level"] = entry.Level.String()
	data["msg"] = entry.Message

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
