// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"os"
	"path/filepath"
	"strings"
)

// walkFiles returns the full paths of all files in root whose extension matches
// one of the given extensions (case-insensitive; leading dot optional).
// Subdirectories are not descended into.
func walkFiles(root string, extensions []string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	exts := normaliseExts(extensions)

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if matchExt(ext, exts) {
			files = append(files, filepath.Join(root, entry.Name()))
		}
	}
	return files, nil
}

// parseExtList splits a comma-separated extension string (e.g. "jpg,jpeg") into a slice.
func parseExtList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// normaliseExts lowercases extensions and ensures each has a leading dot.
func normaliseExts(exts []string) []string {
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, e)
	}
	return out
}

func matchExt(ext string, exts []string) bool {
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}
