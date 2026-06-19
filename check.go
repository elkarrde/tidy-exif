// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	dir := fs.String("dir", ".", "directory to scan")
	extFlag := fs.String("ext", "jpg,jpeg", "comma-separated file extensions")
	fs.Parse(args)

	exts := parseExtList(*extFlag)
	files, err := walkFiles(*dir, exts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tidy-exif: %v\n", err)
		die(1)
	}

	fmt.Printf("Scanning %s (%s)...\n", *dir, strings.Join(exts, ", "))
	fmt.Printf("Found %d file(s).\n\n", len(files))

	if len(files) == 0 {
		return
	}

	fmt.Printf("  %3s  %-24s  %-26s  %3s  %4s  %-26s\n",
		"#", "Filename", "CreatorTool", "IDs", "Hist", "EXIF Software")
	fmt.Printf("  %s\n", strings.Repeat("-", 95))

	var withAdobe int
	for i, path := range files {
		name := filepath.Base(path)

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %3d  %-24s  error: %v\n", i+1, truncate(name, 24), err)
			continue
		}

		report, err := InspectJPEG(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %3d  %-24s  error: %v\n", i+1, truncate(name, 24), err)
			continue
		}

		if !report.HasAdobeData() {
			fmt.Printf("  %3d  %-24s  (no Adobe metadata)\n", i+1, truncate(name, 24))
			continue
		}

		withAdobe++
		var creatorTool string
		var idCount, histCount int
		if x := report.XMP; x != nil {
			creatorTool = x.CreatorTool
			histCount = len(x.SoftwareAgents)
			for _, id := range []string{x.DocumentID, x.InstanceID, x.OriginalDocumentID} {
				if id != "" {
					idCount++
				}
			}
		}

		fmt.Printf("  %3d  %-24s  %-26s  %3d  %4d  %-26s\n",
			i+1,
			truncate(name, 24),
			truncate(creatorTool, 26),
			idCount,
			histCount,
			truncate(report.ExifSoftware, 26),
		)
	}

	fmt.Printf("\nScanned: %d   With Adobe metadata: %d\n", len(files), withAdobe)
}

// truncate shortens s to at most n bytes, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
