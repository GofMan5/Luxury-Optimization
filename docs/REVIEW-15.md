# Fifteen-pass project review

Date: 2026-08-03

Scope: complete Luxury Optimization 1.0.0 source tree, public files, build/release contract and generated target matrix.

## Passes

| # | Independent review lens | Result |
|---:|---|---|
| 1 | Product requirements, branding and `1.0.x` version policy | Product/module/artifact/TUI names aligned; legacy state names intentionally isolated |
| 2 | Ponytail whole-repo complexity and dependency audit | Portable network/benchmark/types moved to shared files; Linux release links only `x/sys/unix`; no dependency added |
| 3 | Go build constraints and OS/architecture separation | Windows `amd64/arm64/386` and Linux `amd64/arm64` compile from one module |
| 4 | Linux system detection and capability fallback | Ubuntu 24.04 WSL audit passed; missing GameMode and Windows-only settings are explicit non-fatal skips |
| 5 | Linux game launch, nice, affinity, signals and cpusets | Real `/usr/bin/sleep` boost passed; unavailable `CAP_SYS_NICE` warned without failure; affinity read-back passed |
| 6 | Steam discovery and saved game profiles | Traversal bounds, executable validation, atomic JSON, `flock`, ID/path/argument limits reviewed and tested |
| 7 | Startup and service inventory | XDG disable/enable round trip passed; system scope remains read-only; systemd uses absolute tool path, C locale and timeout |
| 8 | Network and benchmark statistics | Portable tests passed; worst-case network test duration is capped; median/MAD comparison retained |
| 9 | Update discovery and supply-chain boundary | Exact repository/channel/target, HTTPS host+port allowlist, response limits, asset size and SHA-256 tested |
| 10 | Update replacement, concurrency and recovery | Update lock, corrupt-config recovery and atomic Linux mode-preserving replacement tested; Windows replacement is deferred with rollback |
| 11 | Windows UAC, backup and legacy compatibility | Existing allowlisted PowerShell, parent binding, result-file identity, sealed backup and rollback tests stayed green |
| 12 | Filesystem deletion and transactional publication | Linux clean no longer recurses; shell staging uses `mktemp`; both builders preserve prior managed artifacts on failure |
| 13 | CLI parsing and terminal output | Existing positional-argument regression suite passed; untrusted display strings now strip ANSI/bidi controls and are bounded |
| 14 | OSS, documentation, workflows and release assets | MIT/community/security files present; local links, PowerShell parser, `sh -n` and actionlint passed |
| 15 | Tests, static analysis, vulnerabilities and reproducibility | tests×15, race×3, vet, staticcheck, govulncheck and two byte-identical five-target builds passed |

## Confirmed findings and closure

| ID | Finding | Closure |
|---|---|---|
| LO-001 | User-facing TUI and artifact names still exposed the pre-rename product | Central brand constants, new artifact contract and TUI header |
| LO-002 | Every Go source file was Windows-selected, so Linux could not build | Shared portable core plus native Linux vertical slices |
| LO-003 | Linux affinity assumed CPUs were contiguous from zero | Validate requested bits against the process's actual cpuset and read back all CPUSet bits |
| LO-004 | Linux cleanup recursively removed matching non-empty temp directories | Remove only direct app-owned files and empty directories; nested data is never traversed |
| LO-005 | Test-only loopback HTTP allowance could have enabled production updater SSRF | Insecure loopback is disabled by default and exists only behind a test seam; production requires trusted HTTPS/443 |
| LO-006 | Concurrent self-updates and damaged update config lacked recovery | Platform update locks plus atomic replaceable config; enable/disable can recover malformed JSON |
| LO-007 | Automatic update did not run for the no-argument TUI/default flow | Auto-check now covers both default and command launches while excluding internal/elevated children |
| LO-008 | Remote/local names could inject terminal controls into human output | Shared control/format stripping and 512-rune display bound; JSON remains lossless |
| LO-009 | Shell build staging used a predictable PID directory | `mktemp -d` creates the staging boundary before any artifact is written |
| LO-010 | Network limits allowed a 1000-second worst case | Reject configurations whose timeout budget exceeds five minutes |
| LO-011 | Empty game/service/startup/finding collections serialized as `null` | Stable empty arrays for machine-readable inventory contracts |
| LO-012 | Build scripts accepted `+metadata` while the updater intentionally did not | Release builders now accept stable or `-prerelease` `1.0.x` versions only |
| LO-013 | Update asset redirects and URLs did not constrain non-standard HTTPS ports | Every production request and redirect must stay on approved hosts and port 443 |
| LO-014 | A remote release URL was printed without needing remote presentation data | Display URL is constructed from the pinned repository and validated tag |
| LO-015 | Windows 8.3 temp roots and canonical EXE paths compared as different directories in CI | Canonicalize the Windows discovery root before walking and containment checks |
| LO-016 | Official Actions using Node 20 emitted runner deprecation warnings | Moved checkout, setup-go and artifact upload to verified Node 24 major tags |

## Review-of-review evidence

- `go test -count=15 -mod=readonly ./...`: passed.
- `go test -race -count=3 -mod=readonly ./...`: passed.
- `go vet`, `go mod verify`, `go mod tidy -diff`: passed/clean.
- `staticcheck v0.6.1`: clean.
- `govulncheck v1.1.4`: no vulnerabilities found.
- `actionlint v1.7.7`, PowerShell AST parse and `sh -n build.sh`: passed.
- Native Linux test binary on Ubuntu 24.04 WSL: all tests passed.
- Real Linux boost fallback with unavailable GameMode/CAP_SYS_NICE and explicit affinity: passed without persistent mutation.
- Windows amd64 audit and Windows amd64/386 version smoke: passed at `1.0.0`.
- Two consecutive complete builds produced identical hashes for all five binaries and `SHA256SUMS.txt`.
- Banner SVG was rendered at 1600×520 and visually inspected.

## Remaining platform evidence

- Linux arm64 and Windows arm64 are cross-build verified but were not executed on physical arm64 hardware in this review.
- Feral GameMode was unavailable in the Ubuntu WSL environment; the direct-launch fallback was live-tested and the wrapper path is covered by capability selection.
- The Windows updater helper was compile/static reviewed rather than allowed to replace the running development binary; release checksums and Linux atomic replacement were exercised directly.
