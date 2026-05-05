# tidy-exif

**tidy-exif** is a small Go CLI tool that removes Adobe software signatures from image EXIF/XMP metadata.

Adobe tools (Lightroom, Photoshop, Bridge, Camera Raw) embed software identification strings into image metadata when saving or exporting. These fields accumulate across edits and carry no photographic value — only information about which proprietary software touched the file. tidy-exif finds those fields and empties them (or replaces them with a configured value).

## Background

JPEG files can contain an XMP metadata block alongside standard EXIF data. Adobe writes to several XMP fields as a matter of course:

- `xmp:CreatorTool` — the application that last wrote the file (e.g. `Adobe Photoshop Lightroom Classic 13.0`)
- `xmpMM:History[*]/stEvt:softwareAgent` — a log of every save operation and which application performed it
- `xmp:MetadataDate` — when the metadata was last written (distinct from the photo's capture date)
- `xmpMM:DocumentID` and `xmpMM:InstanceID` — GUIDs linking file versions within Adobe's ecosystem
- `xmpMM:OriginalDocumentID` — traces the file back to its original Adobe-managed source

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
|---|---|---|
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
```

If no config file is provided, all targeted fields are emptied.

## Fields cleaned

| XMP Field | Description |
|---|---|
| `xmp:CreatorTool` | Last application to write the file |
| `xmpMM:History[]/stEvt:softwareAgent` | Per-save software agent log |
| `xmp:MetadataDate` | Date metadata was last modified |
| `xmpMM:DocumentID` | Adobe document GUID |
| `xmpMM:InstanceID` | Adobe instance GUID |
| `xmpMM:OriginalDocumentID` | Original document GUID |

## Building

Requires Go 1.21+.

```
go build -o tidy-exif.exe .
```

Cross-compile for Windows from Linux/macOS:
```
GOOS=windows GOARCH=amd64 go build -o tidy-exif.exe .
```

A `Makefile` with a `build-windows` target is provided.

## Relation to exif2xlsx

tidy-exif shares its file-walking and EXIF-reading approach with the [exif2xlsx](../exif2xlsx/) project in this repository. The `goexif` library (`github.com/rwcarlsen/goexif`) is used for the read/check path; XMP segment manipulation is handled directly on the raw JPEG bytes, since `goexif` is read-only.

## License

MIT
