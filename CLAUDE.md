# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running / building

```bash
make build            # build the native binary
make build-windows    # cross-compile tidy-exif.exe (GOOS=windows)
make dist             # build both + package dist/ archives (binary + LICENSE + README)
make clean
go test ./...         # run the test suite (passing)
```

Usage: `tidy-exif check` and `tidy-exif clean`. Both `/flag` (Windows) and
`--flag` (Unix) styles are accepted on all platforms.

## Architecture

Go CLI that empties Adobe software-identifier fields from a JPEG's XMP (APP1)
segment and its Exif `Software` tag (IFD0 0x0131) via raw byte manipulation. There
is **no** `goexif` (or other EXIF-library) dependency: both the read (`check`) and
write (`clean`) paths use a hand-rolled JPEG segment parser, since the cleaning step
needs to rewrite bytes that read-only EXIF libraries cannot. The only third-party
dependency is `github.com/BurntSushi/toml` for config parsing.

Layout is lapis-style (`cmd/` binary + `internal/` engine):

- `cmd/tidy-exif/` — the CLI (`package main`)
  - `main.go` — version constants, arg/flag normalisation, subcommand dispatch
  - `check.go` / `clean.go` — the two commands
  - `config.go` — optional TOML config of replacement values (default: empty all)
  - `files.go` — directory walking, case-insensitive extension matching
- `internal/meta/` — the metadata engine (`package meta`); this is the code slated
  to extract into the `exifscalpel` library
  - `jpeg.go` — JPEG segment parser/writer
  - `xmp.go` — find/parse/clean/marshal the XMP APP1 segment
  - `exif.go` — Exif IFD0 Software tag (0x0131) read/clean; Adobe-only gate
  - `inspect.go` — `InspectJPEG` / `CleanJPEG` unify XMP + Exif in one parse/write
- `*_test.go` — unit tests alongside the code they exercise (CLI tests in
  `cmd/tidy-exif`, engine/white-box tests in `internal/meta`)

The CLI consumes the engine through `meta.InspectJPEG` / `meta.CleanJPEG`; the
already-exported `FileReport` / `XMPData` carry the results.

Both edits are length-preserving so downstream JPEG offsets never move: cleaned
XMP is whitespace-padded, and the Exif Software value is overwritten in place
(NUL-padded). The Exif Software tag is only cleaned when its value contains an
Adobe signature — non-Adobe Software (camera firmware, VueScan, …) is preserved.

## Conventions

Extensions `jpg,jpeg` only. Default action empties all target fields; `/backup`
opts into a `.bak` copy; `/dry-run` reports without writing.

`TODO.md` reflects actual code state: phases 1–8 plus the Exif Software work are
implemented and checked off, the real-JPEG smoke test is done, and `make dist`
packages the MPL-2.0-compliant release archives. Only the release task remains
(tag `v0.1.3`, run `make dist`, attach the archives). Keep its checkboxes in sync.

## Housekeeping

Update `STATUS.md` (and reconcile `TODO.md`) as part of any change that alters
build, test, version, or release state.
