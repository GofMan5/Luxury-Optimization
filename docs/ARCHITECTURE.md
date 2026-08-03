# Architecture

Luxury Optimization is one Go module with a shared CLI core and build-selected platform files.

## Shared

- `brand.go`: product, release channel and repository identity.
- `types.go`: stable audit, plan, capability and Windows backup data contracts.
- `network.go`, `benchmark.go`: portable measurement tools.
- `update.go`: GitHub Release discovery, version/channel policy, URL and size boundaries, checksum verification and config.

## Windows

Windows files retain the mature transaction: plan → sealed backup → apply → native read-back → rollback. The game process never inherits elevation. Legacy state, registry and mutex names remain unchanged so old backups can be restored and old/new binaries cannot mutate concurrently.

## Linux

Linux files use `/etc/os-release`, `/proc`, `/sys`, XDG paths, `systemctl`, `flock`, `setpriority` and `sched_setaffinity`. There is no fake registry or universal governor mutation. Missing GameMode, systemd or `CAP_SYS_NICE` produces a capability result and a safe fallback.

## Release boundary

Build scripts produce five raw binaries and one checksum manifest. The updater accepts only the configured release line and exact target filename; platform installers receive bytes only after metadata, host, size and SHA-256 checks pass.
