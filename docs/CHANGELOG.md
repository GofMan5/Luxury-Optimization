# Changelog

## 1.0.1 - 2026-08-04

### Tweaks

- Added compact catalog sorting by name, risk and required action.
- Added manual apply and one-click rollback inside each expanded tweak row.
- Every manual Windows tweak now creates its own sealed backup before mutation and keeps a persistent quick-rollback marker.
- Made Ethernet tweak IDs adapter-specific so two NICs can never share an action or rollback target.
- Made single mouse tweaks update and restore only their selected live SPI field.
- Made single power tweaks clone the active plan and change one supported value; out-of-order power rollback fails safely.
- Marked Linux session capabilities as non-persistent and unavailable for manual system mutation.

### Reliability

- Serialized per-tweak UAC transactions, extended desktop mutation timeouts and added the new methods to both Go and Tauri allowlists.
- Added per-tweak backup-shape validation, power-setting allowlists and focused Go/TypeScript tests.
- Fixed the Linux build by providing the missing process-local startup transaction lock.

### Desktop updates and release

- Switched the GUI from the raw sidecar updater to the official signed Tauri v2 updater with automatic startup checks and explicit installation.
- Removed sidecar binary replacement from the WebView method allowlist.
- Added Windows/Linux Tauri release jobs, updater artifacts, `latest.json`, frontend/Rust CI and Dependabot coverage for npm and Cargo.
- Repaired root build scripts and CI paths after the `frontend/` and `backend/` split.
- Updated transitive `plist`, `quick-xml` and `time` dependencies to remove three RustSec denial-of-service findings; CI now runs RustSec audit.

## 1.0.0 - 2026-08-03

### Project

- Renamed the public project to Luxury Optimization and reset the release line for the new cross-platform product.
- Added the MIT license, contribution/security/support policies, issue forms, PR template, code ownership and third-party notices.
- Replaced the project README with a verified SVG banner, platform matrix, safety contract and complete release instructions.

### Linux

- Added native Linux audit using `/etc/os-release`, `/proc` and `/sys` with explicit capability reporting.
- Added Steam discovery for native and Flatpak roots, bounded executable scanning and atomic XDG game profiles.
- Added session-only game boost with Feral GameMode when installed, direct fallback otherwise, explicit nice/affinity and read-back.
- Added reversible user XDG startup management and read-only systemd service inventory.
- Persistent Windows-only profiles, restore and backups are successful explicit no-ops instead of partial emulation.

### Windows

- Preserved the existing reversible registry, mouse, power, Ethernet, startup, game profile and sealed backup contracts.
- Preserved legacy state, registry and mutex identifiers so pre-rename backups remain restorable and old/new processes still coordinate.
- Rebranded user-visible output, temporary UAC result files and newly created performance power plans.

### Updates and releases

- Added opt-in GitHub Release updates with a pinned `1.0.x` channel, exact OS/architecture selection, bounded HTTPS downloads and SHA-256 verification.
- Added safe atomic Linux replacement and deferred transactional Windows replacement after process exit.
- Added Windows and Linux build scripts for Windows `amd64/arm64/386` and Linux `amd64/arm64`, plus `SHA256SUMS.txt`.
- Added native Windows/Linux CI, race/vet/staticcheck/govulncheck gates and a tag-driven public release workflow.

### Review

- Re-ran the full project review and review-of-review across compatibility, security, update supply chain, release reproducibility and existing Windows rollback behavior.

## Pre-1.0 lineage

Luxury Optimization 1.0.0 is the open-source cross-platform continuation of the Windows-only GofMan3 Optimization Pack 3.2.0. The old version numbers are not part of the new release line; compatible Windows state locations remain intentionally unchanged.
