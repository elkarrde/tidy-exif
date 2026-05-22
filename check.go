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

	fmt.Printf("  %3s  %-24s  %-30s  %-19s  %3s  %4s\n",
		"#", "Filename", "CreatorTool", "Date", "IDs", "Hist")
	fmt.Printf("  %s\n", strings.Repeat("-", 93))

	var withAdobe int
	for i, path := range files {
		name := filepath.Base(path)

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %3d  %-24s  error: %v\n", i+1, truncate(name, 24), err)
			continue
		}

		xmp, err := ParseXMPFromJPEG(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %3d  %-24s  error: %v\n", i+1, truncate(name, 24), err)
			continue
		}

		if xmp == nil || !xmp.HasAdobeData() {
			fmt.Printf("  %3d  %-24s  (no Adobe metadata)\n", i+1, truncate(name, 24))
			continue
		}

		withAdobe++
		idCount := 0
		if xmp.DocumentID != "" {
			idCount++
		}
		if xmp.InstanceID != "" {
			idCount++
		}
		if xmp.OriginalDocumentID != "" {
			idCount++
		}

		fmt.Printf("  %3d  %-24s  %-30s  %-19s  %3d  %4d\n",
			i+1,
			truncate(name, 24),
			truncate(xmp.CreatorTool, 30),
			truncate(xmp.MetadataDate, 19),
			idCount,
			len(xmp.SoftwareAgents),
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
