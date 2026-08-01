# Product and tweak decisions

## Product rule

The project optimizes gaming responsiveness, frametime stability and general Windows responsiveness. A tweak belongs in the product only when it has a plausible measurable benefit, a bounded target, exact backup, read-back verification and rollback.

## Accepted

| Tweak | Why it stays | Guardrail |
|---|---|---|
| Game Mode on | Windows-native gaming scheduling policy | Current value is planned, backed up and verified |
| Game DVR capture off | Removes background capture work when unused | Only capture settings are changed |
| Mouse acceleration off | Predictable raw pointer response for games | Live state is captured, applied through SPI and restored |
| Separate performance power plan | Avoids mutating the user's original plan | New GUID, read-back verification, delete on rollback |
| CPU minimum 5%, maximum 100% | Preserves fast boost without pinning idle clocks at 100% | AC-only temporary plan; EPP/boost remain capability checked |
| CPU EPP 0 / aggressive boost | Can improve AC responsiveness | Applied only when the platform exposes each setting |
| PCIe/USB AC power saving off | Avoids wake latency under the maximum profile | AC only; maximum profile warns about heat/power |
| Ethernet EEE / Interrupt Moderation off | Can reduce latency | Physical Ethernet only; driver must advertise the keyword |
| Startup/menu delay off | Improves perceived desktop responsiveness | User values are backed up and restored |
| Temporary game boost | Keeps the maximum profile scoped to one play session | Game stays non-elevated; session lock, exact backup and automatic restore |
| Per-game process priority | Useful for a measured CPU-contention case | Explicit only, process-scoped, native read-back; realtime is rejected |
| HKCU startup manager | Removing unused launchers lowers persistent background load | Exact string type/value backup; no automatic disabling and no HKLM mutation |
| Game library discovery | Reduces setup friction without installing a service | Read-only Steam/Epic/Xbox scan; every selected EXE is revalidated |
| Service inventory | Makes background activity visible | Minimal-query SCM handles; no start-mode mutation |
| TCP latency baseline | Detects reachability, jitter and tail-latency changes | Explicit endpoint, bounded count/timeouts; does not claim UDP/bufferbloat |
| Benchmark comparison | Separates real gains from one-run noise | Minimum three runs, medians and MAD-derived threshold |
| Restore center | Makes rollback state inspectable without weakening backup ACL | Admin-only list, seal validation and exact ID restore |

## Conditional or benchmark first

| Tweak | Decision |
|---|---|
| HAGS | Per-GPU/per-driver result varies; add only with before/after frametime evidence |
| Core parking | Can hurt hybrid scheduling and boost; benchmark per CPU before adding |
| MSI/IRQ affinity | Hardware topology and driver dependent; keep outside universal profiles |
| RSS/RSC/offloads | Throughput/CPU/latency trade-off; do not apply globally |
| Process priority changes | Games and anti-cheat may override them; require per-game evidence |
| CPU affinity | Hybrid topology and anti-cheat behavior vary; explicit per-game mask only |

## Rejected

- Defender/Firewall/mitigation disabling.
- Global `PowerThrottlingOff`: Game Mode already scopes gaming behavior, while the global override can increase heat and battery drain outside games.
- Undocumented power GUID `d4e98f31-5ffe-4ce1-be31-1b38b384c009`: no stable alias or query semantics, so it is not an optimization target.
- BCD/HPET/dynamic-tick recipes.
- Memory cleaners, standby-list loops and memory-compression disabling.
- Fixed thread counts, affinity masks and timer-resolution daemons.
- Private NVIDIA/AMD/Intel registry keys.
- Disabling Windows Update, scheduled maintenance or core services.
- Mass service start-mode changes, device disabling and forced MSI/IRQ affinity.
- IFEO CPU/I/O/page priority for system processes or persistent game entries; live process APIs cover the scoped case.
- Mutable runtime downloads or bundled unsigned tools.

## Review checklist

1. State the exact measurable outcome: median FPS, 1% low, frametime variance, latency or startup time.
2. Run at least three identical before/after passes.
3. Reject changes within normal run-to-run variance.
4. Keep hardware capability checks and AC/battery scope explicit.
5. Require backup, apply, read-back verify and rollback before shipping.
6. Record every accepted or rejected candidate in this file and every release change in `CHANGELOG.md`.
