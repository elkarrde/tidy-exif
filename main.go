package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	version   = "0.1.0"
	build     = "1"
	buildDate = "2026-05-22"
)

func main() {
	args := normaliseArgs(os.Args[1:])
	os.Args = append(os.Args[:1], args...)

	if len(os.Args) < 2 {
		printHelp()
		waitIfWindows()
		os.Exit(0)
	}

	cmd := os.Args[1]
	switch cmd {
	case "check":
		runCheck(os.Args[2:])
	case "clean":
		runClean(os.Args[2:])
	case "--version", "-version":
		printVersion()
		waitIfWindows()
		os.Exit(0)
	case "--help", "-help", "-h":
		printHelp()
		waitIfWindows()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "tidy-exif: unknown command %q\n\n", cmd)
		printHelp()
		die(1)
	}

	waitIfWindows()
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

// die prints the Windows pause if applicable and exits with code.
func die(code int) {
	waitIfWindows()
	os.Exit(code)
}

// waitIfWindows pauses for Enter on Windows so the console window stays open
// when the tool is launched by double-clicking the .exe.
func waitIfWindows() {
	if runtime.GOOS == "windows" {
		fmt.Println("\nPress <Enter> to close.")
		fmt.Scanln()
	}
}

func printVersion() {
	fmt.Printf("tidy-exif %s (build %s, %s)\n", version, build, buildDate)
}

func printHelp() {
	fmt.Printf(`tidy-exif %s - remove Adobe software signatures from image metadata

Usage:
  tidy-exif check [options]   scan files, report Adobe metadata fields
  tidy-exif clean [options]   remove or replace Adobe metadata fields

Options (both /flag and --flag accepted on all platforms):
  /dir PATH       directory to process (default: ./)
  /ext LIST       file extensions, comma-separated (default: jpg,jpeg)
  /dry-run        [clean] show what would change without writing
  /backup         [clean] write .bak backup before modifying each file
  /config PATH    [clean] TOML file mapping field names to replacement values
  /version        print version and exit
`, version)
}
