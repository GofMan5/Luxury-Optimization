# Background advisor review for 1.0.5

## Scope

The advisor is a read-only diagnostic surface. It does not terminate processes, alter priority, disable startup entries or change service configuration. Any follow-up opens the existing explicit single-target screen with its own confirmation, protection and rollback contract.

## Measurement contract

| Signal | Source | Meaning |
|---|---|---|
| CPU | Two native process CPU counters divided by total logical CPU capacity | Share of whole-machine capacity during the actual sample interval; thresholds scale down on high-core systems to keep quarter-core/full-core activity visible |
| I/O | Delta of process read/write transfer bytes | Process I/O rate; not presented as storage-only on Windows |
| Working set | Second native snapshot | Context only; memory alone never produces an action recommendation |
| Threads | Second native snapshot | Context only |

The request accepts 500–5000 ms and the desktop uses 1500 ms. Output is capped at 64 ranked processes, eight startup links and eight service links per process. Correlation inventories stop at 4096 entries and warnings at 32. Result frames remain below the one-megabyte protocol limit.

## Correlation and advice

- Windows service links use the exact running service PID from SCM.
- Linux service links use the sampled process's systemd cgroup unit and the read-only systemd inventory.
- Startup links require the identity-verified full executable path as a bounded token in the current startup command; basename-only guesses are rejected.
- A current-user enabled startup link may yield `review_startup` only at measured medium/high activity.
- A manageable non-system Windows service may yield `review_service` only at measured medium/high activity.
- Any system, critical or read-only service yields `protected_service` regardless of activity.
- Everything else yields `observe`; there is no guessed disable list.

## Failure and compatibility boundaries

- PID creation time is checked before executable/cgroup enrichment, preventing PID reuse from changing correlation identity.
- Counter decreases, missing snapshots, inaccessible paths and unsupported service managers are skipped or reported without partial mutation.
- Windows snapshot buffers, Linux proc reads, process counts, result counts and displayed strings are bounded.
- The Windows native buffer is capped at 32 MiB; every structure offset, next-entry step and UTF-16 pointer range is checked before the audited `unsafe` view is used.
- The existing startup/service backup, registry, mutex and restore identifiers are unchanged.

## Verification evidence

- Pure Go tests cover ranking, thresholds, executable boundaries, startup advice and protected services.
- A live Windows sidecar pass measured hundreds of processes without warnings and produced only conservative exact-link advice.
- Three consecutive 500 ms Windows passes returned at most 64 processes in sub-19 KiB frames with no warnings; median end-to-end wall time was 1.25 s on the verification machine.
- A 5000 ms request cancelled through `system.cancel` in 109 ms and the sidecar exited cleanly.
- Browser preview covers startup, third-party service and protected-service outcomes, navigation, RU/EN and the 980 px layout.
- Release approval still requires Windows/Linux Go race/vet/staticcheck/vulnerability checks, frontend/Rust checks, five recovery-CLI targets, Tauri bundles, signed updater assets and rollback evidence.

## Review-of-review decisions

- Replaced fixed CPU percentages with topology-aware thresholds so a busy thread remains visible on high-core machines.
- Removed basename-only startup matching after review showed generic executables could create false links; full identity-verified paths are now required.
- Moved executable/cgroup enrichment after service inventory and gated third-party service advice on a verified executable identity.
- Capped correlation inventories and warnings after reviewing the one-megabyte result boundary.
- Cleared gosec integer-conversion findings with explicit range guards. The remaining five G103 reports are the audited, bounds-checked Windows native buffer views described above.
- Removed redundant result fields and reused `slices.Contains`; no speculative dependency or mutation API was added.
