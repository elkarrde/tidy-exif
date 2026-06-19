# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running / building

```bash
make build            # build the native binary
make build-windows    # cross-compile tidy-exif.exe (GOOS=windows)
make clean
go test ./...         # run the test suite (passing)
```

Usage: `tidy-exif check` and `tidy-exif clean`. Both `/flag` (Windows) and
`--flag` (Unix) styles are accepted on all platforms.

## Architecture

Single-package Go CLI that empties Adobe software-identifier fields from a JPEG's
XMP (APP1) segment and its Exif `Software` tag (IFD0 0x0131) via raw byte
manipulation. There is **no** `goexif` (or other EXIF-library) dependency: both
the read (`check`) and write (`clean`) paths use a hand-rolled JPEG segment
parser, since the cleaning step needs to rewrite bytes that read-only EXIF
libraries cannot. The only third-party dependency is `github.com/BurntSushi/toml`
for config parsing.

- `main.go` — version constants, arg/flag normalisation, subcommand dispatch
- `files.go` — directory walking, case-insensitive extension matching
- `xmp.go` — find/parse/clean/marshal/replace the XMP APP1 segment
- `exif.go` — Exif IFD0 Software tag (0x0131) read/clean; Adobe-only gate
- `inspect.go` — `InspectJPEG` / `CleanJPEG` unify XMP + Exif in one parse/write
- `jpeg.go` — JPEG segment helpers
- `check.go` / `clean.go` — the two commands
- `config.go` — optional TOML config of replacement values (default: empty all)
- `*_test.go` — unit tests with JPEG/XMP/Exif fixtures

Both edits are length-preserving so downstream JPEG offsets never move: cleaned
XMP is whitespace-padded, and the Exif Software value is overwritten in place
(NUL-padded). The Exif Software tag is only cleaned when its value contains an
Adobe signature — non-Adobe Software (camera firmware, VueScan, …) is preserved.

## Conventions

Extensions `jpg,jpeg` only. Default action empties all target fields; `/backup`
opts into a `.bak` copy; `/dry-run` reports without writing.

`TODO.md` reflects actual code state: phases 1–8 are implemented and checked off;
only the Phase 9 release tasks remain (real-JPEG smoke test, tag `v0.1.0`, attach
binaries). Keep its checkboxes in sync when those land.

## Housekeeping

Update `STATUS.md` (and reconcile `TODO.md`) as part of any change that alters
build, test, version, or release state.
