# Release policy

- Current public line: `1.0.x`.
- Patch versions may continue beyond three digits; there is no automatic minor bump.
- A `1.1.0` release requires an explicit project decision and a code change to the pinned release channel.
- Tags are `v1.0.N` and must point to the exact source used by the release workflow.
- Every release contains five binaries plus `SHA256SUMS.txt`.
- The latest patch is the only supported version.

The release workflow rejects tags outside `v1.0.N`. The built-in updater independently rejects a release outside the compiled `1.0` channel.
