<p align="center">
  <img src="assets/banner.svg" alt="Luxury Optimization" width="100%">
</p>

<p align="center">
  <a href="https://github.com/GofMan5/Luxury-Optimization/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/GofMan5/Luxury-Optimization/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/GofMan5/Luxury-Optimization/releases"><img alt="Release" src="https://img.shields.io/github/v/release/GofMan5/Luxury-Optimization?display_name=tag&sort=semver"></a>
  <a href="LICENSE"><img alt="MIT License" src="https://img.shields.io/badge/license-MIT-d7b665"></a>
  <a href="go.mod"><img alt="Go 1.25" src="https://img.shields.io/badge/Go-1.25-5c8dbc"></a>
</p>

Luxury Optimization is an open-source gaming and system optimization toolkit for Windows and Linux. It favors changes that can be measured, bounded, verified and reversed. Unsupported settings are reported and skipped instead of being guessed.

It does not disable Defender, Firewall, mitigations, Windows Update or core services. It does not ship memory cleaners, timer daemons, private GPU registry values, fixed IRQ masks or third-party binaries.

## Platform coverage

| Capability | Windows | Linux |
|---|---|---|
| System and hardware audit | WMI/CIM, power, profile drift | `/etc/os-release`, `/proc`, `/sys`, capability report |
| Persistent optimization profile | Reversible registry, mouse, power and supported Ethernet settings | Explicit safe no-op; Windows settings are not emulated |
| Game boost | Temporary profile, non-admin game process, automatic restore | Feral GameMode when present, direct launch otherwise |
| Process tuning | Explicit normal/above-normal/high priority and affinity | Explicit nice/affinity; missing `CAP_SYS_NICE` is skipped |
| Game discovery | Steam, Epic and fixed-drive Xbox | Steam, native and Flatpak roots |
| Saved game profiles | Atomic user file with validation and lock | Atomic XDG file with validation and `flock` |
| Startup manager | Reversible HKCU Run values; HKLM read-only | Reversible user XDG `.desktop` entries; system entries read-only |
| Service inventory | Read-only Windows SCM | Read-only systemd; empty non-fatal report on other init systems |
| Network and benchmark tools | Native interfaces, TCP latency, median/MAD comparison | Same portable implementation |
| Self-update | GitHub Releases, exact architecture, SHA-256, deferred replacement | GitHub Releases, exact architecture, SHA-256, atomic replacement |

Supported release targets: Windows `amd64`, `arm64`, `386`; Linux `amd64`, `arm64`.

## Get the release

