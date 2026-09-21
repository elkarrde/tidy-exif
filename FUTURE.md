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
  single process. Dynamic enable-disable becomes possible. (Static submenus do
  *not* need this tier — see the shared cascade below.)
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

### Shared "EXIF…" cascade with sibling tools  *(plan, 2026-09-22)*

**Idea:** tidy-exif is one of several small EXIF/XMP tools with the same
right-click workflow (`../contextexif/` already ships it). Instead of each tool
adding its own top-level entry, group them under one cascading menu item —
**EXIF… → Tidy, Lapis, Show, Complete** (names are working titles) — with every
submenu item launching its own separate executable.

**Verdict: sound.** Windows supports static cascades natively, per-user, with the
same registry mechanism contextexif already uses (Tier 1 level — no COM, no MSIX):

```
HKCU\Software\Classes\SystemFileAssociations\.jpg\shell\exiftools
    MUIVerb     = "EXIF…"          ← parent label
    SubCommands = ""               ← marks it as a cascade
    Icon        = …
    shell\
        tidy\      (default)="Tidy"   Icon=…   command\ = "…\tidy-exif.exe" …
        lapis\     (default)="Lapis"  …        command\ = "…\lapis.exe" "%1"
        show\      (default)="Show"   …        command\ = "…\contextexif.exe" "%1"
        complete\  …
```

Each child verb has its own command and icon, so "one exe per item" is the natural
shape. Repeat under `.jpeg` (and `Directory\shell` for folder-capable tools — the
set of items may differ per file type).

**Why separate exes (not one launcher):** independent release cycles, sizes, and
dependencies per tool; a bug in one can't break the others; every tool stays usable
as a standalone CLI. The alternative — a single `exif.exe` with subcommands — gets
one installer and no ownership problem, but couples all tools' releases. Since the
tools already live in separate repos, keep separate exes.

#### Plan

1. **Prototype tidy-exif's right-click invocation first** (riskiest part — see
   multi-select below). Tidy's CLI is `--dir`-based today; it needs to accept file
   paths as positional arguments, and the multi-select behavior must be verified on
   real Windows for the chosen `MultiSelectModel`, including the >15-item case
   (Explorer hides verbs on large selections unless a model is set).
2. **Extract a shared `shellmenu` Go package** from contextexif's
   `internal/shellmenu` (~160 lines excl. tests; `keys.go` is pure string
   builders, `register_windows.go` the syscalls). Generalize it to register a child
   verb under the common parent. Every tool imports it, so the parent key name,
   label, and add/remove rules live in one place.
3. **Parent-key ownership rules** (in that package): install = create the parent if
   missing, then add own child (idempotent); uninstall = remove own child, then
   delete the parent only if no children remain. Without this, uninstalling one tool
   either wipes the menu for all or leaves an empty "EXIF…" entry.
4. **Migrate contextexif** from its top-level "Copy image metadata" verb to a child
   of the cascade (its uninstaller must also clean up the old top-level key).
5. **Add tidy-exif** as a child: file items for `.jpg`/`.jpeg`, plus a folder item
   under `Directory\shell` for batch. Pairs with Tier 2 (GUI build) so no console
   window flashes.
6. Other tools (Lapis, Complete, …) join by importing the package — no changes to
   the existing ones.

#### Open questions / constraints

- **Selection contracts differ.** contextexif is deliberately single-file (rejects
  multi-select with a dialog); Tidy wants many files or a folder. Static verbs'
  multi-select handling is limited (see constraint 3 above) — the folder verb is
  the pragmatic batch path.
- **Windows 11:** the cascade lives under "Show more options", same as any registry
  verb (contextexif already does). If Tier 3 ever happens, grouping helps: one
  entry to port to `IExplorerCommand` instead of four.
- **Installer:** separate per-tool installers each need the shared rules above; a
  combined "EXIF tools" installer is possible later but not required.
- Final menu label and item names are still open (`EXIF…` is a placeholder).

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
