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

  it('supports explicit prefix invalidation', async () => {
    const backend = new FakeBackend()
    const client = new CachedBackendClient(backend)
    await client.call('services.list', { state: 'all' })
    client.invalidate('services.')
    await client.call('services.list', { state: 'all' })
    expect(backend.calls).toBe(2)
  })
})
