# tidy-exif — TODO

This file is the implementation guide for Claude Code (or any future session).
Work through phases in order; each phase should leave the project in a buildable state.

---

## Context

- Language: Go 1.21+
- Primary platform: Windows (`.exe`), cross-compiled from Linux/macOS if needed
- Related project in same repo: `../exif2xlsx/` — reuse patterns from there
- Key constraint: `github.com/rwcarlsen/goexif` is **read-only**; XMP writes must be done via raw byte manipulation of the JPEG APP1 segment
- Flag style: support both `/flag` (Windows) and `--flag` (Unix) transparently

---

## Phase 1 — Scaffold

- [ ] Create `go.mod` with module name `codeberg.org/elkarrde/tidy-exif`, `go 1.21`
- [ ] Add dependency: `github.com/rwcarlsen/goexif` (same version as exif2xlsx)
- [ ] Add dependency: `github.com/BurntSushi/toml` for config file parsing
- [ ] Create `Makefile` with targets: `build`, `build-windows`, `clean`
- [ ] Create `main.go` with version/build constants (follow exif2xlsx pattern) and subcommand dispatch

---

## Phase 2 — Flag normalisation

- [ ] Write a `normaliseArgs()` function (or equivalent) that rewrites `os.Args` before `flag.Parse()`, converting any `/flag` or `/flag=value` token to `--flag` / `--flag=value`
- [ ] This must handle boolean flags (`/dry-run`), value flags (`/dir=PATH`), and space-separated values (`/dir PATH`)
- [ ] Write a unit test: given a mixed slice of `/` and `--` args, assert the normalised output

---

## Phase 3 — File walking (`files.go`)

- [ ] Port `IOReadDir` from `../exif2xlsx/exif2xls.go` into `files.go`
- [ ] Extend to accept `root string` and `extensions []string` parameters instead of hardcoded values
- [ ] Extensions should be matched case-insensitively (`.JPG` == `.jpg`)
- [ ] Write a basic test against a temp directory with dummy files

---

## Phase 4 — XMP segment handling (`xmp.go`)

This is the core of the tool. The XMP block in a JPEG is an APP1 segment identified by the marker `0xFF 0xE1` followed by the namespace URI `http://ns.adobe.com/xap/1.0/\x00`.

- [ ] Write `findXMPSegment(data []byte) (start, end int, found bool)` — locates the XMP APP1 segment in raw JPEG bytes
- [ ] Write `parseXMP(segment []byte) (*XMPData, error)` — unmarshals the XMP XML using `encoding/xml` from stdlib; populate a struct covering all six target fields (see README)
- [ ] Write `cleanXMP(xmp *XMPData, replacements map[string]string) *XMPData` — applies replacement values (empty string by default) to all target fields; `xmpMM:History` entries should have their `stEvt:softwareAgent` attribute/element zeroed, but the history entries themselves should not be deleted (preserve structure)
- [ ] Write `marshalXMP(xmp *XMPData) ([]byte, error)` — serialises back to XML; the result must be the same byte length or the segment must be padded/rebuilt at the correct JPEG offset
- [ ] Write `replaceXMPSegment(data []byte, newSegment []byte) ([]byte, error)` — splices the new segment back into the JPEG byte stream
- [ ] All functions should be independently testable; provide test fixtures (a minimal JPEG with known XMP content)

**Note on length:** If the cleaned XMP XML is shorter than the original, pad with whitespace inside the XML to preserve segment length, avoiding the need to rewrite all subsequent JPEG segment offsets.

---

## Phase 5 — Config file (`config.go`)

- [ ] Define `Config` struct with a `[replacements]` table mapping field names to replacement strings
- [ ] Write `loadConfig(path string) (Config, error)` using `github.com/BurntSushi/toml`
- [ ] If no config path is provided, return a default `Config` with all replacement values set to `""`
- [ ] Validate that config keys are recognised field names; warn (do not error) on unknown keys

---

## Phase 6 — `check` command

- [ ] Walk the target directory with the file walker
- [ ] For each file: open, find XMP segment, parse, report presence/value of each target field
- [ ] Output: tabular, one file per line, columns: filename, fields present (Y/N or value preview), similar style to `exif2xlsx check`
- [ ] Honour `/ext` filter
- [ ] Summary line at the end: N files scanned, M with Adobe metadata

---

## Phase 7 — `clean` command

- [ ] Walk the target directory
- [ ] For each file:
  - [ ] If `/dry-run`: parse XMP, compute what would change, print diff-style summary, skip write
  - [ ] If `/backup`: copy `file.jpg` → `file.jpg.bak` before any modification
  - [ ] Read file bytes, find XMP segment, parse, clean, rewrite segment, write back to original path
  - [ ] Print per-file status: `cleaned`, `skipped (no Adobe metadata)`, `error`
- [ ] Summary line at the end: N cleaned, M skipped, P errors

---

## Phase 8 — Polish

- [ ] `printHelp()` — follows exif2xlsx style, lists both `/` and `--` flag forms
- [ ] `--version` flag prints version, build number, date
- [ ] On Windows (`runtime.GOOS == "windows"`), print "Press <Enter> to close." and wait for input before exit (matches exif2xlsx behaviour)
- [ ] Ensure the tool exits cleanly (exit code 0 on success, 1 on error) — useful for scripting

---

## Phase 9 — Build & release

- [ ] Confirm `go build` produces a working binary
- [ ] Confirm `GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe .` works from Linux
- [ ] Test against real Lightroom-exported JPEGs
- [ ] Tag `v0.1.0` on Codeberg

---

## Open decisions (resolved)

- Extensions: `jpg,jpeg` only (no TIFF)
- Default action: empty all fields (zero-length string)
- Backup: opt-in via `/backup` flag, not default
- Config: TOML, optional; no config = empty everything
- Flag style: both `/` and `--` accepted
- In-place modification: yes, with optional backup
