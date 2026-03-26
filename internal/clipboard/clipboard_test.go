package clipboard

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestOSC52Format(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple text", "hello world"},
		{"multiline", "line1\nline2\nline3"},
		{"empty", ""},
		{"special chars", "# Header\n\n- item one\n- item two"},
		{"unicode", "hello \u2713 world \u203A test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := copyOSC52(tt.input, &buf)
			if err != nil {
				t.Fatalf("copyOSC52() error = %v", err)
			}

			out := buf.String()

			// Must start with OSC 52 prefix.
			prefix := "\x1b]52;c;"
			if !strings.HasPrefix(out, prefix) {
				t.Errorf("output should start with OSC 52 prefix, got %q", out)
			}

			// Must end with BEL.
			if !strings.HasSuffix(out, "\x07") {
				t.Errorf("output should end with BEL (\\x07), got %q", out)
			}

			// Extract and decode the base64 payload.
			payload := out[len(prefix) : len(out)-1]
			decoded, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				t.Fatalf("base64 decode error: %v", err)
			}

			if string(decoded) != tt.input {
				t.Errorf("decoded = %q, want %q", string(decoded), tt.input)
			}
		})
	}
}
