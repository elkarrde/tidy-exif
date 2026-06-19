// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalkFiles(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"a.jpg", "b.JPEG", "c.JPG", "d.png", "e.txt", "f.jpg"} {
		os.WriteFile(filepath.Join(tmp, name), []byte{}, 0644)
	}
	// subdir with a JPEG — must NOT appear (non-recursive)
	sub := filepath.Join(tmp, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "nested.jpg"), []byte{}, 0644)

	got, err := walkFiles(tmp, []string{"jpg", "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	// a.jpg, b.JPEG, c.JPG, f.jpg — 4 matches
	if len(got) != 4 {
		t.Errorf("got %d files, want 4: %v", len(got), got)
	}
}

func TestWalkFilesSubdirSkipped(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "subdir")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "photo.jpg"), []byte{}, 0644)

	got, err := walkFiles(tmp, []string{"jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no files, got %v", got)
	}
}

func TestWalkFilesNonexistentDir(t *testing.T) {
	_, err := walkFiles(filepath.Join(t.TempDir(), "does-not-exist"), []string{"jpg"})
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestWalkFilesExtNormalisation(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "photo.jpg"), []byte{}, 0644)

	// leading dot optional, case-insensitive, duplicates de-duped via first match break
	got, err := walkFiles(tmp, []string{".JPG", "jpg", ".Jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %d files, want 1: %v", len(got), got)
	}
}

func TestWalkFilesEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	got, err := walkFiles(tmp, []string{"jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected no files in empty dir, got %v", got)
	}
}

func TestParseExtList(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"jpg,jpeg", []string{"jpg", "jpeg"}},
		{"jpg, jpeg", []string{"jpg", "jpeg"}},
		{"JPG", []string{"JPG"}},
		{"", []string{}},
		{" , ", []string{}},
	}
	for _, c := range cases {
		got := parseExtList(c.input)
		if len(got) != len(c.want) {
			t.Errorf("parseExtList(%q): got %v, want %v", c.input, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseExtList(%q)[%d] = %q, want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}
