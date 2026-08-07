import type { BackendClient } from './client'

interface CacheEntry {
  expiresAt: number
  value?: unknown
  pending?: Promise<unknown>
}

const ttlByMethod: Readonly<Record<string, number>> = {
  'system.handshake': Number.POSITIVE_INFINITY,
  'optimization.audit': 15_000,
  'optimization.plan': 30_000,
  'optimization.checkpoint_status': 30_000,
  'startup.list': 20_000,
  'services.list': 60_000,
  'network.interfaces': 60_000,
  'gaming.saved': 20_000,
  'gaming.history': 10_000,
  'backups.list': 20_000,
  'restore.system_points': 60_000,
  'updates.status': 30_000,
}

const invalidationByMutation: Readonly<Record<string, readonly string[]>> = {
  'optimization.apply': ['optimization.', 'backups.'],
  'optimization.apply_tweak': ['optimization.', 'backups.'],
  'optimization.restore_tweak': ['optimization.', 'backups.'],
  'optimization.create_checkpoint': ['optimization.checkpoint_status', 'backups.'],
  'optimization.restore': ['optimization.', 'backups.'],
  'restore.open_system': ['restore.system_points'],
  'startup.set': ['startup.list', 'optimization.audit'],
  'services.set': ['services.list'],
  'gaming.save': ['gaming.saved', 'gaming.history'],
  'gaming.remove': ['gaming.saved', 'gaming.history'],
  'gaming.launch': ['gaming.history'],
  'gaming.attach_benchmark': ['gaming.history'],
  'updates.check': ['updates.status'],
  'updates.install': ['updates.'],
}

export class CachedBackendClient implements BackendClient {
  #cache = new Map<string, CacheEntry>()

  constructor(private readonly backend: BackendClient, private readonly now: () => number = Date.now) {}

  async call<T>(method: string, payload?: unknown, signal?: AbortSignal): Promise<T> {
    const ttl = ttlByMethod[method]
    if (ttl === undefined) {
      const result = await this.backend.call<T>(method, payload, signal)
      for (const prefix of invalidationByMutation[method] ?? []) this.invalidate(prefix)
      return result
    }

    const key = cacheKey(method, payload)
    const cached = this.#cache.get(key)
    if (cached?.pending) return abortable(cached.pending as Promise<T>, signal)
    if (cached && cached.expiresAt > this.now() && 'value' in cached) return abortable(Promise.resolve(cached.value as T), signal)

    const pending = this.backend.call<T>(method, payload).then((value) => {
      this.#cache.set(key, { value, expiresAt: ttl === Number.POSITIVE_INFINITY ? ttl : this.now() + ttl })
      return value
    }).catch((error: unknown) => {
      this.#cache.delete(key)
      throw error
    })
    this.#cache.set(key, { pending, expiresAt: 0 })
    return abortable(pending, signal)
  }

  invalidate(methodPrefix = ''): void {
    for (const key of this.#cache.keys()) {
      if (key.startsWith(methodPrefix)) this.#cache.delete(key)
    }
  }

  async stop(): Promise<void> {
    this.#cache.clear()
    await this.backend.stop()
  }
}

function cacheKey(method: string, payload: unknown): string {
  return `${method}\0${JSON.stringify(payload ?? null)}`
}

function abortable<T>(promise: Promise<T>, signal?: AbortSignal): Promise<T> {
  if (!signal) return promise
  if (signal.aborted) return Promise.reject(new DOMException('Backend request aborted.', 'AbortError'))
  return new Promise<T>((resolve, reject) => {
    const abort = () => reject(new DOMException('Backend request aborted.', 'AbortError'))
    signal.addEventListener('abort', abort, { once: true })
    promise.then(resolve, reject).finally(() => signal.removeEventListener('abort', abort))
  })
}
