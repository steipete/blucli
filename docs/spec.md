# blu — BluOS CLI (spec)

Status: in progress.

## Goals

- Fast CLI to control BluOS players (HTTP+XML Custom Integration API).
- Zero setup default: auto-discover players via mDNS; pick device via `--device` / `BLU_DEVICE` / config.
- Core controls: status, playback, volume, grouping.
- Good defaults: timeouts, readable errors, JSON output for scripting.
- Strong hygiene: formatter + linter + CI + tests.

## Non-goals

- Fixed groups / surround / advanced settings flows (unless API docs stabilize).
- Auth for BluOS API (it’s unauthenticated on LAN).
- Full streaming-service parity (Spotify search/control requires separate OAuth + Web API; supported as an optional “bonus”).

## Progress checklist

- [x] Repo hygiene: `gofmt`/`golangci-lint`/CI (`.github/workflows/ci.yml`)
- [x] Discovery: mDNS types `musc/musp/musz/mush` + cache (`blu devices`)
- [x] Device selection precedence + aliases + `BLU_DEVICE`
- [x] Core playback: `play/pause/stop/next/prev`
- [x] Shuffle + repeat
- [x] Volume: get/set/up/down + mute on/off/toggle
- [x] Grouping: `group status|add|remove`
- [x] Watch: long-poll status/syncstatus
- [x] Queue: list/clear/delete/move/save (+ optional add)
- [x] Presets: list + load
- [x] Browse/search: `browse`, `playlists`, `radiobrowse` (inputs)
- [x] Sleep timer
- [x] Diagnostics: `diag`, `doctor`
- [x] Power-user: `raw` endpoint runner + `--dry-run`
- [x] Optional: Spotify Web API integration (`blu spotify …`) for search + “play <artist>”
- [x] Compare against BluShell/pyblu/BluOS Controller.app notes in spec
- [x] Real-device verification (no-breakage, minimal disruption)

## Protocol facts used

### Discovery (mDNS/Bonjour)

Observed in `/Applications/BluOS Controller.app` (Electron):

- Browse `_musc._tcp`, `_musp._tcp`, `_musz._tcp`, `_mush._tcp` on `local.`
- Use first IPv4 + advertised port as device endpoint.
- TXT record contains `version=...` (when present).

### BluOS Custom Integration API (HTTP)

- Default port: `11000`
- Requests: mostly `GET`
- Responses: XML

Endpoints used by `blu`:

- Status: `GET /Status` (optional long poll: `?timeout=seconds&etag=etag`)
- Group status: `GET /SyncStatus` (optional long poll: `?timeout=seconds&etag=etag`)
- Playback:
  - `GET /Play` (+ `seek`, `id`, `url`)
  - `GET /Pause` (+ `toggle=1`)
  - `GET /Stop`
  - `GET /Skip`
  - `GET /Back`
  - `GET /Shuffle?state=<0|1>`
  - `GET /Repeat?state=<0|1|2>`
- Volume:
  - `GET /Volume?level=<0..100>&tell_slaves=1`
  - `GET /Volume?db=<delta_db>&tell_slaves=1` (typical ±2)
  - `GET /Volume?mute=<0|1>&tell_slaves=1`
- Grouping:
  - `GET /AddSlave?slave=<ip>&port=<port>[&group=<name>]`
  - `GET /RemoveSlave?slave=<ip>&port=<port>`
  - `GET /Playlist[?start=<n>&end=<n>]`
  - `GET /Clear`
  - `GET /Delete?id=<n>`
  - `GET /Move?old=<n>&new=<n>`
  - `GET /Save?name=<playlist_name>`
  - `GET /Presets`
  - `GET /Preset?id=<n>`
  - `GET /Playlists[?service=<name>&category=<...>&expr=<...>]`
  - `GET /Browse?key=<key>[&q=<query>][&withContextMenuItems=1]`
  - `GET /RadioBrowse?service=Capture`
  - `GET /Sleep`

## CLI UX

### Global flags

