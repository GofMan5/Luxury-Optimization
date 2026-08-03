# Changelog

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