Download the matching binary and `SHA256SUMS.txt` from [GitHub Releases](https://github.com/GofMan5/Luxury-Optimization/releases/latest).

Windows PowerShell:

```powershell
curl.exe -fLO https://github.com/GofMan5/Luxury-Optimization/releases/latest/download/Luxury-Optimization-windows-amd64.exe
curl.exe -fLO https://github.com/GofMan5/Luxury-Optimization/releases/latest/download/SHA256SUMS.txt
$expected = ((Select-String -LiteralPath .\SHA256SUMS.txt -Pattern 'Luxury-Optimization-windows-amd64.exe$').Line -split '\s+')[0]
$actual = (Get-FileHash .\Luxury-Optimization-windows-amd64.exe -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw 'SHA-256 mismatch' }
.\Luxury-Optimization-windows-amd64.exe audit
```

Linux:

```sh
curl -fLO https://github.com/GofMan5/Luxury-Optimization/releases/latest/download/Luxury-Optimization-linux-amd64
curl -fLO https://github.com/GofMan5/Luxury-Optimization/releases/latest/download/SHA256SUMS.txt
grep 'Luxury-Optimization-linux-amd64$' SHA256SUMS.txt | sha256sum -c -
chmod +x Luxury-Optimization-linux-amd64
./Luxury-Optimization-linux-amd64 audit
```

Do not run a Linux game as root. On Windows, the game itself remains non-elevated; only the bounded system-profile operation crosses UAC.

## Core workflow

```text
audit → plan → backup → apply → read-back verify → rollback on failure
```

`audit` and `plan` are read-only. On Windows, every persistent target is journaled before mutation and checked after apply and restore. On Linux, persistent Windows-only targets return success as `skipped`; session tools continue with the capabilities that actually exist.

Windows examples:

```powershell
.\Luxury-Optimization-windows-amd64.exe plan --profile recommended
.\Luxury-Optimization-windows-amd64.exe apply --profile recommended
.\Luxury-Optimization-windows-amd64.exe boost --game "C:\Games\Game\Game.exe" --profile maximum --priority above-normal -- -windowed
.\Luxury-Optimization-windows-amd64.exe games scan --json
.\Luxury-Optimization-windows-amd64.exe startup list --json
.\Luxury-Optimization-windows-amd64.exe services list --state running
.\Luxury-Optimization-windows-amd64.exe backups list --json
.\Luxury-Optimization-windows-amd64.exe restore --id 20260801T010203.123456789Z
```

Linux examples:

```sh
./Luxury-Optimization-linux-amd64 plan --profile maximum
./Luxury-Optimization-linux-amd64 boost --game /opt/game/game --priority above-normal --affinity 0x0f
./Luxury-Optimization-linux-amd64 games scan --json
./Luxury-Optimization-linux-amd64 startup disable --name launcher.desktop
./Luxury-Optimization-linux-amd64 services list --state running --json
```

Portable measurement tools:

```sh
./Luxury-Optimization-linux-amd64 network interfaces --json
./Luxury-Optimization-linux-amd64 network test --address 1.1.1.1:443 --count 10
./Luxury-Optimization-linux-amd64 benchmark template > before.json
./Luxury-Optimization-linux-amd64 benchmark compare --before before.json --after after.json
```

Run at least three identical before/after passes. The comparison uses medians and median absolute deviation; it reports a gain only when the change exceeds observed run-to-run noise.

## Updates

The updater is transparent and opt-in:

```sh
./Luxury-Optimization-linux-amd64 update check
./Luxury-Optimization-linux-amd64 update install
./Luxury-Optimization-linux-amd64 update enable
./Luxury-Optimization-linux-amd64 update status
./Luxury-Optimization-linux-amd64 update disable
```

Once enabled, the program checks at most once per 24 hours. It accepts only the `1.0.x` release line, requires trusted HTTPS hosts, selects the exact OS/architecture asset and verifies it against `SHA256SUMS.txt`. A failed or unavailable check never blocks the requested command.

## Profiles and boundaries

The recommended Windows profile enables Game Mode, disables unused Game DVR capture, disables mouse acceleration through registry and live API state, and reduces purely visual UI work.

The maximum profile adds a separate reversible AC power plan with a 5–100% CPU range, capability-checked EPP/boost settings, PCIe/USB power-saving changes and only Ethernet properties the physical adapter explicitly exposes. It can increase heat and power use.

Linux intentionally keeps system-wide tuning in the distribution and kernel policy layer. `boost` uses GameMode when installed and otherwise launches normally; explicit nice/affinity requests degrade safely when the kernel or permissions do not allow them.

See [tweak decisions](docs/NOTES.md), [BoosterX coverage](docs/BOOSTERX-COVERAGE.md), [roadmap](docs/ROADMAP.md), [changelog](docs/CHANGELOG.md) and [review evidence](docs/REVIEW-15.md).

## Build from source

Go 1.25.12 is the reference toolchain.

```powershell
.\build.ps1 -Version 1.0.0
```

```sh
chmod +x build.sh
./build.sh 1.0.0
```

Both scripts verify modules, run tests and vet, cross-build all five targets, write `dist/SHA256SUMS.txt`, verify published hashes and preserve the previous generated set if publication fails.

## Open source

Licensed under [MIT](LICENSE). Contributions are welcome under [CONTRIBUTING.md](CONTRIBUTING.md). Security reports use the private process in [SECURITY.md](SECURITY.md). Third-party attributions are in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Optimization is workload-specific. Keep before/after evidence and a rollback path; more tweaks do not automatically mean more performance.
