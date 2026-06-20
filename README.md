# Tidy-EXIF

**tidy-exif** is a small Go CLI tool that removes Adobe software signatures from image EXIF/XMP metadata.

Adobe tools (Lightroom, Photoshop, Bridge, Camera Raw) embed software identification strings into image metadata when saving or exporting. These fields accumulate across edits and carry no photographic value — only information about which proprietary software touched the file. tidy-exif finds those fields and empties them (or replaces them with a configured value).

## Background

JPEG files can contain an XMP metadata block alongside standard EXIF data. Adobe writes to several XMP fields as a matter of course:

- `xmp:CreatorTool` — the application that last wrote the file (e.g. `Adobe Photoshop Lightroom Classic 13.0`)
- `xmpMM:History[*]/stEvt:softwareAgent` — a log of every save operation and which application performed it
- `xmp:MetadataDate` — when the metadata was last written (distinct from the photo's capture date)
- `xmpMM:DocumentID` and `xmpMM:InstanceID` — GUIDs linking file versions within Adobe's ecosystem
- `xmpMM:OriginalDocumentID` — traces the file back to its original Adobe-managed source

Adobe also writes the **EXIF `Software` tag** (IFD0 tag 0x0131, e.g. `Adobe Photoshop CS6 (Windows)`). tidy-exif cleans this too, but **only when its value is an Adobe signature** — non-Adobe Software tags (camera firmware, scanner software such as VueScan, etc.) are legitimate metadata and are left untouched.

None of these affect image quality or capture data. tidy-exif replaces their values with empty strings, or with a value you configure.

## Usage

```
tidy-exif check [options]
tidy-exif clean [options]
```

Both `/option` and `--option` flag styles are accepted on all platforms.

### Actions

**`check`** — Scan a directory and report which files contain Adobe metadata fields and what values are present. No files are modified.

**`clean`** — Remove (or replace) Adobe metadata fields in all matching image files.

### Options

| Flag | Default | Description |
|------|:-------:|-------------|
| `/dir` `--dir` | `./` | Directory to process |
| `/ext` `--ext` | `jpg,jpeg` | Comma-separated list of file extensions to process |
| `/dry-run` `--dry-run` | off | Show what would change without writing anything |
| `/backup` `--backup` | off | Write a `.bak` copy of each file before modifying |
| `/config` `--config` | _(none)_ | Path to a TOML config file specifying replacement values |
| `/version` `--version` | — | Print version and exit |

### Examples

Preview which files are affected in the current directory:
```
tidy-exif check
```

Clean all JPEGs in a specific folder, creating backups first:
```
tidy-exif clean /dir=D:\Photos\Export /backup
```

Dry run to verify what would be changed:
```
tidy-exif clean /dir=D:\Photos\Export /dry-run
```

Use a config file to replace fields with custom values instead of emptying them:
```
tidy-exif clean /dir=D:\Photos\Export /config=tidy-exif.toml
```

## Config file

When `/config` is provided, tidy-exif reads replacement values from a TOML file. Any field not specified in the config will be emptied (default behaviour).

Example `tidy-exif.toml`:
```toml
[replacements]
CreatorTool       = ""
MetadataDate      = ""
DocumentID        = ""
InstanceID        = ""
OriginalDocumentID = ""
# HistorySoftwareAgent entries will be emptied unless specified here
SoftwareAgent     = ""
# EXIF IFD0 Software tag (0x0131); only cleaned when it is an Adobe signature
Software          = ""
```

If no config file is provided, all targeted fields are emptied.

## Fields cleaned

| XMP Field | Description |
|-----------|-------------|
| `xmp:CreatorTool` | Last application to write the file |
| `xmpMM:History[]/stEvt:softwareAgent` | Per-save software agent log |
| `xmp:MetadataDate` | Date metadata was last modified |
| `xmpMM:DocumentID` | Adobe document GUID |
| `xmpMM:InstanceID` | Adobe instance GUID |
| `xmpMM:OriginalDocumentID` | Original document GUID |
| EXIF `Software` (IFD0 0x0131) | Writing application — cleaned only when it is an Adobe signature |

## Building

Requires Go 1.16+. The CLI lives in `./cmd/tidy-exif` (the engine is in
`internal/meta`).

```
go build -o tidy-exif ./cmd/tidy-exif
```

Cross-compile for Windows from Linux/macOS:
```
GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe ./cmd/tidy-exif
```

Cross-compile for 64-bit ARM Linux (e.g. Pine64, Raspberry Pi):
```
GOOS=linux GOARCH=arm64 go build -o tidy-exif-arm64 ./cmd/tidy-exif
```
The tool is pure Go (no cgo), so cross-compilation produces a static binary —
no need to build on the target board.

A `Makefile` with `build` / `build-windows` / `build-arm64` targets is provided. To install:
`go install codeberg.org/elkarrde/tidy-exif/cmd/tidy-exif@latest`.

### Release archives

```
make dist
```

builds both binaries and packages release archives in `dist/`, each bundling the
binary together with `LICENSE` and `README.md`:

- `tidy-exif-<version>-linux-amd64.tar.gz`
- `tidy-exif-<version>-linux-arm64.tar.gz`
- `tidy-exif-<version>-windows-amd64.zip`

The version is read from `cmd/tidy-exif/main.go`. Bundling `LICENSE` with the
binary keeps the distribution MPL-2.0 compliant — recipients of the executable
also receive the license and a pointer to the source. (`make dist` requires the
`zip` CLI for the Windows archive.)

## Relation to exif2xlsx

tidy-exif shares its file-walking approach with the [exif2xlsx](../exif2xlsx/) project in this repository. Unlike exif2xlsx, it does **not** depend on the `goexif` library: both the `check` (read) and `clean` (write) paths operate directly on the raw JPEG segment bytes via a hand-rolled parser — the XMP APP1 segment (`xmp.go`) and the Exif APP1 Software tag (`exif.go`), unified in `inspect.go`. The tool's only third-party dependency is `github.com/BurntSushi/toml` for config parsing.

## License

Mozilla Public License 2.0 (MPL-2.0) — see [`LICENSE`](LICENSE).

-----

Main repo: [Codeberg](https://codeberg.org/elkarrde/tidy-exif)
