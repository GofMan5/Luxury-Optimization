# Desktop design system

The two SVG files in this directory are the implementation specification for the Tauri rewrite.

## Direction

- Quiet dark workstation UI, not a gaming launcher or neon dashboard.
- One champagne-gold accent (`#d7b665`) for selected navigation and intentional actions.
- Background `#06070a → #101620`; working surfaces `#0b0f15` / `#0d1118`.
- Content text `#f5f1e7`; secondary text `#7f8996`; success `#63d59f`.
- Segoe UI / Inter-like typography, compact controls, 12–22 px radii.
- Open tables and rails instead of repeated card grids.

## Shell

- Desktop target: 1440×900 logical pixels; concept: 1600×1000.
- Minimum target: 900×640. Sidebar collapses to icon rail below 1080 px.
- Navigation: Overview, Profiles, Games, Startup, Services, Network, Benchmarks, Restore, Updates.
- Runtime state and version stay visible without turning the header into a second navigation bar.

## Components

- Primary action: gold fill, dark text, 52 px height.
- Secondary action: transparent surface with restrained gold border.
- Tables use horizontal separators and fixed semantic columns, not nested cards.
- Capability state is text plus a small colored dot; color is never the only signal.
- Mutating actions require a confirmation surface showing exact scope and rollback behavior.
- Errors use inline `role=alert`; loading uses `aria-busy`; long-running operations remain cancellable where the backend supports cancellation.

## Motion

- 140–180 ms opacity/translate transitions for route and row updates.
- No continuous decorative animation.
- Respect `prefers-reduced-motion`.
