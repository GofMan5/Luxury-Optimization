# BoosterX coverage and tweak decisions

Research baseline: official BoosterX website and PRO matrix, public version 2.2.4.3 metadata, and a public partial 2.2.1.0 source snapshot inspected on 2026-08-01. No BoosterX binary, resource or source code is shipped in this project.

## Product coverage

| BoosterX capability | Luxury Optimization implementation | Status |
|---|---|---|
| System analysis and boost potential | `audit`, exact `plan`, hardware and startup findings | Covered |
| Preview before apply | Current → desired value for every target | Covered |
| Backup and restore | ACL-protected, SHA-256-sealed journal and retryable rollback | Stronger |
| Restore center | Admin-only verified inventory plus exact `restore --id` | Covered in CLI |
| Basic and advanced system tweaks | 100+ native processor, storage, PCIe, USB, gaming and UI actions with capability checks | Curated |
| Power settings | Separate temporary AC plan, native High Performance defaults, 5–100% CPU and exact read-back | Covered |
| GameModeX / game profiles | Desktop discovery/saved profiles, bounded launch history and `boost` | Covered without daemon |
| ProcessX priority and affinity | Live process API, explicit per-game values, read-back | Covered without IFEO |
| Steam/Epic/Xbox discovery | `games scan`; Linux supports Steam native/Flatpak | Covered by platform |
| Startup manager | HKCU list/disable/enable with exact backup | Covered; HKLM remains read-only |
| GameReadyX | `audit` plus profile drift and capability plan | Partial; checks stay transparent |
| Tweak explanation | README, exact plan, NOTES and this matrix | Covered |
| Questionnaire | Automatic audit plus deterministic Lite/Medium/Max plans | Replaced; no ambiguous bulk answers |
| Service groups/manager | Read-only Windows SCM and Linux systemd inventory | Inspection covered; mutation rejected |
| APPX deep removal | Outside gaming performance scope | Excluded |
| NVIDIA panel/profile import | Driver UI remains authoritative | Private/undocumented settings excluded |
| UDP/bufferbloat and scheduler-latency tests | TCP median/p95/jitter baseline implemented | UDP/bufferbloat pending stable protocol |
| Before/after comparison | Desktop manual/import flow plus explicit per-game evidence attachment and MAD noise verdict | Covered |
| Device Control / MSI / IRQ | Hardware-specific and restart-sensitive | Benchmark first |
| Fixes unrelated to optimization | Outside product scope | Excluded |
| AI-generated/imported arbitrary tweaks | No trusted rollback contract | Excluded |

Official feature sources:

- <https://boosterx.org/ru/howitworks/>
- <https://boosterx.org/ru/pro/>

## Registry and system tweak policy

| Class | Decision | Better implementation |
|---|---|---|
| Game Mode and Game DVR capture | Accepted | Fixed allowlist, backup and live verification |
| Mouse acceleration | Accepted | Registry snapshot plus live `SystemParametersInfoW` apply/read-back |
| UI animation/transparency and startup/menu delay | Accepted | User-scoped and exactly restorable |
| Native CPU and storage High Performance policies | Accepted | Enumerate documented power APIs; separate AC plan; apply only exposed settings |
| PCIe/USB AC power saving | Accepted in Max | Temporary plan and heat/power warning |
| Ethernet EEE/Interrupt Moderation | Accepted in Max | Physical Ethernet and advertised keywords only |
| HKCU Run startup entries | Accepted by explicit name | Backup before delete, type/value read-back, exact enable |
| Process priority | Conditional | Live child process only; normal/above-normal/high, never realtime |
| CPU affinity | Conditional | Explicit mask only, processor-group and CPU-count validation |
| HAGS and fullscreen optimizations | Benchmark first | Per-PC/per-game evidence before any mutation |
| MMCSS `SystemResponsiveness` / `NetworkThrottlingIndex` | Benchmark first | No universal values; GPU Priority and SFIO Priority are documented as unused |
| `Win32PrioritySeparation` | Benchmark first | Scheduler policy varies by build and workload |
| RSS/RSC/offloads/TCP ACK recipes | Benchmark first | Protocol, NIC and throughput/latency trade-offs differ |
| Services and scheduled tasks | Explicit advisor only | No bulk disable presets |
| AppX/debloat and cache deletion | Excluded from performance profile | Removing apps/data is not a reversible FPS tweak |
| Defender, Firewall, VBS, Credential Guard and mitigations | Rejected | Security stays enabled |
| BCD, HPET, dynamic tick and timer daemons | Rejected | Preserve Windows timer selection |
| `PowerThrottlingOff`, `DisablePagingExecutive`, `LargeSystemCache` | Rejected | Global side effects without stable gaming gain |
| `SvcHostSplitThresholdInKB` / per-service `SvcHostSplitDisable` | Rejected | No mass service-host topology mutation |
| IFEO priority/mitigation values for CSRSS, games or anti-cheat | Rejected | Process-scoped native API only where requested |
| MSI/IRQ masks and device disabling | Rejected as universal tweaks | Hardware-specific benchmark path only |
| Private NVIDIA/AMD/Intel registry keys or imported opaque profiles | Rejected | Vendor control panel and documented APIs only |
| Memory cleaners, standby-list loops and shader-cache purges | Rejected | They commonly add stutter or force cache rebuilds |

Useful Microsoft references:

- <https://learn.microsoft.com/windows/win32/procthread/multimedia-class-scheduler-service>
- <https://learn.microsoft.com/windows/win32/api/processthreadsapi/nf-processthreadsapi-setpriorityclass>
- <https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-setprocessaffinitymask>
- <https://learn.microsoft.com/windows/win32/setupapi/run-and-runonce-registry-keys>
- <https://learn.microsoft.com/windows/win32/api/powrprof/nf-powrprof-powerenumerate>
- <https://learn.microsoft.com/windows/win32/api/powrprof/nf-powrprof-powerreadacdefaultindex>
