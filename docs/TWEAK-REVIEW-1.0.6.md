# Tweak review for 1.0.6

Date: 2026-08-08

Baseline: `v1.0.5` with the 1.0.2 native power catalog.

Final verification machine: **Lite 6 / Medium 23 / Max 126**, zero plan warnings.

## Profile verdicts

| Profile | Verified actions | Risk result | Final boundary |
|---|---:|---|---|
| Lite | 6 | 6 low | User-scoped Game Mode and interface responsiveness only; no power, storage or network mutation. |
| Medium | 23 | 9 low, 14 medium | Lite plus capture/input controls and 11 reviewed CPU AC policies in a cloned plan; no storage or NIC mutation. |
| Max | 126 | 9 low, 20 medium, 97 high | Complete capability-exposed native catalog plus physical Ethernet and Wi-Fi policies; unsupported hardware is omitted before mutation. |

The sets remain strictly nested and every persistent apply still requires the existing sealed checkpoint, exact backup, native read-back and rollback.

## Additions reviewed in 1.0.6

### Physical Ethernet energy controls

The Max query now recognizes additional driver-advertised energy keywords, including `PowerSavingMode`, `EnableGreenEthernet`, `GreenEthernet`, `*GreenEthernet` and `AutoDisableGigabit`. They stay Max-only because latency, throughput, CPU and power trade-offs are adapter-specific.

Guardrails:

- only `Get-NetAdapter -Physical` adapters with Ethernet physical medium are considered;
- only the fixed keyword allowlist is accepted;
- exact interface GUID, keyword and original values remain in the legacy-compatible backup;
- mutation and rollback both perform native read-back.

The verification machine exposed six supported Ethernet actions.

### Wi-Fi Maximum Performance on AC

The documented wireless power GUID `12bbebe6-58d6-4636-95bb-3217ef867c1a` is added to Max only when native adapter enumeration finds a physical `IF_TYPE_IEEE80211` interface. The setting is applied only inside the cloned AC plan and is omitted on the verification machine because no physical Wi-Fi adapter was present.

### Game-process Power Throttling / EcoQoS

Each explicitly launched game now clears the execution-speed Power Throttling bit through `SetProcessInformation`, then verifies the result with `GetProcessInformation`. This is process-local and session-only: no registry, IFEO, service or global scheduler state is changed. Missing native support is treated as unavailable instead of emulated.

## Exclusions retained

- Defender, Firewall, VBS, mitigations, Windows Update and core services.
- BCD/HPET/dynamic tick, timer daemons and global `PowerThrottlingOff`.
- Universal HAGS, MSI/IRQ, TCP/offload, fixed affinity and private GPU registry recipes.
- Cache purges, memory cleaners, debloat and opaque imported profiles.

## Evidence

```text
Lite   items=6   risks={low:6}                     warnings=[]
Medium items=23  risks={low:9, medium:14}          warnings=[]
Max    items=126 risks={low:9, medium:20, high:97} warnings=[]
```

`TestProfilesAreStrictlyTieredAndLegacyCompatible`, `TestLiteAndMediumContainNoHigherRiskItems`, `TestWiFiPowerAndEthernetKeywordsFollowDetectedCapabilities`, `TestProcessTuningReadback`, full Go race/vet/staticcheck, frontend, Rust and cross-platform CI all passed on the final release commit.
