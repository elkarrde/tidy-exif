# FUTURE — tidy-exif

Exploratory / not-yet-scheduled work. Captured for later; nothing here is committed
to a release.

---

## Windows: right-click (Explorer context menu) integration

**Goal:** let Windows users run tidy-exif by right-clicking a JPEG (or a folder)
in Explorer — the platform-native idiom — instead of using the CLI. Linux/macOS
stay CLI; the friction is specifically a Windows-user expectation.

### How it works (the mechanism)

A right-click action is a *shell verb* registered in the registry; Explorer just
launches the exe with the file/folder path. No core logic changes — the existing
subcommands map directly:

- **File verb** → `tidy-exif.exe clean "%1" --backup` (and/or a `check "%1"` entry)
- **Folder verb** → `tidy-exif.exe clean --dir "%1"` — natural fit for batch.

Registry location (per-user, no admin):
```
HKCU\Software\Classes\SystemFileAssociations\.jpg\shell\TidyExifClean\command
   (Default) = "C:\Program Files\tidy-exif\tidy-exif.exe" clean "%1" --backup
```
`HKLM`/`HKCR` is machine-wide but needs elevation. Add an icon, a friendly label,
and optionally gate behind Shift+right-click (`Extended` value). Repeat for `.jpeg`.

### Three constraints that shape the work

1. **Windows 11 menu split.** Classic registry verbs are demoted to "Show more
   options" (legacy menu). The *main* Win11 menu requires a packaged app (MSIX)
   implementing `IExplorerCommand` — a COM shell extension (not Go; C++/C#/Rust
   that shells out to the Go exe).
2. **Console window.** A console exe launched from Explorer pops a black window;
   the current `waitIfWindows` "Press Enter to close" pause makes it linger. For a
   native feel, build a GUI-subsystem variant (`-ldflags -H=windowsgui`) and report
   via MessageBox/toast instead of stdout. This is the main CLI-vs-app design fork.
3. **Multi-select.** Static verbs launch the exe once *per selected file* (50 files
   → 50 processes/toasts). Single-process multi-file handling needs a real COM
   `IContextMenu`/`IExplorerCommand` extension. Pragmatic dodge: use the folder verb
   for batch; keep the file verb for one-offs.

### Distribution reality

The bare `.exe` is not enough for non-technical users — they expect an installer
that places the exe and writes the registry keys. Low-friction: **Inno Setup** or
**NSIS** (both write verbs + uninstaller easily). Avoiding SmartScreen "unknown
publisher" warnings (and MSIX) effectively needs an **Authenticode code-signing
certificate** — an annual cost to budget for.

### Tiers of effort

#### Tier 1 — Registry verbs + Inno Setup installer  *(low effort)*
- Installer places `tidy-exif.exe` and writes per-user `SystemFileAssociations`
  verbs for `.jpg`/`.jpeg` (file: check + clean-with-backup) and a folder verb for
  batch. Icon included; uninstaller removes keys.
- Lives under "Show more options" on Win11. Console window still flashes unless the
  GUI build (Tier 2) is also done. **No new application code** beyond the installer
  script.

#### Tier 2 — + GUI-subsystem build + toast/dialog feedback  *(medium effort)*
- Build a windowless variant (`-H=windowsgui`); replace console output with a
  MessageBox or toast ("Cleaned 3 files, 1 skipped"). No black window.
- Folder verb covers batch cleanly; sensible silent defaults (e.g. `--backup`).
- Still the legacy menu on Win11 (acceptable).

