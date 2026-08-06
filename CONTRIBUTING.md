# Contributing

Luxury Optimization accepts focused fixes and measurable performance work for Windows and Linux.

## Before coding

- Open an issue for a new tweak. State the target metric, supported hardware/OS boundary, failure mode and rollback.
- Do not submit Defender/Firewall/mitigation disabling, BCD/HPET recipes, memory cleaners, private driver keys, bulk service disabling or bundled unsigned binaries.
- Keep dependencies and permanent mutations to the minimum needed.

## Local checks

Go 1.25.12 or newer in the 1.25 line is required.

```sh
go test -race ./...
go vet ./...
go mod verify
go mod tidy -diff
```

Use `./build.sh 1.0.2` on Linux or `./build.ps1 -Version 1.0.2` on Windows to verify every recovery-CLI target. Build the desktop application from `frontend/` with `pnpm tauri:build`.

## Pull requests

- Keep one root cause per PR.
- Add the smallest regression test that proves non-trivial behavior.
- Update `docs/CHANGELOG.md` for user-visible changes and `docs/NOTES.md` for accepted or rejected tweak decisions.
- Preserve legacy Windows backup/state compatibility unless the PR includes a tested migration and rollback.
- Never commit generated `dist`, local agent files, credentials or machine-specific reports.

By contributing, you agree that your work is licensed under the MIT License.
