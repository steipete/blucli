# Changelog

## Unreleased

## 0.1.9 (2026-10-23)

**Highlights:** Spotify login and requests fail safely, and saved configuration and tokens use user-only permissions.

### Changes

- Security: save configuration and Spotify tokens with user-only file permissions, including updates to existing files.
- Spotify login rejects a redirect URL that has no path, instead of panicking. Thanks @SebTardif.
- Spotify HTTP calls use the global `--timeout`, so a peer that accepts and then stalls cannot hang the process. Thanks @SebTardif.

## 0.1.8 (2026-10-20)

### Highlights

- Fix certificate validation vulnerability GO-2026-6809 in release binaries.

### Changes

- Security: build with Go 1.27.3 to fix certificate validation vulnerability GO-2026-6809.
- Breaking: macOS release binaries now require macOS 13 Ventura or newer, matching Go 1.27's supported minimum.
- CI: move macOS jobs and release verification to `xcode-27` and smoke-test Intel binaries with Rosetta.
- Security: build with Go 1.27.2 to fix standard-library vulnerabilities while retaining Go 1.25 source compatibility.
- Release: update the shared workflow pin and expose a non-publishing dry run.

## 0.1.7 (2026-09-14)

### Highlights

- macOS downloads are signed and notarized.

### Changes

- Release binaries for macOS are now Developer ID signed and notarized, so direct downloads pass Gatekeeper.

## 0.1.6 (2026-09-13)

- Build: use Go 1.26.8 for CI, releases, and Docker while retaining Go 1.25 source compatibility and macOS 12 support.
- Docker: update the runtime base to Alpine 3.24.
- Dependencies: update Go networking/tooling modules and pnpm.
- CI: update checkout, Go setup, and GoReleaser actions to their current major releases.
- Tooling: update pnpm to 12.2.1 and the Dockerfile frontend to 1.27.

## 0.1.5 (2026-06-11)

- Docker: honor the `VERSION` build argument in `blu --version` and cover it in CI.
- Release: standardize archive names for target-specific Homebrew and aqua installs.
- Security: update Go dependencies and pin the Go toolchain to 1.25.11.

## 0.1.4 (2026-05-15)

- Docker: add a local image with `/data` persistence and CI smoke coverage.
- Release: add Linux arm64 artifacts, refresh GoReleaser v2 archive config, and update the Homebrew tap workflow.
- CI: pin Go toolchain to 1.25.10 to clear current standard-library `govulncheck` advisories.
- CI: update golangci-lint workflow and config for v2.
- Tooling: update Go dependencies and pnpm metadata.
- Project metadata: update copyright year to 2026.

## 0.1.3 (2025-12-17)

- Device selection: accept discovery names for `--device`/`BLU_DEVICE` (no config aliases needed).

## 0.1.2 (2025-12-17)

- Packaging fix: stop ignoring `cmd/blu` in `.gitignore` (needed for `go install` + releases).

## 0.1.1 (2025-12-17)

- Release packaging: GoReleaser archive naming + Homebrew-friendly assets.

## 0.1.0 (2025-12-17)

- Discovery: mDNS (`_musc/_musp/_musz/_mush`) + LSDP fallback; discovery cache.
- Device selection: `--device`, `BLU_DEVICE`, config `default_device`, aliases.
- Playback: `play/pause/stop/next/prev`, plus `play --url/--seek/--id`.
- Volume: `volume get|set|up|down`, `mute on|off|toggle`.
- Modes: `shuffle on|off`, `repeat off|track|queue`.
- Grouping: `group status|add|remove`.
- Queue: `queue list|clear|delete|move|save` (JSON includes queue metadata even when 0).
- Browsing: `browse`, `playlists`, `inputs` (Capture).
- Presets: `presets list|load`.
- TuneIn helper: `tunein search|play` (simple “play X” path without Spotify API).
- Spotify helper: `spotify open` (switch to Spotify Connect) + optional `spotify login/search/play` (Spotify Web API).
- Observability: `--json`, `--trace-http`, `--dry-run` (blocks mutating requests).
- Diagnostics: `diag`, `doctor`, plus `raw` endpoint runner.
- UX: `--help`, `version`, bash/zsh completions.
- Tooling: golangci-lint, govulncheck, GitHub Actions CI, GoReleaser release workflow, pnpm helper scripts.
