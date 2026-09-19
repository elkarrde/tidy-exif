# Status

*Last updated: 2026-09-20*

| Field | Value |
|:--|:--|
| Phase | released (v0.1.3); `main` has moved on since |
| Version | v0.1.3 released; `main` is 5 commits ahead and still reports `0.1.3` |
| Build | passing |
| Tests | passing |
| Deployed | released 2026-06-20 — [v0.1.3 on Codeberg](https://codeberg.org/elkarrde/tidy-exif/releases/tag/v0.1.3) with all three archives attached; the site is live at <https://iso3200.org/tidy-exif/> |
| Blocker | none for v0.1.3. The next release needs a version bump in `cmd/tidy-exif/main.go` — `main` carries the exifscalpel migration, unreleased |

## Notes

The `check` and `clean` commands are implemented; `go build`, the Windows
cross-compile, `go test`, and `go vet` all pass. Real-JPEG smoke test **done**
(2026-06-19) against 20 Lightroom/Photoshop exports — see below.

Smoke test surfaced and fixed two bugs: history `stEvt:softwareAgent` is written
in **attribute** form by Lightroom/Photoshop, but the parser/cleaner only handled
the element form, so Adobe history agents were silently missed. Also **extended**
the tool to clean the Exif IFD0 Software tag (0x0131), gated to Adobe-only values
so camera/scanner Software (e.g. VueScan) is preserved. Verified end-to-end:
length-preserving, output decodes as valid JPEG, all Adobe signatures removed.

Note: the planned `goexif` dependency was dropped — the tool is hand-rolled over
raw JPEG bytes with only `BurntSushi/toml` as a third-party dep. Only release
tasks (tag + binaries) remain.

Repo reorganized (2026-06-20) to a lapis-style layout: CLI in `cmd/tidy-exif/`
(`package main`), metadata engine in `internal/meta/` (`package meta`) — the latter
is what will extract into the `exifscalpel` library. Build/vet/tests green after the
move; behavior unchanged.

Added `make dist` (2026-06-20): builds all targets and packages MPL-2.0-compliant
release archives in `dist/` — `tidy-exif-<version>-linux-amd64.tar.gz`,
`tidy-exif-<version>-linux-arm64.tar.gz`, and
`tidy-exif-<version>-windows-amd64.zip`, each bundling the binary plus `LICENSE`
and `README.md`. Version is read from `cmd/tidy-exif/main.go`.

Added 64-bit ARM Linux target (2026-06-20): `make build-arm64`
(`GOOS=linux GOARCH=arm64`), now also packaged by `make dist`. Pure Go / no cgo,
so it cross-compiles to a static binary from the amd64 host — verified building
clean, no need to compile on the Pine64.

Released `v0.1.3` on 2026-06-20: tag pushed and all three `dist/` archives
(linux-amd64, linux-arm64, windows-amd64) attached to the Codeberg release. The
`tidy-exif-web` site went live at <https://iso3200.org/tidy-exif/> and serves
those download links.

**Unreleased work on `main` (2026-09-20).** Five commits since the tag moved the
metadata engine onto the `exifscalpel` library: `internal/meta/xmp.go` and
`jpeg.go` are gone and `exif.go` is largely gutted, with `inspect.go` now driving
the shared primitives (net −874/+309 lines). Build, `go test ./...`, and `go vet`
are all green on that state, but the `version` constant in
`cmd/tidy-exif/main.go` is still `0.1.3`, so the next release must bump it. Note
`go.mod` still pins `exifscalpel v0.1.0` while the library is at v0.3.1 — worth
re-vendoring before cutting the next tag.

`TODO.md` remains stale: every box is unchecked although phases 1–8 shipped.
