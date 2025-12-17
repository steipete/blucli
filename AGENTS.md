# bluecli notes

## Prior art

- BluShell (PowerShell wrapper): `/Users/steipete/Projects/oss/blushell` (no LICENSE file in repo; treat as proprietary)
- pyblu (Python lib): `/Users/steipete/Projects/oss/pyblu` (MIT)

## BluOS Controller.app (macOS)

- Electron app; logic in `.../Contents/Resources/app.asar`
- Quick inspect: `npx -y asar extract "/Applications/BluOS Controller.app/Contents/Resources/app.asar" /tmp/bluos-controller-asar`
- Discovery: mDNS/Bonjour browse types `musc`, `musp`, `musz`, `mush`; uses first IPv4 + advertised port
- Discovery fallback: LSDP UDP broadcast on port `11430` (query/parse logic lives in extracted JS)
