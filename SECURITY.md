# Security policy

## Supported versions

Only the latest `1.0.x` release is supported. Older patch releases should update before reporting a problem.

## Reporting

Use **Security → Report a vulnerability** in the GitHub repository. Do not open a public issue for a suspected vulnerability, bypass, destructive path or update-chain problem.

Include the version, OS/architecture, affected command, preconditions and a minimal non-destructive reproduction. Do not include secrets, personal paths or third-party data.

## Update trust

The built-in updater accepts only the pinned `1.0.x` channel, selects an exact OS/architecture asset from this repository's latest GitHub Release, requires HTTPS and verifies the binary against the release's `SHA256SUMS.txt`. Automatic installation is opt-in with `update enable`.

Checksums protect transfer integrity; they do not protect against compromise of the GitHub maintainer account. Releases should therefore be built by the repository workflow, and maintainers must keep strong GitHub authentication enabled.
