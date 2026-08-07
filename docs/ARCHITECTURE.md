# Architecture

Luxury Optimization is a Tauri v2 desktop application with a React frontend and a Go sidecar. The boundary is a versioned, newline-delimited JSON protocol; the WebView cannot execute arbitrary commands.

## Frontend

`frontend/src` is organized by user-facing slices: overview, tweaks, games, benchmarks, system, restore and updates. Shared contracts describe only JSON data crossing the sidecar boundary. A small TTL cache deduplicates reads and invalidates the affected prefixes after mutations.

The Tauri host in `frontend/src-tauri` owns the sidecar process. Rust validates frame size, request IDs and an exact method allowlist before a command reaches Go. The sidecar has no shell passthrough.

## Backend

`backend/internal/app` dispatches the same exact method allowlist into vertical feature handlers under `backend/internal/features`. Platform and transaction code stays behind `backend/internal/optimizer`; the frontend never receives registry paths or executable primitives to submit.

On Windows, every persistent operation follows:

```text
resolve exact target → capture state → seal backup → journal intent → mutate → native read-back
                                                        └─ failure → verified rollback
```

Manual tweaks receive one backup ID each. Registry targets are allowlisted, Ethernet IDs include the adapter identity, mouse SPI rollback changes only the selected field, and a power tweak clones the active plan rather than editing it. Power backups must be unwound newest first.

Profiles are strict nested tiers: Lite has six low-risk registry targets, Medium adds capture/input plus 11 reviewed processor policies, and Max adds the remaining native processor/storage catalog, PCIe/USB and physical Ethernet. The legacy `recommended` and `maximum` IDs remain read-compatible for old backups and saved games.

Saved-game compatibility stays in the legacy `games.json` v1 location and mutex. Launch/evidence history is isolated in `game-history.json`: bounded records, atomic replacement, protected user permissions, recomputed benchmark verdicts and no executable data beyond the saved profile ID.

On Linux, Windows registry, power and NIC mutations are unavailable. Audit, GameMode/process sessions, XDG startup and read-only systemd inventory use native capabilities and skip unsupported behavior without partial mutation.

## Recovery boundary

Production Windows backups remain under the legacy ProgramData location for compatibility. The JSON file is restricted to administrators and SYSTEM, and its SHA-256 is sealed separately in HKLM. User-local quick-rollback markers contain only validated tweak and backup IDs; the elevated restore path revalidates the seal, owner SID, catalog target and backup shape before changing state.

## Release boundary

Windows and Linux bundles are built from the same source contracts. Updates must select the exact OS/architecture asset, stay on the `1.0.x` line and pass release-signature/checksum verification before installation.
