package utils

// FormatPlural is a simple wrapper which returns different strings based on the count value.
func FormatPlural(count int, single string, plural string) string {
	if count == 1 {
		return single
	}
	return plural
}

func empty(p string) bool {
	return p == ""
}

func byteify(s string) []byte {
	return []byte(s)
}
