<p align="center">
  <img src="assets/banner.svg" alt="Luxury Optimization" width="100%">
</p>

<p align="center">
  <a href="https://github.com/GofMan5/Luxury-Optimization/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/GofMan5/Luxury-Optimization/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/GofMan5/Luxury-Optimization/releases"><img alt="Release" src="https://img.shields.io/github/v/release/GofMan5/Luxury-Optimization?display_name=tag&sort=semver"></a>
  <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-d7b665"></a>
  <a href="backend/go.mod"><img alt="Go 1.25" src="https://img.shields.io/badge/Go-1.25-5c8dbc"></a>
  <a href="frontend/src-tauri/Cargo.toml"><img alt="Tauri 2" src="https://img.shields.io/badge/Tauri-2-d7b665"></a>
</p>

Luxury Optimization is an open-source desktop utility for measurable gaming and system optimization. It detects the current machine, shows the exact change and risk, and skips anything the platform or hardware does not support.

No debloat scripts, timer daemons, memory cleaners, private GPU keys or mass service disabling. Defender, Firewall, mitigations and Windows Update stay intact.

## What is included

- Compact catalog of 100+ native, capability-checked tweaks with search, risk filters and useful sorting.
- Manual apply and one-click rollback for every supported Windows tweak.
- Lite, Medium and Max presets with strict nested scopes and a complete preflight plan.
- Built-in before/after benchmark comparison with CapFrameX JSON and raw MangoHud CSV import.
- Per-game launch profiles with bounded history and explicitly attached benchmark evidence.
- Measured background-load advisor with exact startup/service correlation and protected system targets.
- Startup, system services, RFC 1035 UDP/bufferbloat and verified storage-path diagnostics in one System section.
- Standalone storage analyzer with bounded parallel scanning, cached drill-down, treemap and confirmed Recycle Bin deletion while the main UI stays usable.
- Windows System Restore plus sealed Luxury recovery files.
- Russian and English UI.
- Signed in-app updates from GitHub Releases.

## Safety model

Every persistent Windows change follows the same transaction:

```text
capture exact state → seal backup → journal intent → apply → read back
                                               └─ mismatch → rollback
```

Manual tweaks receive separate backups. Ethernet actions are bound to one physical adapter, mouse rollback touches only the selected live setting, and power tweaks work in a cloned plan instead of editing the user's original plan.

Every preset requires a recent matching local Luxury checkpoint. Unsupported settings stay unavailable or skipped; they are never approximated.

## Platform support

| Area | Windows | Linux |
|---|---|---|
| Desktop GUI | Tauri v2 | Tauri v2 |
| Audit | OS, CPU, GPU, power, profile drift | distro, kernel, CPU, GPU, capabilities |
| Persistent tweaks | Registry, mouse, cloned power plan, supported Ethernet properties | Not emulated; safely unavailable |
| Session optimization | Non-elevated game process with automatic restore | GameMode, nice and affinity when supported |
| Startup | Reversible current-user entries | Reversible user XDG entries |
| Services | Full inventory; critical services protected | Read-only systemd inventory |
| Diagnostics | TCP/UDP, loaded latency, native volumes and verified temporary path probe | Same protocol with native mount/statfs inventory |
| Recovery | Windows restore points and sealed per-operation backups | No fake Windows recovery surface |

The release workflow currently builds x86-64 Windows and Linux desktop bundles. The Go recovery CLI also cross-builds for Windows `amd64`, `arm64`, `386` and Linux `amd64`, `arm64`.

## Install

Download the installer or portable bundle for your platform from [GitHub Releases](https://github.com/GofMan5/Luxury-Optimization/releases/latest). The app checks for newer `1.0.x` releases automatically; installation remains a user decision.

On Windows, only a bounded mutation crosses UAC. Games and the desktop UI remain non-elevated.

## Build from source

Requirements: Go 1.25.12, Node.js 24, pnpm 10.26.2, Rust 1.88+ and the native [Tauri prerequisites](https://v2.tauri.app/start/prerequisites/) for the host platform.

```powershell
cd frontend
pnpm install --frozen-lockfile
pnpm tauri:build
```

The build script compiles the matching Go sidecar automatically. Fast verification:

```powershell
cd backend
go test -race ./...
go vet ./...

cd ../frontend
pnpm check
cargo test --locked --manifest-path src-tauri/Cargo.toml
cargo clippy --locked --all-targets --manifest-path src-tauri/Cargo.toml -- -D warnings
```

Recovery-CLI target matrix:

```powershell
.\build.ps1 -Version 1.0.6
```

```sh
./build.sh 1.0.6
```

## Repository layout

```text
frontend/                 React UI and Tauri v2 host
  src/features/           vertical user-facing slices
  src-tauri/              Rust sidecar boundary and desktop bundles
backend/                  Go sidecar and recovery CLI
  internal/features/      JSON protocol handlers
  internal/optimizer/     platform transactions and rollback
docs/                     decisions, architecture, changelog and review evidence
```

See [architecture](docs/ARCHITECTURE.md), [1.0.6 tweak review](docs/TWEAK-REVIEW-1.0.6.md), [1.0.5 advisor review](docs/BACKGROUND-ADVISOR-REVIEW-1.0.5.md), [1.0.6 diagnostics review](docs/DIAGNOSTICS-PROTOCOL-1.0.6.md), [tweak decisions](docs/NOTES.md), [BoosterX coverage](docs/BOOSTERX-COVERAGE.md), [changelog](docs/CHANGELOG.md) and [release policy](docs/RELEASES.md).

## Contributing

Luxury Optimization is MIT licensed. Read [CONTRIBUTING.md](CONTRIBUTING.md) before changing a persistent tweak: every new target needs a plausible measurable benefit, capability checks, exact backup, read-back and rollback. Security reports follow [SECURITY.md](SECURITY.md).
