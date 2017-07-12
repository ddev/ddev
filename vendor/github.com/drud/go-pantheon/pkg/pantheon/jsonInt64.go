package pantheon

import (
	"bytes"
	"encoding/json"
)

// jsonFloat is an int64 which unmarshals from JSON
// as either unquoted or quoted (with any amount
// of internal leading/trailing whitespace).
// it is based off of https://play.golang.org/p/KNPxDL1yqL
type jsonInt64 int64

func (f jsonInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(f))
}

func (f *jsonInt64) UnmarshalJSON(data []byte) error {
	var v int64

	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		// Remove single set of matching quotes
		data = data[1 : len(data)-1]
	}
	// And remove any whitespace:
	data = bytes.TrimSpace(data)

	err := json.Unmarshal(data, &v)
	*f = jsonInt64(v)
	return err
}
