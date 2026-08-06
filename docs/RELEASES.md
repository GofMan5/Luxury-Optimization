# Release policy

- Current public line: `1.0.x`.
- Patch versions may continue beyond three digits; there is no automatic minor bump.
- A `1.1.0` release requires an explicit project decision and a code change to the pinned release channel.
- Tags are `v1.0.N` and must point to the exact source used by the release workflow.
- Every public release contains Windows and Linux Tauri bundles, updater signatures and `latest.json`.
- The cross-platform Go recovery CLI matrix remains a CI artifact and can be built locally with `build.ps1` or `build.sh`.
- The latest patch is the only supported version.

The release workflow rejects tags outside `v1.0.N` and requires package, Cargo and Tauri versions to match the tag. The desktop updater independently rejects a release outside the compiled `1.0` channel and verifies the platform artifact with the pinned public key.
