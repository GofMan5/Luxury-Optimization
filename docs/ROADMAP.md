# Roadmap

The target is BoosterX-class coverage with fewer permanent tweaks: measure first, use Windows-native controls, and keep every mutation bounded, verified and reversible.

## Current

- Temporary game boost: elevated profile apply, non-elevated direct EXE launch, process monitoring and automatic restore.
- Steam, Epic and fixed-drive Xbox discovery.
- Saved per-game profiles with process-scoped priority/affinity and arguments.
- Reversible HKCU startup manager and startup-load audit.
- Read-only service/interface inventory, TCP latency baseline and multi-run benchmark comparison.
- Admin-only sealed backup inventory and exact backup restore.

## Next vertical slices

1. Launch history and measured per-game before/after results.
2. Background-load advisor that correlates services/startup/processes without bulk disabling.
3. UDP/bufferbloat and storage diagnostics with stable measurement protocols.
4. Automated capture import for FPS, 1% lows and frametime variance.
5. Mouse-first restore-center UI over the sealed backup inventory.

## Excluded

- Defender, Firewall, Windows Update, mitigations or core-service disabling.
- BCD/HPET/dynamic-tick recipes and timer-resolution daemons.
- Memory cleaners, fixed affinity/IRQ masks and private GPU registry values.
- Universal HAGS, MSI, offload or process-priority changes without per-PC benchmark evidence.
