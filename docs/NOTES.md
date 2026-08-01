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
| CPU EPP 0 / aggressive boost | Can improve AC responsiveness | Applied only when the platform exposes each setting |
| PCIe/USB AC power saving off | Avoids wake latency under the maximum profile | AC only; maximum profile warns about heat/power |
| Ethernet EEE / Interrupt Moderation off | Can reduce latency | Physical Ethernet only; driver must advertise the keyword |
| Startup/menu delay off | Improves perceived desktop responsiveness | User values are backed up and restored |

## Conditional or benchmark first

| Tweak | Decision |
|---|---|
| HAGS | Per-GPU/per-driver result varies; add only with before/after frametime evidence |
| Core parking | Can hurt hybrid scheduling and boost; benchmark per CPU before adding |
| MSI/IRQ affinity | Hardware topology and driver dependent; keep outside universal profiles |
| RSS/RSC/offloads | Throughput/CPU/latency trade-off; do not apply globally |
| Process priority changes | Games and anti-cheat may override them; require per-game evidence |

## Rejected

- Defender/Firewall/mitigation disabling.
- Global `PowerThrottlingOff`: Game Mode already scopes gaming behavior, while the global override can increase heat and battery drain outside games.
- Undocumented power GUID `d4e98f31-5ffe-4ce1-be31-1b38b384c009`: no stable alias or query semantics, so it is not an optimization target.
- BCD/HPET/dynamic-tick recipes.
- Memory cleaners, standby-list loops and memory-compression disabling.
- Fixed thread counts, affinity masks and timer-resolution daemons.
- Private NVIDIA/AMD/Intel registry keys.
- Disabling Windows Update, scheduled maintenance or core services.
- Mutable runtime downloads or bundled unsigned tools.

## Review checklist

1. State the exact measurable outcome: median FPS, 1% low, frametime variance, latency or startup time.
2. Run at least three identical before/after passes.
3. Reject changes within normal run-to-run variance.
4. Keep hardware capability checks and AC/battery scope explicit.
5. Require backup, apply, read-back verify and rollback before shipping.
6. Record every accepted or rejected candidate in this file and every release change in `CHANGELOG.md`.
