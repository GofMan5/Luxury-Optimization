# Fifteen-pass project review

Date: 2026-08-01
Scope: complete active repository after the BoosterX-coverage implementation.

## Passes

| # | Independent review lens | Result |
|---:|---|---|
| 1 | Product requirements and BoosterX capability matrix | Implemented coverage and explicit exclusions match the product rule |
| 2 | Ponytail whole-repo complexity audit | No dependency or speculative abstraction to remove |
| 3 | CLI parsing and trust-boundary arguments | Fixed silent acceptance of unexpected positional arguments |
| 4 | Registry journal, apply, rollback and retry state | Added read-back verification for registry, mouse, power and Ethernet rollback |
| 5 | UAC, parent identity, internal flags and result-file ACL | Fixed ACL handle access; same-executable parent and internal boost guards verified |
| 6 | Filesystem, atomic publication, reparse points and path containment | Atomic profile writes and bounded regular-file reads verified |
| 7 | Cross-process locking, race behavior and crash state | Removed OS-thread mutex ownership; handle lifetime is the lock |
| 8 | Steam/Epic/Xbox discovery and saved game profiles | Traversal, stale paths, ID collision, PE and argument limits verified |
| 9 | Process priority/affinity and temporary power plan | Native read-back, realtime rejection, processor bounds and 5–100% CPU plan verified |
| 10 | Startup transaction and service inventory privileges | Added source re-read before delete; SCM uses query-only access |
| 11 | Network and benchmark statistics | Bounded TCP test and median/MAD comparison edge cases verified |
| 12 | TUI navigation, hitboxes, minimum width and discoverability | Repeated UI tests passed; help entry now names all new tools |
| 13 | Security, command execution, dependencies and vulnerabilities | Staticcheck clean; govulncheck reports no reachable vulnerabilities |
| 14 | Windows architecture compatibility | amd64, arm64 and 386 builds passed |
| 15 | Documentation, versions, external references and release contract | 3.2.0 synchronized; links, module graph and diff checks passed |

## Confirmed findings and closure

| ID | Finding | Closure |
|---|---|---|
| R15-01 | Temp files were opened without the access required to set their DACL | Reopen with `READ_CONTROL | WRITE_DAC`, compare file identity, then set protected ACL |
| R15-02 | Several commands ignored trailing positional input | Every non-launch command now rejects `NArg() != 0`; table regression test added |
| R15-03 | Rollback trusted successful API return codes without state read-back | Registry, live mouse, power plan and Ethernet must match the backup before retry flags clear |
| R15-04 | Named mutex release depended on Go returning to the owning OS thread | Locks now use atomic named-object existence and handle lifetime, with no ownership |
| R15-05 | Game manifests and profile IDs needed explicit collision/traversal closure | Canonical containment checks, malicious-manifest test and path collision rejection added |
| R15-06 | A startup value could change after backup but before deletion | Source type/value is read again immediately before delete |
| R15-07 | The standard service manager helper requested all-access | SCM and service handles now request enumeration/query rights only |
| R15-08 | One rollback error violated Go error-style conventions | Message normalized; staticcheck rerun clean |
| R15-09 | TUI help button did not expose the expanded tool surface | Button now names games/startup/services/network/benchmark/rollback |

## Review-of-review

- Final diff Ponytail review: `Lean already. Ship.` No new dependency was added.
- `go test -count=15 ./...`: passed.
- `go test -race -count=3 ./...`: passed.
- `go vet ./...`: passed.
- `staticcheck ./...`: passed.
- `govulncheck ./...`: no vulnerabilities found.
- `go mod verify` and `go mod tidy -diff`: clean.
- amd64, arm64 and 386 review builds: passed; temporary artifacts removed.