- `--device <id|alias>`: `host[:port]` or alias from config.
- `--json`: JSON output (stable for scripting).
- `--timeout <dur>`: HTTP timeout.
- `--dry-run`: block mutating endpoints (still allows reads); use for safe verification.
- `--trace-http`: print `http: GET …` for each request.
- `--discover/--discover=false`: allow discovery fallback.
- `--discover-timeout <dur>`: discovery window.
- `--config <path>`: optional config override.

### Device selection precedence

1. `--device`
2. `BLU_DEVICE`
3. config `default_device`
4. discovery cache (only if exactly 1 device)
5. live discovery (only if exactly 1 device)

If multiple devices: error + ask user to pick `--device`.

### Commands (v0)

- `blu completions bash|zsh`
- `blu version`
- `blu devices`: discover + print devices; refreshes cache.
- `blu status`: current player status.
- `blu now`: condensed now-playing (alias for `status` human output)
- `blu watch status|sync`: long-poll and print changes
- `blu play|pause|stop|next|prev`: playback control.
- `blu shuffle on|off`
- `blu repeat off|track|queue`
- `blu volume get|set <0-100>|up|down`
- `blu mute on|off|toggle`
- `blu group status|add <slave> [--name <group>]|remove <slave>`
- `blu queue list|clear|delete <id>|move <old> <new>|save <name>`
- `blu presets list|load <id>`
- `blu browse --key <key> [--q <query>] [--context]`
- `blu playlists [--service <name>] [--category <cat>] [--expr <search>]`
- `blu inputs` (aka `radiobrowse Capture`)
- `blu tunein search|play [--pick <n>] [--id <id>] <query>`
- `blu spotify login|logout|open|devices|search|play`
- `blu sleep` (cycles sleep timer)
- `blu diag` / `blu doctor`
- `blu raw <path> [--param k=v ...] [--write]` (power tool; `--write` blocked by `--dry-run`)

## Config + cache

### Config file

Path: `$(userConfigDir)/blu/config.json` (macOS: `~/Library/Application Support/blu/config.json`)

Schema:

```json
{
  "default_device": "192.168.1.100:11000",
  "aliases": {
    "kitchen": "192.168.1.100:11000",
    "office": "192.168.1.120:11000"
  }
}
```

### Discovery cache

Path: `$(userCacheDir)/blu/discovery.json`

Contains `updated_at` + last discovered devices (id/host/port).

## Implementation layout

- `cmd/blu`: entrypoint
- `internal/app`: CLI parsing + command routing (testable `Run`)
- `internal/bluos`: HTTP client + XML models (typed; ignores unknown attrs)
- `internal/discovery`: mDNS discovery (zeroconf)
- `internal/config`: config + cache + device parsing
- `internal/output`: printer (human + JSON)

## Testing

- `internal/bluos`: httptest server asserts request URLs; parses XML fixtures.
- `internal/app`: run commands via `Run(ctx, args, stdout, stderr)`; assert output/exit codes.
- `internal/config`: config and device parsing tests.
- `internal/discovery`: unit tests for TXT parsing + entry conversion (no network).

## CI / quality gates

- `go test ./...` (+ race on linux)
- `golangci-lint` (includes gofmt/goimports/gofumpt checks)
- `govulncheck ./...`
- build matrix: linux/mac/windows (compile sanity)

## Reference comparison (notes)

### BluShell (PowerShell)

- Same core endpoints for playback/volume/group/queue.
- Handy XML examples + parameter naming conventions.

### pyblu (Python)

- Same core endpoints (`/Status`, `/SyncStatus`, `/Volume`, `/Play`…).
- Good reference for queue XML parsing + `/RadioBrowse?service=Capture` “inputs”.

### BluOS Controller.app (macOS)

- Confirms discovery details:
  - mDNS browse types `_musc._tcp`, `_musp._tcp`, `_musz._tcp`, `_mush._tcp`
  - LSDP UDP broadcast query on port `11430` (useful fallback when mDNS is flaky).

## Verification notes (2025-12-17)

- Device discovery: works on Peter LAN (LSDP + mDNS merge).
- Verified end-to-end against real players on Peter LAN (status/group/queue/presets/browse/inputs/diag/watch + dry-run tracing for mutating calls).
