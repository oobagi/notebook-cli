package styles

import (
	"regexp"
	"testing"
)

func TestValidateAllCommunityStyles(t *testing.T) {
	for _, name := range List() {
		t.Run(name, func(t *testing.T) {
			data, err := Load(name)
			if err != nil {
				t.Fatalf("Load(%q): %v", name, err)
			}
			if err := Validate(data); err != nil {
				t.Errorf("Validate(%q): %v", name, err)
			}
		})
	}
}

var kebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestCommunityStyleNaming(t *testing.T) {
	for _, name := range List() {
		t.Run(name, func(t *testing.T) {
			if !kebabRe.MatchString(name) {
				t.Errorf("style name %q is not kebab-case", name)
			}
		})
	}
}

func TestValidateMalformedJSON(t *testing.T) {
	err := Validate([]byte(`{invalid json`))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestValidateMissingDocument(t *testing.T) {
	err := Validate([]byte(`{"heading": {"bold": true}}`))
	if err == nil {
		t.Error("expected error for missing document block")
	}
}
