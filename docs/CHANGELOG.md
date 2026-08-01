# Changelog

## 3.2.0 - 2026-08-01

### Added

- Steam, Epic and fixed-drive Xbox game discovery with bounded EXE scanning and PE/MZ validation.
- Atomic per-game profiles: game path, reversible system profile, process priority, optional affinity and launch arguments.
- Explicit process-scoped `normal`, `above-normal` and `high` priority with native read-back; realtime is rejected.
- Reversible HKCU startup management with exact `REG_SZ` / `REG_EXPAND_SZ` backup and read-back.
- Startup-load finding in the read-only audit and a full BoosterX capability/tweak decision matrix.
- Read-only Windows SCM inventory with state/start-type filters; no service mutation.
- Network interface inventory and bounded TCP connect latency/median/p95/jitter measurement.
- Three-or-more-run benchmark comparison using median FPS, 1% low, p95 frametime and MAD-based noise thresholds.
- Admin-only sealed backup inventory and exact `restore --id` selection through the existing UAC parent binding.

### Improved

- The maximum power plan now sets a 5% minimum and 100% maximum CPU state instead of inheriting a permanent 100% minimum from Ultimate/High Performance.
- Fixed ACL publication for UAC result files and user profile files by reopening with `READ_CONTROL | WRITE_DAC`, validating file identity and setting the DACL on that handle.
- Game EXE discovery ranks likely launch targets and skips redistributables, installers, crash reporters, helpers and test artifacts.

### Kept out

- BoosterX-style Defender/VBS/mitigation disabling, BCD/HPET recipes, mass service/device changes, private GPU values, memory cleaners and placebo tweaks.

### Review

- Completed 15 independent whole-project review passes plus a review-of-review; findings and closure evidence are recorded in `docs/REVIEW-15.md`.

## 3.1.0 - 2026-08-01

### Added

- Temporary `boost` sessions apply a selected profile, launch one validated game EXE without elevation, monitor it and restore the exact backup afterwards.
- Absolute-path, regular-file and PE/MZ validation before game launch; arguments are passed directly without a shell.
- A session mutex blocks ordinary `apply` and `restore` while a game boost is active.

### Reliability

- Restore is registered before game launch and runs after normal exit, launch failure, non-zero game exit or Ctrl+C.
- The elevated internal bypass requires an elevated child, a verified same-executable parent PID and a live boost-session mutex.
- Added regression coverage for the mutex boundary and rejected game paths.

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
