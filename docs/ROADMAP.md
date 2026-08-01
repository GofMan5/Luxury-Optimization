# Roadmap

The target is BoosterX-class coverage with fewer permanent tweaks: measure first, use Windows-native controls, and keep every mutation bounded, verified and reversible.

## Current

- Temporary game boost: elevated profile apply, non-elevated direct EXE launch, process monitoring and automatic restore.

## Next vertical slices

1. Game library detection for Steam, Xbox, Epic and manually selected EXEs.
2. Per-game profile selection and launch history without changing game files.
3. Read-only startup/background-load advisor with measured impact and explicit user actions.
4. Network and storage diagnostics that recommend changes only after capability checks.
5. Repeatable before/after benchmark comparison for FPS, 1% lows and frametime variance.
6. Restore center for inspecting and retrying durable backups.

## Excluded

- Defender, Firewall, Windows Update, mitigations or core-service disabling.
- BCD/HPET/dynamic-tick recipes and timer-resolution daemons.
- Memory cleaners, fixed affinity/IRQ masks and private GPU registry values.
- Universal HAGS, MSI, offload or process-priority changes without per-PC benchmark evidence.
