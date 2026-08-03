## What changed

Describe the root cause and the smallest complete fix.

## Evidence

- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] Windows behavior checked when Windows code changed
- [ ] Linux behavior checked when Linux code changed
- [ ] New mutation has capability detection, backup, read-back and rollback
- [ ] User-visible change is in `docs/CHANGELOG.md`
- [ ] Tweak decision is in `docs/NOTES.md`
- [ ] No generated artifacts, secrets or local agent files are included
