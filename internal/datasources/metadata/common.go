package metadata

import "encoding/json"

func formatJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	formatted, _ := json.MarshalIndent(parsed, "", "  ")
	return string(formatted), nil
}
