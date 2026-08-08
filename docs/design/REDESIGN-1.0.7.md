# Desktop redesign specification — 1.0.7

## Direction

Quiet premium workstation, not a neon gaming launcher and not a wall of cards. Near-black layered surfaces, one champagne-gold action/selection color, open tables and strong task hierarchy. Native Segoe UI Variable remains the default; no font or component dependency is added.

## Shell

- 226 px desktop sidebar; 76 px icon rail below 1080 px.
- Persistent 66 px top bar carries route context and signed updater state/action.
- Content width: 1244 px; desktop padding 44 px, compact padding 28–34 px.
- Minimum supported window remains 900×640.

## Home anatomy

1. Page identity and measured OS status.
2. Command center: recommended next action, primary/secondary commands, percentage of Lite targets already matched and four factual system fields.
3. Four-step workflow rail: Audit, Apply, Measure, Restore.
4. Capability list with availability and operating mode.
5. Bounded session cleanup.

The readiness percentage is derived only from `matched Lite targets / total Lite targets`; it is not a health score or promised performance gain.

## Optimization anatomy

- Default view is Profiles: Lite, Medium, Max.
- One compact switch exposes Expert catalogue.
- Profile selection is followed by checkpoint state, selected summary, exact expandable rows and one apply footer.
- Exact current/desired values, benefit, trade-off, backup availability and per-item restore remain visible.

## Measurements and tools

- Measurements tabs: Game result, Network, Storage.
- Tools tabs: Background load, Startup, Services.
- Existing screen implementations are embedded rather than copied; one source owns each request, mutation and state.
- Storage analyzer keeps its separate window and own scroll lifecycle.

## States

- Loading: existing `aria-busy` resource state.
- Empty/error: existing bounded EmptyState and inline `role=alert`, with local retry.
- Applying/installing: initiating controls disabled; progress or busy label visible.
- Confirmation: exact scope plus recovery behavior; destructive storage action keeps name confirmation when required.
- Success: dismissible `role=status`; data cache invalidated at the existing mutation boundary.
- Unsupported capability: visible as skipped/read-only, never partially applied.

## Visual tokens

- Background `#080a0e`; working surfaces `#0e1218`, `#12171f`, `#151b24`.
- Text `#f5f1e7`; muted `#8b96a4`; champagne `#d7b665`; success `#63d59f`; danger `#ef7a7a`.
- Radius 10/14/18 px. Primary controls 50 px; normal controls 38–44 px.
- Motion 140–180 ms; no continuous decorative animation; honor `prefers-reduced-motion`.

## Measurable QA gates

- No relevant console errors/warnings in preview.
- `documentElement.scrollWidth === clientWidth` at 900×640.
- Six primary nav controls, three tabs in Measurements and three in Tools.
- Profile, storage analyzer and recovery actions remain reachable by accessible names.
- `pnpm typecheck`, `pnpm test`, `pnpm build`, Rust and Go release gates pass on the final diff.
