package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// knownFields is the set of field names accepted in [replacements].
var knownFields = map[string]bool{
	"CreatorTool":        true,
	"MetadataDate":       true,
	"DocumentID":         true,
	"InstanceID":         true,
	"OriginalDocumentID": true,
	"SoftwareAgent":      true,
	"Software":           true, // EXIF IFD0 Software tag (0x0131)
}

// Config holds per-field replacement values from a TOML config file.
// Any field absent from Replacements is treated as "" (empty) by the cleaner.
type Config struct {
	Replacements map[string]string
}

// defaultConfig returns a Config that will empty all target fields.
func defaultConfig() Config {
	return Config{Replacements: make(map[string]string)}
}

// loadConfig reads replacement values from a TOML file at path.
// If path is empty, the default config (empty all fields) is returned.
// Unknown keys in [replacements] produce a warning but are not an error.
func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()
	if path == "" {
		return cfg, nil
	}

	if _, err := os.Stat(path); err != nil {
		return cfg, fmt.Errorf("config file not found: %s", path)
	}

	var raw struct {
		Replacements map[string]string `toml:"replacements"`
	}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}

	for k, v := range raw.Replacements {
		if !knownFields[k] {
			fmt.Fprintf(os.Stderr, "tidy-exif: warning: unknown config key %q (ignored)\n", k)
			continue
		}
		cfg.Replacements[k] = v
	}

	return cfg, nil
}
