package output

import (
	"encoding/json"
	"fmt"
)

// JSON marshals v as indented JSON and returns it as a string.
func JSON(v any) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("output: marshal json: %w", err)
	}
	return string(data), nil
}
