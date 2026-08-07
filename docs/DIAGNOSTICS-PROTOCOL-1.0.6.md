# Diagnostics protocol review for 1.0.6

Luxury Optimization 1.0.6 adds measurement tools, not optimization mutations. All requests are strict JSON, cancellable through `system.cancel`, bounded in time and output, and available through the same Rust/Go method allowlists as the existing TCP test.

## UDP DNS round-trip

Method: `network.udp`

```json
{"address":"1.1.1.1:53","count":10,"timeout_ms":2000}
```

- Sends a fixed `example.com A IN` query encoded according to RFC 1035.
- Requires an explicit IP address and port, 3-50 attempts and a 100-5000 ms timeout.
- Accepts only a connected UDP response with the matching transaction ID, response bit, non-zero question count and successful DNS response code.
- Reports successful/failed attempts, ordered median/p95/min/max, sample-order jitter and the raw bounded samples.
- Measures the selected DNS/UDP path. It is not presented as generic game UDP latency.

## Loaded latency / bufferbloat

Method: `network.bufferbloat`

```json
{"probe_address":"1.1.1.1:443","duration_ms":3000,"streams":1}
```

Protocol:

1. Five idle TCP handshakes establish the baseline; at least three must succeed.
2. One bounded HTTPS download phase runs while TCP handshakes sample loaded latency.
3. One bounded HTTPS upload phase repeats the same probe.
4. Each phase reports bytes, approximate throughput, median/p95 loaded latency and p95 increase over idle.

The backend owns the only permitted endpoints: Cloudflare `speed.cloudflare.com/__down` and `speed.cloudflare.com/__up`. The WebView cannot submit arbitrary URLs, credentials or redirect hosts. Redirects remain HTTPS-only, on the same approved host, with a three-hop limit; ambient HTTP proxy variables are not used for the load phase.

Limits are 2-15 seconds, 1-4 streams, 64 latency samples and 128 MiB per direction. Only one heavy network/storage diagnostic can run at once. Download and upload availability are independent: an unsupported endpoint becomes a phase warning instead of invalidating the other measured phase.

The p95 increase labels are deliberately direct and documented:

| Loaded p95 increase | Label |
|---:|---|
| up to 5 ms | low |
| over 5 to 20 ms | moderate |
| over 20 to 50 ms | high |
| over 50 ms | severe |

These labels diagnose queueing under the generated HTTPS load. They do not prove ISP routing quality or prescribe a network tweak.

## Filesystem path probe

Methods: `storage.volumes`, `storage.test`

```json
{"path":"C:\\Games","size_mb":64,"block_kb":1024}
```

- Native Windows volume APIs and Linux mount/statfs data report filesystem, local kind, read-only state, total bytes and caller-available bytes.
- Remote, optical and pseudo filesystems are skipped from local gaming-storage inventory.
- The path must be an existing resolved directory on a writable local volume.
- The probe requires the requested size plus 256 MiB free-space headroom and accepts only 8-256 MiB with 64-4096 KiB power-of-two blocks.
- One uniquely named temporary file is written with a deterministic incompressible pattern, synchronized, read back, SHA-256 verified, closed and removed.
- Cancellation and every error path attempt the same cleanup; a successful result explicitly confirms removal.

Write plus sync is the durable-path metric. Immediate read is explicitly labelled buffered because operating-system cache cannot be portably evicted without raw or platform-specific disk access. Compare only repeated runs using the same path, size, power state and thermal state.

## Standalone space analyzer

Methods: `storage.scan.start`, `storage.scan.status`, `storage.scan.cancel`

The analyzer opens as a separate Tauri window and uses the existing shared sidecar, so scanning does not replace or block navigation in the main window. The initial request accepts only a listed local-volume root. Deeper navigation submits the parent scan ID plus an opaque directory node ID; raw deletion paths are never accepted.

Enumeration uses a bounded worker pool for directory metadata and one coordinator for deterministic aggregation. It skips links/reparse points and nested mount boundaries, reports live file/directory/byte progress and stops at 10 million entries or 15 minutes. Output is capped at 256 direct children, 100 largest files, 64 extension groups and 32 warnings.

Completed reports for up to 24 visited folders are cached for five minutes. Back/forward navigation can therefore reuse the exact bounded snapshot without touching the drive again. `refresh_scan_id` always bypasses the cache; deletion invalidates every related ancestor/descendant cache entry before the current folder is refreshed.

One observed non-controlled verification pass on the release machine enumerated 1,742,101 files and 188,338 directories on `E:\` in 23.816 seconds. This is execution evidence for that machine, not a universal disk-speed claim.

## Confirmed Recycle Bin / Trash deletion

Methods: `storage.delete.preview`, `storage.delete.confirm`

1. `preview` accepts only the current completed scan ID and an opaque deletable node ID.
2. The backend revalidates descendant scope, regular-file/directory type, reparse state, file identity, size/mode/mtime and protected-path policy.
3. A single-use random confirmation token expires after 45 seconds; no more than 32 tokens may be pending.
4. `confirm` consumes the token, repeats validation and invokes the platform Recycle Bin/Trash primitive. No permanent-delete fallback is used.
5. The source path must be absent after the operation before success is returned.

Roots, parent and aggregate nodes, Windows/Linux system-managed paths, system attributes, the running application tree, symlinks/reparse points and changed targets never receive a deletion capability. Folders of at least 1 GiB or 1,000 contained entries additionally require typing the exact scanned name in the UI.

## Deliberate exclusions

- No arbitrary WebView URLs, raw sockets, packet capture or raw-disk handles.
- No ICMP elevation, third-party speed-test executable or mutable runtime download.
- No automatic MTU, DNS, offload, scheduler, filesystem or drive-policy mutation.
- No raw-path delete command or silent permanent-delete fallback.
- No claim that one short diagnostic predicts game FPS; results are evidence for further investigation.
