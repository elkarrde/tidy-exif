# tidy-exif — TODO

This file started as the implementation guide for the v0.1.0 build. Phases 1–8 are
implemented and the test suite passes; only the release tasks in Phase 9 remain.
Checkboxes below reflect actual code state as of 2026-06-19, with notes where the
implementation diverged from the original plan.

---

## Context

- Language: Go (`go.mod` declares **`go 1.16`** — note: not 1.21 as originally planned)
- Primary platform: Windows (`.exe`), cross-compiled from Linux/macOS if needed
- Related project in same repo: `../exif2xlsx/` — file-walking pattern reused from there
- Key constraint: XMP writes are done via raw byte manipulation of the JPEG APP1 segment
- **Divergence:** the planned `goexif` dependency was dropped — the read/check path is
  also hand-rolled (see `jpeg.go` / `xmp.go`), so the only third-party dep is
  `github.com/BurntSushi/toml`
- Flag style: support both `/flag` (Windows) and `--flag` (Unix) transparently

---

## Phase 1 — Scaffold

- [x] Create `go.mod` with module name `codeberg.org/elkarrde/tidy-exif` *(declares `go 1.16`)*
- [x] ~~Add dependency: `github.com/rwcarlsen/goexif`~~ — dropped; not used, parser is hand-rolled
- [x] Add dependency: `github.com/BurntSushi/toml` for config file parsing
- [x] Create `Makefile` with targets: `build`, `build-windows`, `clean`
- [x] Create `main.go` with version/build constants and subcommand dispatch

---

## Phase 2 — Flag normalisation

- [x] `normaliseArgs()` rewrites `/flag` / `/flag=value` tokens to `--` form before `flag.Parse()`
- [x] Handles boolean, `=value`, and space-separated value flags
- [x] Unit test asserting normalised output (`main_test.go`)

---

## Phase 3 — File walking (`files.go`)

- [x] Directory walker reused from `../exif2xlsx/`, parameterised by `root` and `extensions`
- [x] Case-insensitive extension matching (`.JPG` == `.jpg`)
- [x] Test against a temp directory with dummy files (`files_test.go`)

---

## Phase 4 — XMP segment handling (`xmp.go` / `jpeg.go`)

The core of the tool. Implemented, though the function decomposition differs from the
original sketch: JPEG segments are parsed/rewritten via `parseJPEG` / `writeJPEG`
(`jpeg.go`), and the XMP orchestration lives in `ParseXMPFromJPEG` / `CleanXMPInJPEG`
(`xmp.go`), rather than the originally-named `findXMPSegment` / `replaceXMPSegment`.

- [x] Locate the XMP APP1 segment in raw JPEG bytes (`parseJPEG` + `isXMPSeg`)
- [x] `parseXMP` unmarshals the XMP XML (stdlib `encoding/xml`) into `XMPData`
- [x] `cleanXMP` applies replacement values; `xmpMM:History` entries have their
      `stEvt:softwareAgent` zeroed without deleting the history entries
- [x] `marshalXMP` serialises back to XML, length-preserved via `adjustPadding`
- [x] Splice the new segment back into the JPEG byte stream (`writeJPEG`)
- [x] Independently testable with JPEG/XMP fixtures (`xmp_test.go`)

**Length:** shorter cleaned XML is whitespace-padded (`adjustPadding`) to preserve
segment length and avoid rewriting downstream JPEG offsets.

---

## Phase 5 — Config file (`config.go`)

- [x] `Config` struct with a `[replacements]` table mapping field names to strings
- [x] `loadConfig(path)` using `github.com/BurntSushi/toml`
- [x] No config path → default `Config` with all replacements `""`
- [x] Recognised-key handling (`config_test.go`)

---

## Phase 6 — `check` command

- [x] Walks the target directory with the file walker
- [x] Per file: open, find XMP, parse, report presence/value of each target field
- [x] Tabular output, one file per line
- [x] Honours `/ext` filter
- [x] Summary line (files scanned, files with Adobe metadata)

---

## Phase 7 — `clean` command

- [x] Walks the target directory
- [x] `/dry-run`: computes and prints what would change, skips write (`printDryRun`)
- [x] `/backup`: copies `file.jpg` → `file.jpg.bak` before modifying (`copyFile`)
- [x] Read → find → parse → clean → rewrite segment → write back in place
- [x] Per-file status and end-of-run summary

---

## Phase 8 — Polish

- [x] `printHelp()` lists both `/` and `--` flag forms
- [x] `--version` / `/version` prints version, build, date
- [x] On Windows, prints "Press <Enter> to close." before exit
- [x] Clean exit codes (0 success / 1 error)

---

## Phase 9 — Build & release  *(remaining)*

- [x] `go build` produces a working binary
- [x] `GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe .` works from Linux *(verified 2026-06-19)*
- [x] `make build-arm64` (`GOOS=linux GOARCH=arm64`) cross-compiles a static binary from Linux *(2026-06-20; pure Go/no cgo, no need to build on the Pine64)*
- [x] Test against real Lightroom-exported JPEGs *(2026-06-19; found + fixed the attribute-form history bug)*
- [x] `make dist` packages Linux amd64/arm64 (`.tar.gz`) + Windows (`.zip`) archives, each bundling the binary, `LICENSE`, and `README.md` *(2026-06-20)*
- [x] Tag `v0.1.3` (annotated, local) + `make dist` archives built *(2026-06-20)*
- [x] Push the tag to Codeberg (`git push origin v0.1.3`) + upload the Linux/Windows archives to the release
- [x] Move hosting to GitHub: module path `github.com/elkarrde/tidy-exif`, exifscalpel bumped to v0.3.1 *(2026-09-21)*
- [x] Tag `v0.1.4`, push to GitHub, publish the GitHub release with all three `make dist` archives *(2026-09-21)*

---

## Phase 10 — Exif Software tag *(done 2026-06-19)*

Added during real-file smoke testing: Adobe also writes the Exif IFD0 Software
tag (0x0131), which the XMP-only cleaner left behind.

- [x] `exif.go` — parse TIFF/IFD0, read + length-preserving in-place clean of Software (0x0131)
- [x] Adobe-only gate (`isAdobeSoftware`) so camera/scanner Software (VueScan, firmware) is preserved
- [x] `inspect.go` — `InspectJPEG` / `CleanJPEG` unify XMP + Exif in one parse/write
- [x] `check`/`clean`/dry-run report and clean the Exif Software tag
- [x] `Software` config key; regression tests for attribute-form history + Exif

---

## Open decisions (resolved)

- Extensions: `jpg,jpeg` only (no TIFF)
- Default action: empty all fields (zero-length string)
- Backup: opt-in via `/backup` flag, not default
- Config: TOML, optional; no config = empty everything
- Flag style: both `/` and `--` accepted
- In-place modification: yes, with optional backup
- **EXIF library: none** — `goexif` was dropped in favour of a hand-rolled parser