#### Tier 3 — MSIX + `IExplorerCommand` shell extension + signing  *(high effort)*
- Native COM shell extension (C++/C#/Rust front-end shelling out to the Go engine)
  packaged as MSIX and code-signed.
- Gets top-level Win11 context-menu placement and proper multi-select handling in a
  single process. Submenus / dynamic enable-disable become possible.
- Only worth it if Win11 primary-menu placement or large multi-selects become real
  user complaints.

### Recommendation

- **Tier 2 is the correct level of effort** — it removes the console-flash, the
  single biggest UX wart, and delivers a result that feels like a real Windows app.
- **Tier 1 is the realistic near-term goal — do this soon.** It captures ~80% of
  the perceived value (it's in the right-click menu, an installer sets it up) for
  very little work and zero changes to the app itself. Ship Tier 1, then layer
  Tier 2's GUI build on top.
- **Tier 3 is deferred** unless specific complaints justify it.

### Synergy with exifscalpel

This pairs naturally with the `exifscalpel` library extraction (now its own repo
at `../exifscalpel/`; see its `exifscalpel-HANDOFF.md`): once the core logic is a
library, a CLI exe and a thin
Windows GUI/shell wrapper become equally cheap front-ends over the same engine —
making Tier 2 (and even Tier 3's exe-behind-a-shell-ext) materially easier.

---

## Report film-scan (AnalogExif) metadata via `xmp.ReadProperties`

**Goal:** surface the structured film-photography metadata that scanning tools
(ExifNotes / AnalogExif) write into a JPEG's XMP packet — film stock, developer,
lens, scanner — in tidy-exif's `check` output. This is *read-only reporting*, not
scrubbing: these fields are worth showing, not removing. The `clean` path is
untouched.

### Why

AnalogExif writes a cleaner, structured copy of these fields to XMP than to the EXIF
`UserComment` text blob — and some (lens, scanner, scanner software) appear *only*
in XMP, not in the EXIF block at all. exifscalpel **v0.3.1** added
`xmp.ReadProperties` precisely for this: read arbitrary scalar XMP properties by
namespace URI + local name (prefix-independent), with no built-in vocabulary — the
*policy* (which namespaces/fields count as "film metadata") stays here in tidy-exif.

### Fields (the tidy-exif policy)

Namespaces:
- AnalogExif — `http://analogexif.sourceforge.net/ns/`
- aux (Adobe) — `http://ns.adobe.com/exif/1.0/aux/`

Fields seen on real scans (`../masterdata/testdata/Scan-*.jpg`):
`AnalogExif:Film`, `FilmMaker`, `FilmType`, `FilmAlias`, `DevelopProcess`,
`Developer`, `DeveloperMaker`, `DeveloperDilution`, `DevelopTime`, `ExposureNumber`,
`Scanner`, `ScannerMaker`, `ScannerSoftware`, plus `aux:Lens`. Start with a focused
subset (`Film`, `FilmMaker`, `Developer`, `Lens`, `Scanner`) and grow as useful.

### Where it plugs in

- `internal/meta/inspect.go` — add a film field to `FileReport` (e.g.
  `Film map[xmp.Property]string`, or a small typed struct). In `InspectJPEG`, on the
  XMP segment (alongside the existing `xmp.Parse`), call
  `xmp.ReadProperties(seg.Data, filmWant)` where `filmWant` is a package-level
  `[]xmp.Property` built from the list above. Read-only: it must **not** affect
  `HasAdobeData()` or the clean path — film metadata is preserved, not Adobe-signature
  scrub scope.
- `cmd/tidy-exif/check.go` — display the film fields. The current table (`#`,
  Filename, CreatorTool, IDs, Hist, EXIF Software) is already wide, so prefer either
  a per-file detail line printed under the row when film metadata is present, or a
  dedicated `--film` / `-f` flag that switches to a film-oriented view. Don't widen
  the default table.

### Notes / constraints

- Depends on exifscalpel **v0.3.1** (published): `go get
  codeberg.org/elkarrde/exifscalpel@v0.3.1` — no new transitive deps (still one
  direct dep + BurntSushi/toml).
- `ReadProperties` returns only *simple scalar* properties; AnalogExif writes its
  fields as scalars on `rdf:Description`, so this fits. Localized/array/struct XMP is
  out of scope (and not used here).
- Its error contract is best-effort: a non-XMP payload errors, but a
  malformed/truncated packet is not reported (whatever parsed is returned). Treat an
  absent field and a parse fault the same — just "not shown."
- Tests: mirror a minimal AnalogExif packet as a programmatic byte-fixture (per the
  no-real-photos rule); use the `Scan-*` files in `../masterdata/testdata` only for
  manual spot-checks.
- Kept separate from cleaning on purpose: this is *preserved* content the user likely
  wants to see; tidy-exif only ever targets Adobe-signature fields.
