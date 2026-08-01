# Changelog

## 3.0.0 - 2026-08-01

### Focus

- Product scope reduced to gaming and Windows performance optimization.
- Removed `repair-legacy`, Defender/Firewall/CSRSS/NVIDIA remediation and the legacy security audit.
- Removed global `PowerThrottlingOff`; maximum performance now uses bounded AC power-plan controls.
- Removed the undocumented `d4e98f31-...` power setting.
- Removed historical BAT, migration report, bundled third-party tools, drivers, mods and internal finding artifacts from the active project.

### Changed

- Audit now reports whether the recommended gaming profile differs from the current system.
- Removed unconditional reboot advice; applied state is verified live.
- Documentation now separates accepted, conditional and rejected tweaks.
- Release notes and engineering decisions live under `docs/`.

### Kept

- Reversible recommended and maximum profiles.
- Hardware-aware CPU/power/Ethernet capability checks.
- Exact backup, read-back verification and rollback.
- Transactional multi-architecture build publication.

## 2.2.1 - 2026-08-01

- Added transactional `dist` publication with rollback and post-publish hashes.
- Added live mouse-state backup/apply/verification.
- Added supported CPU EPP/Boost settings and native power read-back.
