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
XMP (APP1) segment via raw byte manipulation — `github.com/rwcarlsen/goexif` is
read-only, so XMP writes are done by hand.

- `main.go` — version constants, arg/flag normalisation, subcommand dispatch
- `files.go` — directory walking, case-insensitive extension matching
- `xmp.go` — find/parse/clean/marshal/replace the XMP APP1 segment
- `jpeg.go` — JPEG segment helpers
- `check.go` / `clean.go` — the two commands
- `config.go` — optional TOML config of replacement values (default: empty all)
- `*_test.go` — unit tests with JPEG/XMP fixtures

Cleaned XMP is padded with whitespace to preserve segment length, avoiding a
rewrite of downstream JPEG offsets.

## Conventions

Extensions `jpg,jpeg` only. Default action empties all target fields; `/backup`
opts into a `.bak` copy; `/dry-run` reports without writing.

## Housekeeping

Update `STATUS.md` (and reconcile `TODO.md`) as part of any change that alters
build, test, version, or release state.
