// Package styles provides embedded community Glamour style definitions.
//
// The JSON files live under community/ and are compiled into the binary
// via go:embed so no external file access is needed at runtime.
package styles

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed community/*.json
var communityFS embed.FS

// List returns the names of all embedded community styles, sorted
// alphabetically. Names are the JSON filenames without the .json extension
// (e.g. "gruvbox", "nord").
func List() []string {
	entries, err := communityFS.ReadDir("community")
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(names)
	return names
}

// Load returns the raw JSON bytes of the named community style.
// The name should not include the .json extension.
func Load(name string) ([]byte, error) {
	data, err := communityFS.ReadFile(path.Join("community", name+".json"))
	if err != nil {
		return nil, fmt.Errorf("community style %q not found", name)
	}
	return data, nil
}

// Has reports whether a community style with the given name exists.
func Has(name string) bool {
	_, err := communityFS.ReadFile(path.Join("community", name+".json"))
	return err == nil
}
