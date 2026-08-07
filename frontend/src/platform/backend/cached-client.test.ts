import { describe, expect, it } from 'vitest'
import type { BackendClient } from './client'
import { CachedBackendClient } from './cached-client'

class FakeBackend implements BackendClient {
  calls = 0
  async call<T>(): Promise<T> { this.calls += 1; return { call: this.calls } as T }
  invalidate(): void {}
  async stop(): Promise<void> {}
}

describe('CachedBackendClient', () => {
  it('deduplicates read calls and expires them', async () => {
    const backend = new FakeBackend()
    let now = 100
    const client = new CachedBackendClient(backend, () => now)
    const [first, second] = await Promise.all([
      client.call<{ call: number }>('optimization.audit', {}),
      client.call<{ call: number }>('optimization.audit', {}),
    ])
    expect(first.call).toBe(1)
    expect(second.call).toBe(1)
    expect(backend.calls).toBe(1)
    now += 15_001
    expect((await client.call<{ call: number }>('optimization.audit', {})).call).toBe(2)
  })

  it('invalidates dependent reads after mutation', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('optimization.audit', {})
    await client.call('optimization.apply_tweak', { id: 'game-mode-enable' })
    await client.call('optimization.audit', {})
    expect(backend.calls).toBe(3)
  })

  it('refreshes per-game history after a launch', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('gaming.history', { id: '0123456789ab' })
    await client.call('gaming.history', { id: '0123456789ab' })
    await client.call('gaming.launch', { id: '0123456789ab' })
    await client.call('gaming.history', { id: '0123456789ab' })
    expect(backend.calls).toBe(3)
  })

  it('never caches live background measurements', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('advisor.background', { sample_ms: 1500 })
    await client.call('advisor.background', { sample_ms: 1500 })
    expect(backend.calls).toBe(2)
  })

  it('caches volume inventory but never loaded diagnostics', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('storage.volumes', {})
    await client.call('storage.volumes', {})
    await client.call('network.bufferbloat', { duration_ms: 2000 })
    await client.call('network.bufferbloat', { duration_ms: 2000 })
    await client.call('storage.test', { path: 'C:\\', size_mb: 8, block_kb: 64 })
    await client.call('storage.test', { path: 'C:\\', size_mb: 8, block_kb: 64 })
    await client.call('storage.scan.status', { scan_id: 'x' })
    await client.call('storage.scan.status', { scan_id: 'x' })
    expect(backend.calls).toBe(7)
  })

  it('supports explicit prefix invalidation', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('services.list', { state: 'all' })
    client.invalidate('services.')
    await client.call('services.list', { state: 'all' })
    expect(backend.calls).toBe(2)
  })
})
