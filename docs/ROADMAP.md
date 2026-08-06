# Roadmap

The target is broad gaming-optimization coverage with fewer permanent tweaks and stronger evidence.

## Current

- Capability-aware Windows and Linux audit and plans.
- Reversible Windows recommended/maximum profiles and sealed restore center.
- Session game boost on Windows and Linux with platform-native process controls.
- Steam discovery on both platforms; Epic/Xbox discovery on Windows.
- Atomic per-game profiles, startup management, service/network inventory and benchmark comparison.
- Opt-in verified updates from GitHub Releases.
- Tauri v2 desktop UI with RU/EN, compact per-tweak actions and signed automatic update checks.

## Next vertical slices

1. Import CapFrameX/MangoHud captures into the existing median/MAD comparison.
2. Per-game launch history with explicit before/after result attachment.
3. Background-load advisor that correlates processes, startup and services without bulk disabling.
4. UDP/bufferbloat and storage diagnostics with stable, documented protocols.
5. Linux desktop/TUI parity only where it reuses the existing capability contract.

## Excluded

- Security-control, update or core-service disabling.
- BCD/HPET/timer daemons, memory cleaners and cache-purge loops.
- Universal HAGS, MSI, IRQ, governor, offload or priority changes without machine-specific evidence.
- Private driver keys, opaque profile imports and bundled third-party executables.
