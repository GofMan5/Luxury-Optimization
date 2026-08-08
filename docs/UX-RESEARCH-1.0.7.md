# UX research — 1.0.7

Date: 2026-08-08. Scope: current desktop source, rendered preview at 1440×900 and 900×640, official BoosterX product flow, Microsoft Windows navigation guidance, WinUtil, Optimizer, FlyOOBE, UniGetUI and BleachBit.

## Product decision

Luxury Optimization is not a catalogue of magic switches. Its primary job is a short, provable loop:

`audit current state → choose Lite/Medium/Max → inspect exact changes → create checkpoint → apply/read back → measure → restore if worse`

The competitor pattern worth retaining is the clear `analysis → mode → preview → apply → restore` sequence. AI-generated arbitrary tweaks, APPX removal, bulk service disabling, undocumented driver profiles, IFEO/MSI/IRQ recipes, protection disabling, BCD/timer recipes and memory cleaners remain excluded because they fail the product's capability, evidence or rollback contract.

## Source inventory and observed friction

| Existing surface | Preserved actions | Finding |
|---|---|---|
| Overview | audit, Lite plan, capabilities, bounded temp cleanup | A synthetic `/10` score did not say what to do next; workflow was spread across routes. |
| Tweaks | 124+ capability-dependent items, exact values, filters, manual apply/restore, Lite/Medium/Max, checkpoint | Expert catalogue opened first and hid the normal profile flow. |
| Games | discovery, save/remove, launch, history, benchmark attachment | Correct standalone task; keep. |
| Benchmarks | import/manual three-run comparison, noise verdict, attach to game | Measurement belonged beside network and storage diagnostics. |
| System | background sample, startup, services, network, storage | Five equal nested tabs mixed mutation tools and diagnostics. |
| Restore | sealed Luxury backups, latest/exact restore, native Windows Restore | Correct standalone safety surface; keep. |
| Updates | automatic/manual signed update | Too little content for a permanent primary-navigation destination. |
| Storage analyzer | independent window, bounded parallel scan, cache, drill-down, explicit rescan, recycle-bin confirmation | Correct independent task; main app stays usable. |

Backend contracts and legacy backup, registry, state and mutex identifiers are unchanged.

## Target IA

1. **Home** — current state, one recommended action, workflow, supported capabilities and quick cleanup.
2. **Optimization** — profiles first; exact preview/checkpoint/apply. Expert catalogue is secondary but fully retained.
3. **Games** — discovery, saved game sessions and evidence history.
4. **Measurements** — game before/after, network and storage including the independent analyzer.
5. **Tools** — measured background load, startup and services.
6. **Recovery** — exact Luxury rollback and native Windows Restore.

Signed updates and RU/EN are global shell controls because they apply to the whole product.

## Acceptance criteria

- Primary navigation has six items and no empty destination.
- From Home, Optimization, Measurements and Recovery are each one action away.
- Optimization opens Lite/Medium/Max; expert catalogue remains reachable in one action.
- Every mutation still shows scope and confirmation; checkpoint and rollback remain enforced by existing backend contracts.
- Measurements groups benchmark, network and storage without duplicating their fetches or business logic.
- Update check starts after first render, still runs automatically once, can be triggered manually, accepts only `1.0.x`, verifies the signed platform bundle and relaunches after install.
- At 1440×900 and minimum 900×640: no horizontal overflow, clipped primary action or inaccessible navigation.
- Keyboard focus remains visible; tab semantics and text labels supplement color; reduced motion remains respected.
- No new runtime dependency, backend method, tweak or unsupported mutation is introduced by the redesign.

## Rendered baseline result

Playwright fallback was used because the in-app Browser runtime returned `No browser is available`. Preview rendered with the repository `PreviewBackendClient`. Home, Optimization, Measurements, Storage and the independent analyzer were inspected; page console had no warning/error and the 900×640 document width matched the viewport.
