package roomctl

import (
	"encoding/json"
	"os"
)

// Manifest describes a built room bundle.
type Manifest struct {
	SchemaVersion    int      `json:"schemaVersion"`
	Slug             string   `json:"slug"`
	Engine           string   `json:"engine"`
	Version          int      `json:"version"`
	BundleHashSha256 string   `json:"bundleHashSha256"`
	Files            []string `json:"files"`
	BuiltAt          string   `json:"builtAt"`
}

func writeManifest(path string, m Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
