# Status

*Last updated: 2026-06-20*

| Field | Value |
|:--|:--|
| Phase | feature-complete (phases 1–8 + Exif done) |
| Version | v0.1.3 tagged (annotated, local — not pushed) |
| Build | passing |
| Tests | passing |
| Deployed | not released |
| Blocker | push tag `v0.1.3` to Codeberg + upload Linux/Windows archives to the release |

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

Tagged `v0.1.3` (annotated, local) and built the `dist/` archives (2026-06-20).
Remaining: `git push origin v0.1.3` and upload the two archives to the Codeberg
release.
