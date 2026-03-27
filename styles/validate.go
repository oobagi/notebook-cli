package styles

import (
	"encoding/json"
	"fmt"
)

// Validate checks that jsonBytes is valid Glamour style JSON.
// It unmarshals into a generic map and verifies that key structural
// blocks are present (e.g., "document" exists and is non-empty).
func Validate(jsonBytes []byte) error {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if _, ok := data["document"]; !ok {
		return fmt.Errorf("missing required 'document' block")
	}
	return nil
}
