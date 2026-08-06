# Product and tweak decisions

## Product rule

A change belongs in Luxury Optimization only when it targets a measurable outcome, has a bounded platform/hardware scope and fails safely. Persistent mutations additionally require exact backup, read-back verification and rollback.

Unsupported settings are capabilities, not errors: report them, mark them `skipped` and continue without partial mutation.

## Accepted

| Capability | Why it stays | Guardrail |
|---|---|---|
| Windows Game Mode and Game DVR controls | Native gaming policy and removable background capture work | Fixed targets, exact backup and read-back |
| Windows mouse acceleration | Predictable pointer response | Registry plus live SPI capture/apply/restore |
| Manual per-tweak apply | Lets users opt into only the changes they understand | Separate sealed backup, exact target read-back and persistent one-click rollback |
| Lite / Medium / Max profiles | Makes risk and optimization intensity explicit instead of one ambiguous preset | Strictly nested targets; Lite low-risk only, Medium reviewed CPU subset, Max full native catalog |
| Separate Windows performance plan | Does not mutate the user's original plan | New GUID, AC-only values, read-back and delete on rollback |
| CPU 5–100%, EPP and boost | Responsive without pinning idle clocks at 100% | Only settings exposed by the machine |
| Native processor and storage AC catalog | Uses the installed Windows High Performance defaults instead of third-party constants | Processor/storage subgroups only; 512-entry bound, capability filter, cloned plan and exact read-back |
| PCIe/USB AC power saving | Can avoid wake latency in Max | Temporary plan; battery/heat warning |
| Ethernet EEE/Interrupt Moderation | Can reduce latency on some physical NICs | Only driver-advertised keywords; exact restore |
| Temporary game boost | Keeps aggressive state scoped to a game session | Non-admin game, session lock and automatic Windows restore |
| Explicit process priority/affinity | Useful in measured contention cases | Per-game only, no realtime, capability/read-back checks |
| Game discovery and saved profiles | Removes setup friction | Read-only manifests, bounded scan, canonical executable revalidation |
| Startup manager | Reduces deliberate background load | Explicit user entry only; system scope remains read-only |
| Service inventory | Makes background state visible | Read-only on Windows SCM and systemd |
| TCP and benchmark tools | Produce comparable evidence | Bounded timeouts, at least three runs, median/MAD threshold |
| Linux GameMode | Native session policy used by games and distributions | Used only when `gamemoderun` exists; direct fallback |
| Linux nice/affinity | Native session process controls | Missing `CAP_SYS_NICE` or affinity support is skipped |
| GitHub Release updater | Keeps standalone binaries current | Opt-in, `1.0.x` pin, HTTPS allowlist, size limits and SHA-256 |

## Conditional or benchmark first

| Tweak | Decision |
|---|---|
| HAGS and fullscreen optimizations | Per-GPU/per-driver results vary; require frametime evidence |
| Custom core-parking and scheduler values | Hybrid CPUs differ; only the installed Windows High Performance defaults are accepted without a per-system benchmark |
| MSI/IRQ affinity | Hardware topology and driver dependent |
| RSS/RSC/offloads/TCP ACK recipes | Throughput, CPU and latency trade-offs differ |
| Linux CPU governor changes | Distribution power managers and GameMode own this policy |
| Negative Linux nice | Requires `CAP_SYS_NICE`; never elevate a game to obtain it |

## Rejected

- Defender, Firewall, VBS, Credential Guard or mitigation disabling.
- BCD, HPET, dynamic-tick and timer-resolution recipes.
- Memory cleaners, standby-list loops, compression disabling and cache purges.
- Fixed affinity, IRQ masks, thread counts or universal scheduler values.
- Private NVIDIA/AMD/Intel registry keys or opaque profile imports.
- Disabling Windows Update, scheduled maintenance, systemd units or core services in bulk.
- IFEO priority/mitigation values for CSRSS, games or anti-cheat.
- App removal or debloat presented as an FPS optimization.
- Mutable runtime downloads, unsigned bundled tools or update assets without a release checksum.
- Pretending Windows registry/power tweaks exist on Linux.

## Measurement checklist

1. Name the metric: median FPS, 1% low, p95 frametime, latency, startup time or resource use.
2. Run at least three identical before/after passes.
3. Reject differences inside normal run-to-run variance.
4. Record OS, hardware, driver, power and thermal conditions.
5. Verify capability detection and unsupported-system behavior.
6. For persistent state, prove backup, apply, read-back and rollback including failure paths.
