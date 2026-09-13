package jsonformat

import "encoding/json"

// String returns stable, indented JSON while preserving non-JSON responses as JSON strings.
func String(raw []byte) string {
	if len(raw) == 0 {
		return "null"
	}

	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, _ := json.Marshal(string(raw))
		return string(formatted)
	}

	formatted, _ := json.MarshalIndent(parsed, "", "  ")
	return string(formatted)
}
