package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	args := normaliseArgs(os.Args[1:])
	os.Args = append(os.Args[:1], args...)

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "check":
		runCheck(os.Args[2:])
	case "clean":
		runClean(os.Args[2:])
	case "--version", "-version":
		fmt.Println(version)
	case "--help", "-help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "tidy-exif: unknown command %q\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

// normaliseArgs converts /flag and /flag=value tokens to --flag style.
// Paths containing multiple slashes (e.g. /usr/local/file) are not converted.
func normaliseArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "/") && strings.Count(arg, "/") == 1 {
			out[i] = "--" + arg[1:]
		} else {
			out[i] = arg
		}
	}
	return out
}

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	dir := fs.String("dir", ".", "directory to scan")
	ext := fs.String("ext", "jpg,jpeg", "comma-separated file extensions")
	fs.Parse(args)
	_, _ = dir, ext
	fmt.Fprintln(os.Stderr, "check: not yet implemented")
	os.Exit(1)
}

func runClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dir := fs.String("dir", ".", "directory to process")
	ext := fs.String("ext", "jpg,jpeg", "comma-separated file extensions")
	dryRun := fs.Bool("dry-run", false, "show what would change without writing")
	backup := fs.Bool("backup", false, "write .bak copy before modifying")
	cfg := fs.String("config", "", "path to TOML config file")
	fs.Parse(args)
	_, _, _, _, _ = dir, ext, dryRun, backup, cfg
	fmt.Fprintln(os.Stderr, "clean: not yet implemented")
	os.Exit(1)
}

func printHelp() {
	fmt.Printf(`tidy-exif %s - remove Adobe software signatures from image metadata

Usage:
  tidy-exif check [options]   scan files and report Adobe metadata fields
  tidy-exif clean [options]   remove or replace Adobe metadata fields

Options (both /flag and --flag accepted on all platforms):
  /dir PATH       directory to process (default: ./)
  /ext LIST       comma-separated extensions (default: jpg,jpeg)
  /dry-run        show changes without writing (clean only)
  /backup         write .bak copy before modifying (clean only)
  /config PATH    TOML config file for replacement values (clean only)
  /version        print version and exit
`, version)
}
