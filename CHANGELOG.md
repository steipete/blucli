# Changelog

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
