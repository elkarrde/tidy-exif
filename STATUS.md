# Status

*Last updated: 2026-06-19*

| Field | Value |
|:--|:--|
| Phase | feature-complete |
| Version | untagged (v0.1.0 in code) |
| Build | passing |
| Tests | passing |
| Deployed | not released |
| Blocker | tag `v0.1.0`, build + attach Linux/Windows binaries |

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
