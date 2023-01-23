package types

type Event struct {
	EventType string `json:"event_type"`
	EventOptions
	EventProperties map[string]interface{}                `json:"event_properties,omitempty"`
	UserProperties  map[IdentityOp]map[string]interface{} `json:"user_properties,omitempty"`
	Groups          map[string][]string                   `json:"groups,omitempty"`
	GroupProperties map[IdentityOp]map[string]interface{} `json:"group_properties,omitempty"`

	// UserID is a user identifier. The value is ignored if EventOptions.UserID is set.
	UserID string `json:"-"`

	// DeviceID is a device-specific identifier. The value is ignored if EventOptions.DeviceID is set.
	DeviceID string `json:"-"`
}

func (e Event) Clone() Event {
	optionsClone := e.EventOptions.Clone()

	return Event{
		EventType:       e.EventType,
		EventOptions:    *optionsClone,
		EventProperties: cloneProperties(e.EventProperties),
		UserProperties:  cloneIdentityProperties(e.UserProperties),
		Groups:          cloneGroups(e.Groups),
		GroupProperties: cloneIdentityProperties(e.GroupProperties),
		UserID:          e.UserID,
		DeviceID:        e.DeviceID,
	}
}

func cloneProperties(properties map[string]interface{}) map[string]interface{} {
	if properties == nil {
		return nil
	}

	clone := make(map[string]interface{}, len(properties))

	for k, v := range properties {
		clone[k] = cloneUnknown(v)
	}

	return clone
}

func cloneIdentityProperties(properties map[IdentityOp]map[string]interface{}) map[IdentityOp]map[string]interface{} {
	if properties == nil {
		return nil
	}

	clone := make(map[IdentityOp]map[string]interface{})

	for operation, p := range properties {
		clone[operation] = cloneProperties(p)
	}

	return clone
}

func cloneGroups(properties map[string][]string) map[string][]string {
	if properties == nil {
		return nil
	}

	clone := make(map[string][]string, len(properties))
	for k, v := range properties {
		clone[k] = make([]string, len(v))
		copy(clone[k], v)
	}

	return clone
}

func cloneIntegers(values []int) []int {
	if values == nil {
		return nil
	}

	clone := make([]int, len(values))
	copy(clone, values)

	return clone
}

func cloneFloats(values []float64) []float64 {
	if values == nil {
		return nil
	}

	clone := make([]float64, len(values))
	copy(clone, values)

	return clone
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}

	clone := make([]string, len(values))
	copy(clone, values)

	return clone
}

func cloneBooleans(values []bool) []bool {
	if values == nil {
		return nil
	}

	clone := make([]bool, len(values))
	copy(clone, values)

	return clone
}

func cloneUnknowns(values []interface{}) []interface{} {
	if values == nil {
		return nil
	}

	clone := make([]interface{}, len(values))
	for i, value := range values {
		clone[i] = cloneUnknown(value)
	}

	return clone
}

func cloneUnknown(value interface{}) interface{} {
	switch value := value.(type) {
	case []int:
		return cloneIntegers(value)
	case []float64:
		return cloneFloats(value)
	case []string:
		return cloneStrings(value)
	case []bool:
		return cloneBooleans(value)
	case []interface{}:
		return cloneUnknowns(value)
	case map[string]interface{}:
		clone := make(map[string]interface{}, len(value))
		for k, v := range value {
			clone[k] = cloneUnknown(v)
		}

		return clone
	default:
		return value
	}
}
