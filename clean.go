package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dir := fs.String("dir", ".", "directory to process")
	extFlag := fs.String("ext", "jpg,jpeg", "comma-separated file extensions")
	dryRun := fs.Bool("dry-run", false, "show what would change without writing")
	backup := fs.Bool("backup", false, "write .bak copy before modifying")
	cfgPath := fs.String("config", "", "path to TOML config file")
	fs.Parse(args)

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tidy-exif: %v\n", err)
		die(1)
	}

	exts := parseExtList(*extFlag)
	files, err := walkFiles(*dir, exts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tidy-exif: %v\n", err)
		die(1)
	}

	verb := "Cleaning"
	if *dryRun {
		verb = "Dry run"
	}
	fmt.Printf("%s: %s (%s)...\n\n", verb, *dir, strings.Join(exts, ", "))

	var cleaned, skipped, errors int
	for _, path := range files {
		name := filepath.Base(path)

		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", name, err)
			errors++
			continue
		}

		report, err := InspectJPEG(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", name, err)
			errors++
			continue
		}

		if !report.HasAdobeData() {
			fmt.Printf("  %-30s skipped (no Adobe metadata)\n", name)
			skipped++
			continue
		}

		if *dryRun {
			printDryRun(name, report, cfg.Replacements)
			cleaned++
			continue
		}

		if *backup {
			if err := copyFile(path, path+".bak"); err != nil {
				fmt.Fprintf(os.Stderr, "  %-30s error (backup failed): %v\n", name, err)
				errors++
				continue
			}
		}

		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", name, err)
			errors++
			continue
		}

		_, result, err := CleanJPEG(data, cfg.Replacements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", name, err)
			errors++
			continue
		}

		if err := os.WriteFile(path, result, info.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "  %-30s error: %v\n", name, err)
			errors++
			continue
		}

		fmt.Printf("  %-30s cleaned\n", name)
		cleaned++
	}

	fmt.Println()
	if *dryRun {
		fmt.Printf("Dry run complete. Would clean: %d   Would skip: %d\n", cleaned, skipped)
	} else {
		fmt.Printf("Cleaned: %d   Skipped: %d   Errors: %d\n", cleaned, skipped, errors)
	}

	if errors > 0 {
		die(1)
	}
}

// printDryRun shows what fields would be changed in a file without writing.
func printDryRun(name string, report *FileReport, replacements map[string]string) {
	repl := func(key string) string {
		if v, ok := replacements[key]; ok {
			return v
		}
		return ""
	}

	fmt.Printf("  %s [dry-run]\n", name)

	type change struct{ field, from, to string }
	var changes []change

	if xmp := report.XMP; xmp != nil {
		if xmp.CreatorTool != "" {
			changes = append(changes, change{"CreatorTool", xmp.CreatorTool, repl("CreatorTool")})
		}
		if xmp.MetadataDate != "" {
			changes = append(changes, change{"MetadataDate", xmp.MetadataDate, repl("MetadataDate")})
		}
		if xmp.DocumentID != "" {
			changes = append(changes, change{"DocumentID", xmp.DocumentID, repl("DocumentID")})
		}
		if xmp.InstanceID != "" {
			changes = append(changes, change{"InstanceID", xmp.InstanceID, repl("InstanceID")})
		}
		if xmp.OriginalDocumentID != "" {
			changes = append(changes, change{"OriginalDocumentID", xmp.OriginalDocumentID, repl("OriginalDocumentID")})
		}
		for i, agent := range xmp.SoftwareAgents {
			if agent != "" {
				changes = append(changes, change{
					fmt.Sprintf("SoftwareAgent[%d]", i),
					agent,
					repl("SoftwareAgent"),
				})
			}
		}
	}
	if report.ExifSoftware != "" {
		changes = append(changes, change{"EXIF:Software", report.ExifSoftware, repl("Software")})
	}

	for _, c := range changes {
		to := `""`
		if c.to != "" {
			to = fmt.Sprintf("%q", c.to)
		}
		fmt.Printf("    %-22s %q → %s\n", c.field+":", truncate(c.from, 48), to)
	}
}

// copyFile copies src to dst, preserving the source file's permissions.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode())
}
