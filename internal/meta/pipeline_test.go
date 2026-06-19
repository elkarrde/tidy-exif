// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package meta

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestCleanPipeline exercises the engine clean pipeline across multiple files:
// read → ParseXMPFromJPEG → (skip or CleanXMPInJPEG) → write → verify.
func TestCleanPipeline(t *testing.T) {
	tmp := t.TempDir()

	// Place two JPEGs: one with Adobe metadata, one without.
	withXMP := buildTestJPEGWithXMP(sampleXMP)
	if err := os.WriteFile(filepath.Join(tmp, "with.jpg"), withXMP, 0644); err != nil {
		t.Fatal(err)
	}

	bare := []byte{0xFF, 0xD8, 0xFF, 0xD9} // minimal JPEG, no XMP
	if err := os.WriteFile(filepath.Join(tmp, "bare.jpg"), bare, 0644); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(tmp, "*.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	var cleaned, skipped int
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		xmp, err := ParseXMPFromJPEG(data)
		if err != nil {
			t.Fatal(err)
		}

		if xmp == nil || !xmp.HasAdobeData() {
			skipped++
			continue
		}

		_, result, err := CleanXMPInJPEG(data, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, result, 0644); err != nil {
			t.Fatal(err)
		}
		cleaned++
	}

	if cleaned != 1 {
		t.Errorf("cleaned = %d, want 1", cleaned)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}

	// Verify the cleaned file no longer has Adobe data.
	data, _ := os.ReadFile(filepath.Join(tmp, "with.jpg"))
	xmp, err := ParseXMPFromJPEG(data)
	if err != nil {
		t.Fatal(err)
	}
	if xmp == nil {
		t.Fatal("XMP segment missing after clean")
	}
	if xmp.HasAdobeData() {
		t.Errorf("file still has Adobe data after clean: %+v", xmp)
	}
}

// TestCleanPipelineWithBackup verifies a pre-clean copy preserves the original bytes.
func TestCleanPipelineWithBackup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "photo.jpg")
	bak := path + ".bak"

	original := buildTestJPEGWithXMP(sampleXMP)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	// Back up (plain copy), then clean.
	if err := os.WriteFile(bak, original, 0644); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	_, result, err := CleanXMPInJPEG(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, result, 0644)

	// .bak should contain original, unmodified content.
	bakData, err := os.ReadFile(bak)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, original) {
		t.Error(".bak file content does not match original")
	}

	// Cleaned file should differ from original.
	cleaned, _ := os.ReadFile(path)
	if bytes.Equal(cleaned, original) {
		t.Error("cleaned file is identical to original — expected changes")
	}
}
