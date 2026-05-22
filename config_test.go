package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg, err := loadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Replacements) != 0 {
		t.Errorf("default config Replacements len = %d, want 0", len(cfg.Replacements))
	}
}

func TestLoadConfigValid(t *testing.T) {
	path := writeTempTOML(t, `
[replacements]
CreatorTool = "cleaned"
DocumentID  = ""
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := cfg.Replacements["CreatorTool"]; v != "cleaned" {
		t.Errorf("CreatorTool = %q, want %q", v, "cleaned")
	}
	if v, ok := cfg.Replacements["DocumentID"]; !ok || v != "" {
		t.Errorf("DocumentID = %q present=%v, want empty string present", v, ok)
	}
}

func TestLoadConfigAllFields(t *testing.T) {
	path := writeTempTOML(t, `
[replacements]
CreatorTool        = "a"
MetadataDate       = "b"
DocumentID         = "c"
InstanceID         = "d"
OriginalDocumentID = "e"
SoftwareAgent      = "f"
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Replacements) != 6 {
		t.Errorf("got %d replacements, want 6", len(cfg.Replacements))
	}
}

func TestLoadConfigUnknownKeyDropped(t *testing.T) {
	path := writeTempTOML(t, `
[replacements]
CreatorTool  = ""
UnknownField = "foo"
`)
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Replacements["UnknownField"]; ok {
		t.Error("unknown field should have been dropped")
	}
	if _, ok := cfg.Replacements["CreatorTool"]; !ok {
		t.Error("valid field CreatorTool missing")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := loadConfig(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	path := writeTempTOML(t, `this is not valid toml ===`)
	_, err := loadConfig(path)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tidy-exif.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
