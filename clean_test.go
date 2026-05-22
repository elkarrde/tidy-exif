package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.jpg")
	dst := filepath.Join(tmp, "dst.jpg")

	content := []byte("test content for copy")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("copied content does not match source")
	}

	// permissions should be preserved
	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("mode mismatch: src %v, dst %v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile(filepath.Join(tmp, "missing.jpg"), filepath.Join(tmp, "out.jpg"))
	if err == nil {
		t.Error("expected error copying missing source")
	}
}

// TestCleanPipeline exercises the full file-level clean pipeline:
// walk → read → CleanXMPInJPEG → write → verify.
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

	files, err := walkFiles(tmp, []string{"jpg"})
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

// TestCleanPipelineWithBackup verifies that .bak files are created before modification.
func TestCleanPipelineWithBackup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "photo.jpg")
	bak := path + ".bak"

	original := buildTestJPEGWithXMP(sampleXMP)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	// Backup then clean
	if err := copyFile(path, bak); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	_, result, err := CleanXMPInJPEG(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, result, 0644)

	// .bak should contain original, unmodified content
	bakData, err := os.ReadFile(bak)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bakData, original) {
		t.Error(".bak file content does not match original")
	}

	// Cleaned file should differ from original
	cleaned, _ := os.ReadFile(path)
	if bytes.Equal(cleaned, original) {
		t.Error("cleaned file is identical to original — expected changes")
	}
}
