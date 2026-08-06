# Tweak review for 1.0.2

Date: 2026-08-06

Baseline: `8278656` (19 actions on the verification machine)

Reviewed result: 124 actions, exactly **+105**.

## Review boundary

The added power actions are not copied registry recipes. Windows enumerates them through `PowerEnumerate`; desired AC values come from the installed High Performance personality through `PowerReadACDefaultIndex`. Only processor and storage subgroups are accepted. Every supported value is applied to a clone of the active plan, read back, and deleted during rollback.

The review kept these global exclusions: security controls, core services, BCD/timers, device disabling, private GPU keys, fixed IRQ/affinity values, cache purges and debloat.

## Profile verdicts

| Profile | Verified actions | Risk result | Included | Excluded |
|---|---:|---|---|---|
| Lite | 6 | 6 low | Game Mode, transparency/animation reductions and menu response | Capture, mouse feel, power, storage and network |
| Medium | 23 | 9 low, 14 medium | Lite + Game DVR/app capture, three mouse fields, 11 reviewed CPU AC policies | Hidden scheduler/parking policies, storage, PCIe/USB and Ethernet |
| Max | 124 | 9 low, 18 medium, 97 high | Complete supported reviewed catalog | Unsupported settings and globally rejected classes |

The sets are strictly nested: `Lite ⊂ Medium ⊂ Max`. A matching sealed checkpoint is required before any preset apply.

## Added power catalog review

| Class | Count | Tier | Verdict and guardrail |
|---|---:|---|---|
| Processor | 95 | 11 Medium, all Max | Native OS defaults only. The 84 internal scheduler, parking, heterogeneous-core and transition policies remain Max-only because their benefit is workload and topology dependent. |
| Storage | 10 | Max | AHCI/NVMe/disk AC policies can reduce wake latency but can increase power and heat. Disk power-off is deliberately overridden to `0` on AC to avoid spin-up stalls. |
| PCIe / USB | 2 | Max | Link-state and selective-suspend changes are capability-checked and restricted to the cloned plan. |
| Physical Ethernet | Dynamic (4 here) | Max | Only driver-advertised EEE/interrupt-moderation properties; exact adapter identity and value are backed up. |

### Medium CPU allowlist

Medium contains only these reviewed setting families:

- Maximum processor state, efficiency classes 0–2: `bc5038f7-23e0-4960-96da-33abaf5935ec` through `...35ee`.
- Minimum processor state, efficiency classes 0–2: `893dee8e-2bef-41e0-89c6-b55d0929964c` through `...964e`; all deliberately capped at 5% idle floor.
- Energy Performance Preference, efficiency classes 0–2: `36687f9e-e3a5-4dbf-b1dc-15eb381c6863` through `...6865`.
- Processor boost mode: `be337238-0d82-4146-a960-4f3749d470c7`.
- Active system cooling policy: `94d3a615-a899-4ac5-ae2b-e4d8f634367f`.

All other enumerated processor GUIDs are Max-only. Unsupported efficiency classes disappear from the plan before mutation.

## Compatibility decision

- New public IDs are `lite`, `medium` and `max`.
- Legacy `recommended` retains its original 10 registry targets for old backup validation.
- Legacy `maximum` retains the original registry scope and maps to Max power behavior.
- Legacy backup format, catalog version, state paths, registry seal and mutex names remain unchanged.

## Verification contract

```powershell
Luxury-Optimization-windows-amd64.exe plan --profile lite --json
Luxury-Optimization-windows-amd64.exe plan --profile medium --json
Luxury-Optimization-windows-amd64.exe plan --profile max --json
go test -race -mod=readonly ./...
pnpm check
cargo test --locked
```

The verification machine returned `6 / 23 / 124` actions with zero plan warnings. Platform-specific omissions are expected and are reported rather than approximated.
